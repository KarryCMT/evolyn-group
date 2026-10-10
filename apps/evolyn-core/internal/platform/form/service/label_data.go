package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// LabelRecord 是表单域向标签域提供的权限裁剪后只读视图。
type LabelRecord struct {
	FormID uint
	Fields map[string]any
	System map[string]any
}

// LabelRecordSource 隔离标签域与表单物理表/JSONB 存储实现。
type LabelRecordSource interface {
	ReadLabelRecord(ctx context.Context, member *iammodel.User, formID, recordID uint) (*LabelRecord, error)
	ReadLabelRecordForOperation(ctx context.Context, member *iammodel.User, formID, recordID uint, operation string) (*LabelRecord, error)
	ReadLabelRecordsForOperation(ctx context.Context, member *iammodel.User, formID uint, recordIDs []uint, operation string) ([]*LabelRecord, error)
}

// LabelOperationSource 为无具体记录的运行态入口提供粗粒度操作可用性；
// 具体记录仍必须调用 ReadLabelRecordForOperation 复核数据范围。
type LabelOperationSource interface {
	CanUseLabelOperation(ctx context.Context, member *iammodel.User, formID uint, operation string) (bool, error)
}

func (s *formService) CanUseLabelOperation(ctx context.Context, member *iammodel.User, formID uint, operation string) (bool, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return false, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return false, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if operation != model.PermissionOpBatchPrint || !s.access.Permissions(ctx, member)["form-records:get"] {
		return false, nil
	}
	resolved, err := s.evaluatePermissions(ctx, member, formID)
	if err != nil {
		return false, err
	}
	if resolved == nil || resolved.Admin || resolved.Baseline {
		return true, nil
	}
	for index := range resolved.Matched {
		if resolved.Matched[index].Operations[operation] {
			return true, nil
		}
	}
	return false, nil
}

// ReadLabelRecord 复用表单记录读取、权限组范围和字段矩阵，输出同时以不可变
// fieldId 和历史 widgetName 建索引的数据视图；标签域不接触动态物理表。
func (s *formService) ReadLabelRecord(ctx context.Context, member *iammodel.User, formID, recordID uint) (*LabelRecord, error) {
	return s.ReadLabelRecordForOperation(ctx, member, formID, recordID, model.PermissionOpView)
}

// ReadLabelRecordForOperation 让标签运行态明确携带消费目的。batch_print 与
// view 共用同一条记录范围/字段矩阵管线，但操作不能在标签域内降级替换。
func (s *formService) ReadLabelRecordForOperation(ctx context.Context, member *iammodel.User, formID, recordID uint, operation string) (*LabelRecord, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot view form records"))
	}
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	if record.FormID != formID {
		return nil, gorm.ErrRecordNotFound
	}
	_, versionID, values, err := s.RecordData(ctx, recordID)
	if err != nil {
		return nil, err
	}
	resolved, err := s.evaluatePermissions(ctx, member, formID)
	if err != nil {
		return nil, err
	}
	if operation != model.PermissionOpView && operation != model.PermissionOpBatchPrint {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("unsupported label record operation %s", operation))
	}
	if resolved != nil && !resolved.AllowsOperation(operation, values) {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("record %d is outside %s scope", recordID, operation))
	}
	version, err := s.versions.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
		return nil, err
	}
	visible := map[string]FieldPermission(nil)
	if resolved != nil {
		visible = resolved.FieldsFor(operation, values)
	}
	fields := make(map[string]any, len(mappings)*2)
	for _, mapping := range mappings {
		if visible != nil && !visible[mapping.WidgetName].Visible {
			continue
		}
		value := values[mapping.WidgetName]
		fields[mapping.WidgetName] = value
		if mapping.FieldID != "" {
			fields[mapping.FieldID] = value
		}
	}
	china := time.FixedZone("Asia/Shanghai", 8*60*60)
	return &LabelRecord{
		FormID: formID,
		Fields: fields,
		System: map[string]any{
			"recordId":    strconv.FormatUint(uint64(record.ID), 10),
			"createdAt":   record.SubmittedAt.String(),
			"updatedAt":   record.UpdatedAt.String(),
			"createdBy":   record.SubmittedByName,
			"currentUser": member.Nickname,
			"currentDate": time.Now().In(china).Format("2006-01-02"),
		},
	}, nil
}

// ReadLabelRecordsForOperation 把 ID 集合与权限组数据范围合并进同一个受控
// 查询，避免批量任务创建阶段按记录逐条读取。返回顺序严格跟随 recordIDs；
// 任一 ID 不存在或越权时整批拒绝，防止用返回差集探测记录是否存在。
func (s *formService) ReadLabelRecordsForOperation(ctx context.Context, member *iammodel.User, formID uint, recordIDs []uint, operation string) ([]*LabelRecord, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if len(recordIDs) == 0 || !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot batch read form records"))
	}
	if operation != model.PermissionOpView && operation != model.PermissionOpBatchPrint {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("unsupported label record operation %s", operation))
	}
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %d is not published", formID))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return nil, err
	}
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
		return nil, err
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("published schema decode: %w", err)
	}
	fieldList, err := buildPermissionFieldList(content)
	if err != nil {
		return nil, err
	}
	opts, listBinding, err := s.physicalListBinding(ctx, form)
	if err != nil {
		return nil, err
	}
	resolved, err := s.evaluatePermissions(ctx, member, formID)
	if err != nil {
		return nil, err
	}

	args := make([]any, len(recordIDs))
	for index, id := range recordIDs {
		args[index] = id
	}
	idColumn := "id"
	if listBinding != nil {
		idColumn = "r.id"
	}
	predicates := []CompiledRecordQuery{{
		Where: fmt.Sprintf("%s IN (%s)", idColumn, strings.TrimSuffix(strings.Repeat("?,", len(recordIDs)), ",")),
		Args:  args,
	}}
	if resolved != nil && !resolved.Admin && !resolved.Baseline {
		scopes := make([]CompiledRecordQuery, 0, len(resolved.Matched))
		for _, group := range resolved.Matched {
			if !group.Operations[operation] {
				continue
			}
			scope, compileErr := CompilePermissionScopeSQL(group.DataScope, mappings, fieldList, opts)
			if compileErr != nil {
				return nil, fmt.Errorf("compile %s scope for group %s: %w", operation, group.Code, compileErr)
			}
			scopes = append(scopes, scope)
		}
		if len(scopes) == 0 {
			return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot %s records", operation))
		}
		predicates = append(predicates, joinCompiled(scopes, " OR "))
	}
	predicate := joinCompiled(predicates, " AND ")
	params := repository.RecordListParams{
		TenantID: tenantID, FormID: formID, Page: 1, PageSize: len(recordIDs),
		Where: predicate.Where, Args: predicate.Args,
	}
	var rows []envelopeRow
	if listBinding != nil {
		err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
			physicalRows, _, listErr := s.physical.ListJoinControlled(tctx, params, *listBinding)
			if listErr != nil {
				return listErr
			}
			if listErr = s.hydratePhysicalListChildren(tctx, physicalRows, listBinding.Children); listErr != nil {
				return listErr
			}
			rows = rowsOfPhysical(physicalRows)
			return nil
		})
	} else {
		var records []model.FormRecord
		records, _, err = s.records.ListControlled(ctx, params)
		rows = rowsOfLegacy(records)
	}
	if err != nil {
		return nil, err
	}
	if len(rows) != len(recordIDs) {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("one or more records are missing or outside %s scope", operation))
	}
	byID := make(map[uint]envelopeRow, len(rows))
	for _, row := range rows {
		byID[row.record.ID] = row
	}
	result := make([]*LabelRecord, 0, len(recordIDs))
	for _, id := range recordIDs {
		row, exists := byID[id]
		if !exists {
			return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("record %d is unavailable", id))
		}
		result = append(result, labelRecordFromEnvelope(member, formID, row, mappings, resolved, operation))
	}
	return result, nil
}

func labelRecordFromEnvelope(member *iammodel.User, formID uint, row envelopeRow, mappings []SnapshotFieldMapping, resolved *ResolvedFormPermission, operation string) *LabelRecord {
	visible := map[string]FieldPermission(nil)
	if resolved != nil {
		visible = resolved.FieldsFor(operation, row.values)
	}
	fields := make(map[string]any, len(mappings)*2)
	for _, mapping := range mappings {
		if visible != nil && !visible[mapping.WidgetName].Visible {
			continue
		}
		value := row.values[mapping.WidgetName]
		fields[mapping.WidgetName] = value
		if mapping.FieldID != "" {
			fields[mapping.FieldID] = value
		}
	}
	china := time.FixedZone("Asia/Shanghai", 8*60*60)
	return &LabelRecord{
		FormID: formID,
		Fields: fields,
		System: map[string]any{
			"recordId": strconv.FormatUint(uint64(row.record.ID), 10), "createdAt": row.record.SubmittedAt.String(),
			"updatedAt": row.record.UpdatedAt.String(), "createdBy": row.record.SubmittedByName,
			"currentUser": member.Nickname, "currentDate": time.Now().In(china).Format("2006-01-02"),
		},
	}
}
