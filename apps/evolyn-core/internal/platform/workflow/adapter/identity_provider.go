// 身份窄端口适配器（ADR-012 Phase 3）：把 IAM 成员数据桥接为引擎
// provider.IdentityProvider。只做只读查询与校验，不缓存组织事实——
// 审批人快照在任务创建事务内一次性固化（v1.1 定版）。
//
// 租户隔离：Model(&iammodel.User{}) 路径由 GORM 租户 Callback 自动过滤；
// Table()/JOIN 聚合路径 Schema 未解析，必须显式携带 tn_users.tenant_id 条件
// （与 enterpriseLog 域 JOIN 查询同口径）。
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

// IdentityProvider 引擎身份窄端口的 IAM 适配。
type IdentityProvider struct {
	db *gorm.DB
}

// NewIdentityProvider 构造身份适配器（base 连接经 ResolveDB 加入调用方事务）。
func NewIdentityProvider(base *gorm.DB) *IdentityProvider {
	return &IdentityProvider{db: base}
}

// activeMemberScope 在任成员条件（active 且未离职），配合 Model(users) 使用。
func activeMemberScope() string { return "tn_users.status = ? AND tn_users.resigned_at IS NULL" }

// ValidateMembers 校验成员集合同租户（ctx 租户过滤）且在任（active 未离职）。
func (p *IdentityProvider) ValidateMembers(ctx context.Context, tenantID uint, memberIDs []uint) error {
	if len(memberIDs) == 0 {
		return nil
	}
	var count int64
	if err := infrastructure.ResolveDB(ctx, p.db).Model(&iammodel.User{}).
		Where("id IN ?", memberIDs).
		Where(activeMemberScope(), iammodel.MemberStatusActive).
		Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(memberIDs)) {
		return errors.New("存在无效或不在任的成员")
	}
	return nil
}

// MemberDisplayName 成员显示名快照：成员昵称为空回落账号昵称/登录名。
// 聚合查询使用 IAM 模型的真实表名；Table 不会自动把旧 users 名称转换为 tn_users。
func (p *IdentityProvider) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	var row struct {
		MemberNickname  string
		AccountNickname string
		AccountName     string
	}
	if err := infrastructure.ResolveDB(ctx, p.db).
		Table((&iammodel.User{}).TableName()).
		Select("tn_users.nickname AS member_nickname, COALESCE(pf_accounts.nickname, '') AS account_nickname, COALESCE(pf_accounts.name, '') AS account_name").
		Joins("LEFT JOIN pf_accounts ON pf_accounts.id = tn_users.account_id").
		Where("tn_users.id = ? AND tn_users.tenant_id = ?", memberID, tenantID).
		Take(&row).Error; err != nil {
		return ""
	}
	switch {
	case row.MemberNickname != "":
		return row.MemberNickname
	case row.AccountNickname != "":
		return row.AccountNickname
	default:
		return row.AccountName
	}
}

// tenantAdminRoleName 基线内置管理员角色名（迁移 000010 播种；系统管理员组
// 成员由 tenant-admin 角色绑定实时推导的单一事实源）。
const tenantAdminRoleName = "tenant-admin"

// ResolveTenantAdmins 解析租户管理员（基线内置角色 tenant-admin 的在任成员），
// 作为审批人解析失败的兜底转交对象（v1.1 第 17 章补充语义）。
func (p *IdentityProvider) ResolveTenantAdmins(ctx context.Context, tenantID uint) ([]model.Actor, error) {
	rows := make([]iammodel.User, 0)
	if err := infrastructure.ResolveDB(ctx, p.db).
		Model(&iammodel.User{}).
		Joins("JOIN tn_user_roles ON tn_user_roles.user_id = tn_users.id").
		Joins("JOIN tn_roles ON tn_roles.id = tn_user_roles.role_id").
		Where("tn_roles.name = ?", tenantAdminRoleName).
		Where(activeMemberScope(), iammodel.MemberStatusActive).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	actors := make([]model.Actor, 0, len(rows))
	for i := range rows {
		actors = append(actors, model.Actor{
			MemberID:    rows[i].ID,
			DisplayName: p.MemberDisplayName(ctx, tenantID, rows[i].ID),
		})
	}
	return actors, nil
}

// MemberContext 成员运行上下文：starter.user_id 取登录名（账号 name，全局
// 唯一的组织维度键），starter.department_id 取首个归属部门（多部门成员取
// tn_department_users 最小部门 ID，表达式场景的确定性约定）。
func (p *IdentityProvider) MemberContext(ctx context.Context, tenantID, memberID uint) (string, uint, error) {
	var row struct {
		AccountName string
	}
	if err := infrastructure.ResolveDB(ctx, p.db).
		Table((&iammodel.User{}).TableName()).
		Select("COALESCE(pf_accounts.name, '') AS account_name").
		Joins("LEFT JOIN pf_accounts ON pf_accounts.id = tn_users.account_id").
		Where("tn_users.id = ? AND tn_users.tenant_id = ?", memberID, tenantID).
		Take(&row).Error; err != nil {
		return "", 0, err
	}
	var deptRow struct {
		DepartmentID uint
	}
	// 归属关系行不存在时取 0（成员无部门是合法状态，不视为错误）
	if err := infrastructure.ResolveDB(ctx, p.db).
		Table("tn_department_users").
		Select("COALESCE(MIN(tn_department_users.department_id), 0) AS department_id").
		Joins("JOIN tn_departments ON tn_departments.id = tn_department_users.department_id AND tn_departments.tenant_id = ?", tenantID).
		Where("tn_department_users.user_id = ?", memberID).
		Take(&deptRow).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, err
	}
	return row.AccountName, deptRow.DepartmentID, nil
}

// EnsureInterfaces 编译期端口契约自检。
var _ provider.IdentityProvider = (*IdentityProvider)(nil)
