package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- 发布（P2，后端契约 §2.2） ----

// Publish 发布：草稿口令复核 → 白名单校验 → 严格字典校验 → 事务内创建不可变快照
// 并回写 forms.latest_version_id/published_version。历史版本不被触碰。
func (s *formService) Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error) {
	if !s.access.Permissions(ctx, member)["forms:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot publish form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.DraftRevision != form.DraftRevision {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %s draft revision %d != %d on publish", code, req.DraftRevision, form.DraftRevision))
	}
	// 白名单先行：给出比结构错误更明确的能力提示（FORM_PUBLISH_UNSUPPORTED_FIELD）。
	if issues := ValidatePublishable([]byte(form.DraftContent), form.ProtocolVersion); len(issues) > 0 {
		return nil, httpx.Wrap(apperrors.ErrPublishUnsupportedField.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("form %s publish blocked: %s", code, issues[0].Path))
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
		nextNo, err := s.versions.MaxVersionNo(tctx, form.ID)
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
			FormID:              form.ID,
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
		if err := s.repo.MarkPublished(tctx, form.ID, created.ID, nextNo); err != nil {
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
			ResourceID: form.Code,
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
func (s *formService) GetRuntime(ctx context.Context, appCode, formCode string) (*model.FormRuntime, error) {
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

	form, err := s.loadByCode(ctx, formCode)
	if err != nil {
		return nil, err
	}
	if form.ApplicationID != app.ID {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
			fmt.Errorf("form %s not in application %s", formCode, appCode))
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", formCode))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotPublished, err)
		}
		return nil, err
	}
	return &model.FormRuntime{
		FormCode:         form.Code,
		Name:             form.Name,
		PublishedVersion: version.VersionNo,
		SchemaRevision:   strconv.FormatInt(version.SchemaRevision, 10),
		ProtocolVersion:  version.ProtocolVersion,
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

	form, err := s.loadByCode(ctx, req.FormCode)
	if err != nil {
		return nil, err
	}
	// 应用编码属于提交上下文的一部分：按编码加载并复核表单归属，禁止只凭
	// formCode 跨应用构造请求；归档应用停止受理提交（与 bootstrap 同口径）。
	appCode := strings.TrimSpace(req.AppCode)
	app, notFound, err := s.apps.ApplicationByCode(ctx, appCode)
	if err != nil {
		return nil, err
	}
	if notFound || app.ID != form.ApplicationID || app.Status != applicationStatusActive {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
			fmt.Errorf("application %s unavailable for form %s submit", appCode, req.FormCode))
	}
	entryCode := strings.TrimSpace(req.EntryCode)
	if entryCode != "" && s.references != nil {
		references, rerr := s.references.ListFormReferences(ctx, form.ID)
		if rerr != nil {
			return nil, rerr
		}
		matched := false
		for _, reference := range references {
			if reference.ApplicationCode == appCode && reference.EntryID == entryCode {
				matched = true
				break
			}
		}
		if !matched {
			return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
				fmt.Errorf("entry %s does not reference form %s in application %s", entryCode, req.FormCode, appCode))
		}
	}
	if req.HasResult == nil || !*req.HasResult {
		return nil, httpx.Wrap(apperrors.ErrRecordInvalid, fmt.Errorf("hasResult must be true"))
	}
	operationID, err := uuid.Parse(strings.TrimSpace(req.DataOpID))
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordInvalid, fmt.Errorf("invalid dataOpId: %w", err))
	}
	version, err := s.versions.GetByFormAndVersionNo(ctx, form.ID, req.PublishedVersion)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrVersionConflict,
				fmt.Errorf("form %s version %d not found", req.FormCode, req.PublishedVersion))
		}
		return nil, err
	}
	if strconv.FormatInt(version.SchemaRevision, 10) != req.SchemaRevision {
		return nil, httpx.Wrap(apperrors.ErrVersionConflict,
			fmt.Errorf("form %s version %d revision mismatch", req.FormCode, req.PublishedVersion))
	}

	content := make(map[string]any)
	if err := json.Unmarshal([]byte(version.Content), &content); err != nil {
		return nil, fmt.Errorf("snapshot decode: %w", err)
	}
	cleaned, fieldErrors := ValidateSubmittedRecordValues(content, req.Values)
	if len(fieldErrors) > 0 {
		return nil, httpx.Wrap(apperrors.ErrRecordInvalid.WithData(map[string]any{"fieldErrors": fieldErrors}),
			fmt.Errorf("form %s record invalid: %d field(s) rejected", req.FormCode, len(fieldErrors)))
	}

	valuesJSON, err := json.Marshal(cleaned)
	if err != nil {
		return nil, err
	}
	var (
		record  *model.FormRecord
		created bool
	)
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		canonicalOperationID := operationID.String()
		var entryCodeSnapshot *string
		if entryCode != "" {
			entryCodeSnapshot = &entryCode
		}
		draft := &model.FormRecord{
			FormID:              form.ID,
			FormVersionID:       version.ID,
			DataOpID:            &canonicalOperationID,
			EntryCode:           entryCodeSnapshot,
			Values:              model.JSONContent(valuesJSON),
			SubmittedByMemberID: member.ID,
			SubmittedAt:         kernel.JSONTime(time.Now()),
		}
		draft.TenantID = tenantID
		stored, wasCreated, cerr := s.records.CreateIdempotent(tctx, draft)
		if cerr != nil {
			return cerr
		}
		if !wasCreated && (stored.FormID != draft.FormID ||
			stored.FormVersionID != draft.FormVersionID ||
			stored.SubmittedByMemberID != draft.SubmittedByMemberID ||
			!sameJSON(stored.Values, draft.Values)) {
			return httpx.Wrap(apperrors.ErrRecordInvalid,
				fmt.Errorf("dataOpId %s reused by a different submission", canonicalOperationID))
		}
		record = stored
		created = wasCreated
		return nil
	}); err != nil {
		return nil, err
	}

	if created && s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "submit", ResourceType: "form_record",
			ResourceID: strconv.FormatUint(uint64(record.ID), 10),
			After: map[string]any{
				"formCode":         req.FormCode,
				"publishedVersion": req.PublishedVersion,
			},
		})
	}
	return &model.SubmitRecordResult{RecordID: record.ID}, nil
}

// sameJSON 以解码后的 JSON 值比较幂等重放内容，避免 PostgreSQL jsonb 规范化
// 键顺序后与客户端原始字节不同而误判为另一笔提交。
func sameJSON(left, right model.JSONContent) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
