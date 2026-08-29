// Package provider Workflow 引擎平台能力窄端口（第 4/15 章架构）。
//
// 内核禁止依赖具体 IAM/Form/Notification 实现；平台适配层
// （internal/platform/workflow/adapter）实现本包接口后注入装配。
// 端口方法保持最小面：只暴露流程语义必需的能力，避免引擎反向
// 膨胀为平台门面。
package provider

import (
	"context"

	"evolyn/internal/engine/workflow/model"
)

// BusinessRef 业务数据定位（第 15.2 章）：至少包含租户、应用、表单、
// 表单版本快照与业务标识；流程运行期间 FormVersionID 固定为发起时绑定值。
type BusinessRef struct {
	TenantID      uint
	AppID         uint
	FormID        uint
	FormVersionID uint
	BusinessID    string
}

// BusinessDataProvider 业务数据窄端口（第 15.1 章铁律的唯一豁口）：
// Workflow 禁止直接 UPDATE form_records，表单读写必须经本端口由
// Form Domain 完成校验（Schema Snapshot 约束、字段类型与必填）。
type BusinessDataProvider interface {
	// GetData 读取绑定表单快照的业务数据（表达式 form.* 数据源）
	GetData(ctx context.Context, ref BusinessRef) (map[string]any, error)
	// UpdateData 更新业务数据：values 仅允许节点字段权限授权字段，
	// Form Domain 按 Snapshot 校验失败必须整体报错（事务回滚）
	UpdateData(ctx context.Context, ref BusinessRef, values map[string]any) error
}

// OrganizationProvider 组织窄端口（第 17 章前置约束）：只暴露既有 IAM
// 真实具备的组织语义；IAM 未支持的能力（如部门负责人）不得在本端口
// 出现「猜测实现」，前置能力落地前对应方法返回明确不支持错误。
type OrganizationProvider interface {
	// ResolveRoleMembers 解析角色下有效成员（同租户）
	ResolveRoleMembers(ctx context.Context, tenantID uint, roleCode string) ([]model.Actor, error)
	// ResolveDepartmentMembers 解析部门（含子部门可选策略）有效成员
	ResolveDepartmentMembers(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error)
	// ResolveDepartmentManager 解析部门负责人（需 IAM leader 前置能力）
	ResolveDepartmentManager(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error)
	// ResolveStarterManager 解析发起人直属主管（需 IAM reporting 前置能力）
	ResolveStarterManager(ctx context.Context, tenantID, starterMemberID uint) ([]model.Actor, error)
	// MemberDisplayName 成员显示名快照（任务创建时固化）
	MemberDisplayName(ctx context.Context, tenantID, memberID uint) string
}

// IdentityProvider 身份窄端口：运行期身份与兜底能力（审批人解析失败
// 默认转交租户管理员，v1.1 补充语义）。
type IdentityProvider interface {
	// ValidateMembers 校验成员集合同租户且有效（快照写入前复核）
	ValidateMembers(ctx context.Context, tenantID uint, memberIDs []uint) error
	// MemberDisplayName 成员显示名快照（任务创建时固化）
	MemberDisplayName(ctx context.Context, tenantID, memberID uint) string
	// ResolveTenantAdmins 解析租户管理员成员（Assignee 兜底/terminate 通知）
	ResolveTenantAdmins(ctx context.Context, tenantID uint) ([]model.Actor, error)
}
