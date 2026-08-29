package repository

import (
	"context"

	"evolyn/internal/engine/workflow/model"
)

// InstanceRepository 流程实例仓储（行锁/部分唯一索引由适配层承担）。
type InstanceRepository interface {
	// CreateInstance 创建实例；tenant+business_type+business_id 的
	// RUNNING 部分唯一索引兜底业务幂等（第 14.1 章）
	CreateInstance(ctx context.Context, instance *model.Instance) error
	// FindInstanceByID 读取实例（租户过滤）
	FindInstanceByID(ctx context.Context, tenantID, instanceID uint) (*model.Instance, error)
	// FindInstanceByIDForUpdate 行锁读取（Approve 事务模板第 4 步：
	// 校验 instance = RUNNING 在锁内进行）
	FindInstanceByIDForUpdate(ctx context.Context, tenantID, instanceID uint) (*model.Instance, error)
	// SaveInstance 保存实例状态变更（状态迁移必须先经 task 包迁移表校验）
	SaveInstance(ctx context.Context, instance *model.Instance) error
	// FindRunningInstanceByBusiness 幂等查询：同 tenant+type+id 的
	// RUNNING 实例（无则返回 nil, nil）
	FindRunningInstanceByBusiness(ctx context.Context, tenantID uint, businessType, businessID string) (*model.Instance, error)
	// FindInstanceByIdempotencyKey 请求幂等查询：同租户幂等键的既有实例
	//（无则返回 nil, nil；命中即重放返回，第 14.2 章）
	FindInstanceByIdempotencyKey(ctx context.Context, tenantID uint, key string) (*model.Instance, error)
	// HasRunningInstanceByDefinition 定义删除前置校验：是否存在运行中实例
	HasRunningInstanceByDefinition(ctx context.Context, definitionID uint) (bool, error)
}

// ExecutionRepository 执行路径仓储（V1 仅根路径，Phase 8 扩展并行）。
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *model.Execution) error
	FindExecutionByID(ctx context.Context, tenantID, executionID uint) (*model.Execution, error)
	ListExecutionsByInstance(ctx context.Context, instanceID uint) ([]model.Execution, error)
	SaveExecution(ctx context.Context, execution *model.Execution) error
}
