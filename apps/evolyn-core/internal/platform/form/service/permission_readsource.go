// 权限组只读查询端口实现（表单权限 P1）：formService 执行点（switch-type
// 阻塞、发布阻塞）所需的窄查询面，实现为权限组仓储的投影适配——接口化便于
// 单测桩替换，避免 formService 直依赖完整仓储面。
package service

import (
	"context"

	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
)

// PermissionGroupReadSource formService 执行点所需权限组只读查询窄端口。
type PermissionGroupReadSource interface {
	// HasWorkflowOperations 表单是否存在任一权限组（含禁用组，不含软删）的
	// operations 配置了流程专属操作键（§3.3 类型切换阻塞）
	HasWorkflowOperations(ctx context.Context, formID uint) (bool, error)
	// EnabledDataScopeFields 表单全部启用权限组 data_scope 引用的字段键集合
	//（§5.2 发布阻塞判定）
	EnabledDataScopeFields(ctx context.Context, formID uint) (map[string]bool, error)
}

// permissionGroupReadSource 端口实现：经权限组仓储查询，业务判定在域内。
type permissionGroupReadSource struct {
	groups repository.PermissionGroupRepository
}

// NewPermissionGroupReadSource 构造只读查询端口（server 装配注入）。
func NewPermissionGroupReadSource(groups repository.PermissionGroupRepository) PermissionGroupReadSource {
	return &permissionGroupReadSource{groups: groups}
}

func (s *permissionGroupReadSource) HasWorkflowOperations(ctx context.Context, formID uint) (bool, error) {
	groups, err := s.groups.ListByAsset(ctx, model.PermissionAssetTypeForm, formID)
	if err != nil {
		return false, err
	}
	for i := range groups {
		for _, op := range groups[i].Operations {
			if model.IsWorkflowPermissionOperation(op) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *permissionGroupReadSource) EnabledDataScopeFields(ctx context.Context, formID uint) (map[string]bool, error) {
	groups, err := s.groups.ListEnabledByAssetIDs(ctx, model.PermissionAssetTypeForm, []uint{formID})
	if err != nil {
		return nil, err
	}
	fields := make(map[string]bool)
	for i := range groups {
		scope := model.PermissionDataScopeSpec(groups[i].DataScope)
		for _, condition := range scope.Conditions {
			fields[condition.Field] = true
		}
	}
	return fields, nil
}
