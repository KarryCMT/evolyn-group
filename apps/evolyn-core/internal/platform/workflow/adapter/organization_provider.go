// 组织窄端口适配器（ADR-012 Phase 3）：把 IAM 角色/部门数据桥接为引擎
// provider.OrganizationProvider。只暴露 IAM 真实具备的组织语义；IAM 未
// 支持的能力不在本端口出现「猜测实现」（第 17 章原则）。
//
// 租户隔离口径同 identity_provider.go：Model(结构体) 路径走租户 Callback，
// JOIN 聚合路径显式携带 tenant_id 条件。
package adapter

import (
	"context"
	"errors"

	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/infrastructure"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// OrganizationProvider 引擎组织窄端口的 IAM 适配。
type OrganizationProvider struct {
	db       *gorm.DB
	identity *IdentityProvider
}

// NewOrganizationProvider 构造组织适配器（显示名快照复用身份适配器口径）。
func NewOrganizationProvider(base *gorm.DB) *OrganizationProvider {
	return &OrganizationProvider{db: base, identity: NewIdentityProvider(base)}
}

// ResolveRoleMembers 解析指定角色的在任成员。roleCode 语义 = 角色名称
// （租户内唯一，与 IAM Role 现行模型一致，无独立 code 列）。
func (p *OrganizationProvider) ResolveRoleMembers(ctx context.Context, tenantID uint, roleCode string) ([]model.Actor, error) {
	rows := make([]iammodel.User, 0)
	if err := infrastructure.ResolveDB(ctx, p.db).
		Model(&iammodel.User{}).
		Joins("JOIN tn_user_roles ON tn_user_roles.user_id = tn_users.id").
		Joins("JOIN tn_roles ON tn_roles.id = tn_user_roles.role_id").
		Where("tn_roles.name = ?", roleCode).
		Where("tn_users.status = ?", iammodel.MemberStatusActive).
		Where("tn_users.resigned_at IS NULL").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return p.toActors(ctx, tenantID, rows), nil
}

// ResolveDepartmentMembers 解析部门（不含子部门，V1 直属口径）的在任成员。
func (p *OrganizationProvider) ResolveDepartmentMembers(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error) {
	rows := make([]iammodel.User, 0)
	if err := infrastructure.ResolveDB(ctx, p.db).
		Model(&iammodel.User{}).
		Joins("JOIN tn_department_users ON tn_department_users.user_id = tn_users.id").
		Where("tn_department_users.department_id = ?", deptID).
		Where("tn_users.status = ?", iammodel.MemberStatusActive).
		Where("tn_users.resigned_at IS NULL").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return p.toActors(ctx, tenantID, rows), nil
}

// ResolveDepartmentManager 解析部门负责人（迁移 000050 leader 前置能力）。
// 未设置负责人或负责人不在任时返回空集——由 Resolver/执行器统一转
// WORKFLOW_ASSIGNEE_NOT_FOUND 并走租户管理员兜底。
func (p *OrganizationProvider) ResolveDepartmentManager(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error) {
	var row iammodel.Department
	if err := infrastructure.ResolveDB(ctx, p.db).
		Where("id = ?", deptID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []model.Actor{}, nil
		}
		return nil, err
	}
	if row.LeaderMemberID == nil {
		return []model.Actor{}, nil
	}
	// 负责人必须在任（含租户过滤：跨租户 ID 视为未设置）
	var leader iammodel.User
	if err := infrastructure.ResolveDB(ctx, p.db).
		Where("id = ?", *row.LeaderMemberID).
		Where("status = ?", iammodel.MemberStatusActive).
		Where("resigned_at IS NULL").First(&leader).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []model.Actor{}, nil
		}
		return nil, err
	}
	return []model.Actor{{
		MemberID:    leader.ID,
		DisplayName: p.identity.MemberDisplayName(ctx, tenantID, leader.ID),
	}}, nil
}

// ResolveStarterManager 发起人直属主管：IAM 现行模型无 reporting 汇报关系
// 语义（第 17.2 章前置约束），本端口显式返回不支持错误，禁止猜测实现；
// 对应 Resolver 能力矩阵保持关闭，发布校验器会先行拦截。
func (p *OrganizationProvider) ResolveStarterManager(ctx context.Context, tenantID, starterMemberID uint) ([]model.Actor, error) {
	return nil, errors.New("IAM 未提供直属主管（reporting）组织语义，starter_manager 暂不支持")
}

// MemberDisplayName 成员显示名快照（复用身份适配器实现）。
func (p *OrganizationProvider) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	return p.identity.MemberDisplayName(ctx, tenantID, memberID)
}

// toActors 查询结果 → 参与人快照（显示名即时固化）。
func (p *OrganizationProvider) toActors(ctx context.Context, tenantID uint, rows []iammodel.User) []model.Actor {
	actors := make([]model.Actor, 0, len(rows))
	for i := range rows {
		actors = append(actors, model.Actor{
			MemberID:    rows[i].ID,
			DisplayName: p.identity.MemberDisplayName(ctx, tenantID, rows[i].ID),
		})
	}
	return actors
}

// EnsureInterfaces 编译期端口契约自检。
var _ provider.OrganizationProvider = (*OrganizationProvider)(nil)
