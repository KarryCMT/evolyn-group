package service

import (
	"context"

	tenantmodel "evolyn/internal/platform/tenant/model"
)

// TenantService 租户域服务（运营面，P3-1）。调用链来自 /platform 域
// 控制器；仓储层内部已剥离租户上下文，防止 tenants 表自我过滤
type TenantService interface {
	// Open 开通租户：租户 + owner 账号/成员 + 租户内系统组/角色种子
	Open(ctx context.Context, req *OpenTenantRequest) (*tenantmodel.Tenant, error)
	List(ctx context.Context) ([]tenantmodel.Tenant, error)
	Get(ctx context.Context, id string) (*tenantmodel.Tenant, error)
	Update(ctx context.Context, id string, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error)
	// SetStatus 生命周期流转：active / frozen / deleted
	SetStatus(ctx context.Context, id string, status string) error
}
