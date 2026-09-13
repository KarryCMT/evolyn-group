package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

const maxDeleteRecordsPerRequest = 100

// DeleteRecords 永久删除数据管理中选中的记录。删除路径和列表路径一样以发布
// 快照/物理存储作为值事实源：先逐条读取值并判定 delete 数据范围，再按「子表
// → 物理父表 → 记录信封」顺序清理，避免外键残留或跨表单删除。
func (s *formService) DeleteRecords(
	ctx context.Context,
	member *iammodel.User,
	code string,
	req *model.DeleteFormRecordsRequest,
) (*model.DeleteFormRecordsResult, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if !s.access.Permissions(ctx, member)["form-records:delete"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot delete form records"))
	}
	if req == nil {
		return nil, apperrors.ErrRecordDeleteInvalid
	}
	ids, err := normalizeDeleteRecordIDs(req.RecordIDs)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordDeleteInvalid, err)
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if s.records == nil {
		return nil, fmt.Errorf("form record repository is not configured")
	}
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}

	var deleted int64
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		for _, id := range ids {
			record, getErr := s.records.GetByID(tctx, id)
			if getErr != nil {
				if getErr == gorm.ErrRecordNotFound {
					return httpx.Wrap(apperrors.ErrRecordDeleteInvalid, fmt.Errorf("record %d not found", id))
				}
				return getErr
			}
			if record.FormID != form.ID {
				return httpx.Wrap(apperrors.ErrRecordDeleteInvalid, fmt.Errorf("record %d is not in form %s", id, form.Code))
			}
			if record.WorkflowInstanceNo != "" {
				return httpx.Wrap(apperrors.ErrRecordWorkflowActive, fmt.Errorf("record %d is bound to workflow %s", id, record.WorkflowInstanceNo))
			}
			values, valueErr := s.recordValues(tctx, record, form)
			if valueErr != nil {
				return valueErr
			}
			if resolved != nil && !resolved.AllowsOperation(model.PermissionOpDelete, values) {
				return httpx.Wrap(apperrors.ErrPermissionDenied, fmt.Errorf("member cannot delete record %d", id))
			}
		}

		if err := s.deletePhysicalRecordValues(tctx, form, tenantID, ids); err != nil {
			return err
		}
		deleted, err = deleteRecordEnvelopes(tctx, s.records, form.ID, ids)
		if err != nil {
			return err
		}
		if deleted != int64(len(ids)) {
			return httpx.Wrap(apperrors.ErrRecordDeleteInvalid, fmt.Errorf("expected %d records deleted, got %d", len(ids), deleted))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		appID, appCode, appName := s.appSnapshot(ctx, form.ApplicationID)
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "delete", ResourceType: "form_record",
			ResourceID: strconv.FormatUint(uint64(ids[0]), 10),
			After:      map[string]any{"formCode": form.Code, "recordIds": ids, "deletedCount": deleted},
			TargetName: form.Name, ApplicationID: appID, ApplicationCode: appCode, ApplicationName: appName,
		})
	}
	return &model.DeleteFormRecordsResult{DeletedCount: int(deleted)}, nil
}

// deleteRecordEnvelopes 保持原有 FormRecordRepository 兼容性：物理存储落地
// 前的测试桩不需要承担删除能力；生产仓储通过可选窄接口提供原子硬删除。
func deleteRecordEnvelopes(ctx context.Context, records any, formID uint, ids []uint) (int64, error) {
	type deleter interface {
		DeleteByIDs(context.Context, uint, []uint) (int64, error)
	}
	store, ok := records.(deleter)
	if !ok {
		return 0, fmt.Errorf("form record deletion is not configured")
	}
	return store.DeleteByIDs(ctx, formID, ids)
}

func normalizeDeleteRecordIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 || len(ids) > maxDeleteRecordsPerRequest {
		return nil, fmt.Errorf("record count must be between 1 and %d", maxDeleteRecordsPerRequest)
	}
	seen := make(map[uint]struct{}, len(ids))
	normalized := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, fmt.Errorf("record id is required")
		}
		if _, duplicated := seen[id]; duplicated {
			return nil, fmt.Errorf("duplicate record id %d", id)
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

// recordValues 把 legacy JSONB 和新物理存储统一为 delete 范围判定需要的值视图。
func (s *formService) recordValues(ctx context.Context, record *model.FormRecord, form *model.Form) (map[string]any, error) {
	if len(record.Values) > 0 && string(record.Values) != "null" {
		values := make(map[string]any)
		if err := json.Unmarshal(record.Values, &values); err != nil {
			return nil, fmt.Errorf("record %d values decode: %w", record.ID, err)
		}
		return values, nil
	}
	physical, err := s.resolvePhysicalContext(ctx, form)
	if err != nil {
		return nil, err
	}
	if physical == nil {
		return map[string]any{}, nil
	}
	values, err := s.readPhysicalValues(ctx, physical, record.ID)
	if err != nil {
		return nil, err
	}
	if values == nil {
		return nil, httpx.Wrap(apperrors.ErrRecordDeleteInvalid, fmt.Errorf("physical values missing for record %d", record.ID))
	}
	return values, nil
}

func (s *formService) deletePhysicalRecordValues(ctx context.Context, form *model.Form, tenantID uint, recordIDs []uint) error {
	if s.storages == nil {
		return nil // legacy JSONB form
	}
	binding, err := s.loadStorageByForm(ctx, form.ID)
	if err != nil {
		return err
	}
	if binding == nil {
		return nil
	}
	if s.physical == nil || s.storageChildren == nil {
		return fmt.Errorf("physical storage deletion pipeline is not configured")
	}
	children, err := s.storageChildren.ListByStorage(ctx, binding.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := s.physical.DeleteChildRows(ctx, child.PhysicalTable, tenantID, recordIDs); err != nil {
			return err
		}
	}
	return s.physical.DeleteParentRows(ctx, binding.PhysicalTable, tenantID, recordIDs)
}
