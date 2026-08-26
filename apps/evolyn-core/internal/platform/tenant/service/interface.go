package service

import (
	"context"

	tenantmodel "evolyn/internal/platform/tenant/model"
)

// TenantService 租户域服务（P3-1）。运营面调用链来自 /platform 域控制器；
// SelfOpen/SelfOpenInTx 供认证域自助开通。仓储层内部已剥离租户上下文，
// 防止 tenants 表自我过滤
type TenantService interface {
	// Open 开通租户：租户 + owner 账号/成员 + 租户内系统组/角色种子
	Open(ctx context.Context, req *OpenTenantRequest) (*tenantmodel.Tenant, error)
	// SelfOpen 自助开通（登录态独立事务）：owner 取自当前账号，编码服务端
	// 生成，套餐默认免费版；onboarding 为注册向导采集的企业画像（写入租户
	// Config，选填）；提交成功后记审计
	SelfOpen(ctx context.Context, ownerAccountID uint, name string, onboarding tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error)
	// SelfOpenInTx 事务内自助开通（注册向导最终提交的组合通道）：加入调用
	// 方外层事务（ctx 携带事务 session），不开独立事务、不记审计——审计由
	// 调用方在外层提交成功后补记，避免回滚留下假流水
	SelfOpenInTx(ctx context.Context, ownerAccountID uint, name string, onboarding tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error)
	List(ctx context.Context) ([]tenantmodel.Tenant, error)
	Get(ctx context.Context, id string) (*tenantmodel.Tenant, error)
	Update(ctx context.Context, id string, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error)
	// UpdateMyName 仅更新当前会话所属租户名称，供租户后台组织根节点自助维护。
	UpdateMyName(ctx context.Context, tenantID uint, name string) (*tenantmodel.Tenant, error)
	// SetStatus 生命周期流转：active / frozen / deleted。
	// deleted 记录注销申请与保留截止时间（FIX-012），到期由 Purge Worker 清理
	SetStatus(ctx context.Context, id string, status string) error
}
