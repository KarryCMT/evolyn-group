package model

// Instance 业务数据的一次流程执行（第 6.2 / 8.2 章）。
// 发起时一次性绑定 Definition Version 与 Form Version / Snapshot，
// 运行期间禁止因设计态重新发布而切换。
type Instance struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// DefinitionID 所属定义（内部外键；与版本一起冗余便于查询）
	DefinitionID uint
	// DefinitionVersionID 绑定的不可变流程版本
	DefinitionVersionID uint
	// BusinessType / BusinessID 业务绑定（同一 tenant+type+id 同一时间
	// 至多一个 RUNNING 实例，部分唯一索引保障，第 14.1 章）
	BusinessType string
	BusinessID   string
	// AppID / FormID / FormVersionID 发起时冻结的表单绑定（0=未绑定）
	AppID         uint
	FormID        uint
	FormVersionID uint
	// Status 实例状态机（迁移表见 task/state_machine.go）
	Status InstanceStatus
	// StarterMemberID 发起人成员 ID（Withdraw 资格与退回发起人的目标）
	StarterMemberID uint
	// IdempotencyKey 请求幂等键（空=未携带；同租户非空唯一，第 14.2 章）
	IdempotencyKey string
}

// InstanceStatus 流程实例状态机（第 9.1 章，Phase 0 冻结，Runtime 代码
// 不得自行解释；迁移表唯一定义见 task 包 state_machine.go）。
type InstanceStatus string

const (
	// InstanceStatusDRAFT 定义保留；V1 Runtime 可不创建草稿实例
	InstanceStatusDRAFT InstanceStatus = "DRAFT"
	// InstanceStatusRUNNING 流程进行中（含等待人工任务或退回发起人修改）
	InstanceStatusRUNNING InstanceStatus = "RUNNING"
	// InstanceStatusCOMPLETED 正常进入 End（终态）
	InstanceStatusCOMPLETED InstanceStatus = "COMPLETED"
	// InstanceStatusREJECTED 执行终止型驳回（终态）
	InstanceStatusREJECTED InstanceStatus = "REJECTED"
	// InstanceStatusCANCELLED 发起人撤回或管理员终止（终态）
	InstanceStatusCANCELLED InstanceStatus = "CANCELLED"
)

// Execution 运行路径，为并行（Phase 8）、子流程预留；V1 仅根执行路径。
type Execution struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// InstanceID 所属实例
	InstanceID uint
	// ParentExecutionID 父路径（V1 根路径为 0）
	ParentExecutionID uint
	// Status 执行路径状态机
	Status ExecutionStatus
}

// ExecutionStatus 执行路径状态。V1 顺序流只经历 RUNNING → 终态；
// 并行 split/join 语义 Phase 8 扩展（届时追加状态而非复用现有值）。
type ExecutionStatus string

const (
	ExecutionStatusRUNNING   ExecutionStatus = "RUNNING"
	ExecutionStatusCOMPLETED ExecutionStatus = "COMPLETED"
	ExecutionStatusCANCELLED ExecutionStatus = "CANCELLED"
)
