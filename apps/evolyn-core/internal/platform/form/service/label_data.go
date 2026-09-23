package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
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
}

// ReadLabelRecord 复用表单记录读取、权限组范围和字段矩阵，输出同时以不可变
// fieldId 和历史 widgetName 建索引的数据视图；标签域不接触动态物理表。
func (s *formService) ReadLabelRecord(ctx context.Context, member *iammodel.User, formID, recordID uint) (*LabelRecord, error) {
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
	if resolved != nil && !resolved.AllowsOperation(model.PermissionOpView, values) {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("record %d is outside view scope", recordID))
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
		visible = resolved.FieldsFor(model.PermissionOpView, values)
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
