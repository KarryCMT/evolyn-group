package service

import (
	"context"
	"errors"

	"evolyn/internal/contextx"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/tenantproduct/model"
	"evolyn/internal/platform/tenantproduct/repository"

	"gorm.io/gorm"
)

// accessEvaluator 产品访问判定器（文档 6.4）：只依赖仓储的只读能力，
// 与管理面服务解耦——受保护入口不应因管理面依赖（edition/审计）故障
// 而误判。判定不写库、不开事务，可高频调用
type accessEvaluator struct {
	repo repository.Repository
}

// NewTenantProductAccessEvaluator 构造产品访问判定器：产品真实入口、
// 工作台路由的后端数据接口或后续跨服务网关注入使用
func NewTenantProductAccessEvaluator(repo repository.Repository) TenantProductAccessEvaluator {
	return &accessEvaluator{repo: repo}
}

// CanAccess 判定当前成员能否使用指定产品。租户取 ctx 携带的租户上下文
// （与鉴权中间件同源）；memberID 必须是该租户的有效成员。
// 返回 (false, nil) 表示任一判定步骤不满足；error 仅表示基础设施故障
func (a *accessEvaluator) CanAccess(ctx context.Context, productCode string, memberID uint) (bool, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 || memberID == 0 {
		return false, nil
	}

	// 1. 平台目录 active：平台停用后所有租户不可访问
	catalog, err := a.repo.GetCatalogByCode(ctx, productCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if catalog.Status != model.CatalogStatusActive {
		return false, nil
	}

	// 2. 租户产品 enabled：未初始化（目录先于回填到达）视同不可用
	config, err := a.repo.GetConfig(ctx, tenantID, catalog.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !config.Enabled {
		return false, nil
	}

	// 3. 成员为当前租户有效成员：跨租户/离职/禁用成员一律拒绝
	member, err := a.repo.GetMember(ctx, tenantID, memberID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if member.Status != iammodel.MemberStatusActive {
		return false, nil
	}

	// 4. 范围命中：all 直接放行；partial 命中直接成员或选中部门（含子部门）
	if config.ScopeMode == model.ScopeModeAll {
		return true, nil
	}
	memberIDs, err := a.repo.ListScopeMembers(ctx, config.ID)
	if err != nil {
		return false, err
	}
	if containsUint(memberIDs, memberID) {
		return true, nil
	}
	departmentIDs, err := a.repo.ListScopeDepartments(ctx, config.ID)
	if err != nil {
		return false, err
	}
	if len(departmentIDs) == 0 {
		return false, nil
	}
	departments, err := a.repo.ListTenantDepartments(ctx, tenantID)
	if err != nil {
		return false, err
	}
	// 展开集只含有效部门：停用/删除的选中部门及其子树不授予访问（文档 5.5）
	expanded := expandActiveDescendants(departmentIDs, departments)
	if len(expanded) == 0 {
		return false, nil
	}
	return a.repo.MemberInDepartments(ctx, tenantID, memberID, expanded)
}
