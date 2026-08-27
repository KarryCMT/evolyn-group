package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// ---- 发布（P2，后端契约 §2.2） ----

// Publish 发布：草稿口令复核 → 白名单校验 → 严格字典校验 → 事务内创建不可变快照
// 并回写 forms.latest_version_id/published_version。历史版本不被触碰。
func (s *formService) Publish(ctx context.Context, member *iammodel.User, id uint, req *model.PublishRequest) (*model.PublishResult, error) {
	if !s.access.Permissions(ctx, member)["forms:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot publish form %d", id))
	}
	form, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.DraftRevision != form.DraftRevision {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %d draft revision %d != %d on publish", id, req.DraftRevision, form.DraftRevision))
	}
	// 白名单先行：给出比结构错误更明确的能力提示（FORM_PUBLISH_UNSUPPORTED_FIELD）。
	if issues := ValidatePublishable([]byte(form.DraftContent)); len(issues) > 0 {
		return nil, httpx.Wrap(apperrors.ErrPublishUnsupportedField.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("form %d publish blocked: %s", id, issues[0].Path))
	}

	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member required"))
	}

	var result *model.PublishResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		// 发布号在事务内取 max+1；(form_id, version_no) 唯一约束兜底并发发布。
		nextNo, err := s.versions.MaxVersionNo(tctx, id)
		if err != nil {
			return err
		}
		nextNo++

		content := make(map[string]any)
		if err := json.Unmarshal([]byte(form.DraftContent), &content); err != nil {
			return err
		}
		fieldKeys, err := json.Marshal(ExtractSnapshotTopFieldKeys(content))
		if err != nil {
			return err
		}

		version := &model.FormVersion{
			FormID:              id,
			VersionNo:           nextNo,
			Content:             form.DraftContent,
			FieldKeys:           model.JSONContent(fieldKeys),
			ProtocolVersion:     form.ProtocolVersion,
			PublishedByMemberID: member.ID,
			PublishedAt:         kernel.JSONTime(time.Now()),
		}
		version.TenantID = tenantID
		created, err := s.versions.Create(tctx, version)
		if err != nil {
			return err
		}
		// 修订口令 = 版本行 id（创建事务内一次性回填，此后无更新路径）。
		if err := s.versions.SetSchemaRevision(tctx, created.ID, int64(created.ID)); err != nil {
			return err
		}
		// 回写资产行的最新发布指针（草稿不被覆盖）。
		if err := s.repo.MarkPublished(tctx, id, created.ID, nextNo); err != nil {
			return err
		}
		result = &model.PublishResult{
			PublishedVersion: nextNo,
			SchemaRevision:   strconv.FormatInt(int64(created.ID), 10),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "publish", ResourceType: "form",
			ResourceID: strconv.FormatUint(uint64(id), 10),
			After: map[string]any{
				"publishedVersion": result.PublishedVersion,
				"schemaRevision":   result.SchemaRevision,
			},
		})
	}
	return result, nil
}

// ---- 运行时 bootstrap（P2） ----

// GetRuntime 运行时 bootstrap：appCode 归属复核 + 应用 active + 表单已发布；
// 普通成员可读（路由经 applications:get，与菜单同口径），无 forms 管理权限要求。
func (s *formService) GetRuntime(ctx context.Context, appCode string, formID uint) (*model.FormRuntime, error) {
	app, notFound, err := s.apps.ApplicationByCode(ctx, appCode)
	if err != nil {
		return nil, err
	}
	if notFound {
		return nil, httpx.Wrap(apperrors.ErrFormNotFound, fmt.Errorf("application %s not found", appCode))
	}
	if app.Status != applicationStatusActive {
		return nil, httpx.Wrap(apperrors.ErrFormNotFound, fmt.Errorf("application %s status %s", appCode, app.Status))
	}

	form, err := s.load(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form.ApplicationID != app.ID {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
			fmt.Errorf("form %d not in application %s", formID, appCode))
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %d not published", formID))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotPublished, err)
		}
		return nil, err
	}
	return &model.FormRuntime{
		FormID:           form.ID,
		Name:             form.Name,
		PublishedVersion: version.VersionNo,
		SchemaRevision:   strconv.FormatInt(version.SchemaRevision, 10),
		Content:          version.Content,
	}, nil
}

// ---- 记录提交（P2） ----

// SubmitRecord 提交：按 (form_id, version_no) 定位版本（历史版本合法），复核修订口令；
// 逐字段按快照校验值（错误按 widgetName 回填），校验通过后落 form_records。
func (s *formService) SubmitRecord(ctx context.Context, member *iammodel.User, req *model.SubmitRecordRequest) (*model.SubmitRecordResult, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	// 提交资源与设计资源分离（form-records:create 授全体成员；表单管理面不受影响）
	if !s.access.Permissions(ctx, member)["form-records:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot submit form records"))
	}

	form, err := s.load(ctx, req.FormID)
	if err != nil {
		return nil, err
	}
	// 应用状态门控：归档应用的表单停止受理提交（与 bootstrap 同口径）。
	app, notFound, err := s.apps.ApplicationByID(ctx, form.ApplicationID)
	if err != nil {
		return nil, err
	}
	if notFound || app.Status != applicationStatusActive {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
			fmt.Errorf("application %d unavailable for submit", form.ApplicationID))
	}
	version, err := s.versions.GetByFormAndVersionNo(ctx, req.FormID, req.PublishedVersion)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrVersionConflict,
				fmt.Errorf("form %d version %d not found", req.FormID, req.PublishedVersion))
		}
		return nil, err
	}
	if strconv.FormatInt(version.SchemaRevision, 10) != req.SchemaRevision {
		return nil, httpx.Wrap(apperrors.ErrVersionConflict,
			fmt.Errorf("form %d version %d revision mismatch", req.FormID, req.PublishedVersion))
	}

	content := make(map[string]any)
	if err := json.Unmarshal([]byte(version.Content), &content); err != nil {
		return nil, fmt.Errorf("snapshot decode: %w", err)
	}
	rawValues := make(map[string]json.RawMessage, len(req.Values))
	for key, value := range req.Values {
		rawValues[key] = json.RawMessage(value)
	}
	cleaned, fieldErrors := ValidateRecordValues(content, rawValues)
	if len(fieldErrors) > 0 {
		return nil, httpx.Wrap(apperrors.ErrRecordInvalid.WithData(map[string]any{"fieldErrors": fieldErrors}),
			fmt.Errorf("form %d record invalid: %d field(s) rejected", req.FormID, len(fieldErrors)))
	}

	valuesJSON, err := json.Marshal(cleaned)
	if err != nil {
		return nil, err
	}
	var record *model.FormRecord
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		draft := &model.FormRecord{
			FormID:              req.FormID,
			FormVersionID:       version.ID,
			Values:              model.JSONContent(valuesJSON),
			SubmittedByMemberID: member.ID,
			SubmittedAt:         kernel.JSONTime(time.Now()),
		}
		draft.TenantID = tenantID
		created, cerr := s.records.Create(tctx, draft)
		if cerr != nil {
			return cerr
		}
		record = created
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "submit", ResourceType: "form_record",
			ResourceID: strconv.FormatUint(uint64(record.ID), 10),
			After: map[string]any{
				"formId":           req.FormID,
				"publishedVersion": req.PublishedVersion,
			},
		})
	}
	return &model.SubmitRecordResult{RecordID: record.ID}, nil
}
