// Package workflow Workflow 平台适配层（流程引擎 V1.1 定版，设计文档
// 第 5.2/20 章）。本包是引擎内核与平台设施之间的装配薄壳：
//
//	platform/workflow → engine/workflow（依赖方向唯一）
//	platform/workflow → infrastructure/TxManager（唯一事务边界）
//	platform/workflow → form / iam / notification（Provider Adapter 桥接）
//
// 禁止引擎内核反向依赖本包；本包禁止绕过内核 SPI 直接解释流程状态。
//
// 子目录规划（随里程碑逐批落地，不预建空壳）：
//
//	controller/  Definition / Instance / Task API（/api/v1，Tenant→
//	             TenantStatus→Authorization 中间件链，Swagger 中文注解）
//	service/     应用服务与 httpx.BizError 稳定错误码（见下方错误码段）
//	repository/  GORM 仓储适配（engine/workflow/repository SPI 实现；
//	             一律经 infrastructure.ResolveDB 加入调用方事务）
//	model/       GORM 持久化模型（wf_* 表；JSONB 单文档存 DSL）
//	dto/         出网 DTO（不暴露引擎内部模型；时间统一 model.JSONTime）
//	adapter/     BusinessDataProvider / OrganizationProvider /
//	             IdentityProvider / EventPublisher（桥接既有 Outbox）
//	worker/      WorkflowJobWorker（FOR UPDATE SKIP LOCKED 轮询；
//	             自动动作必须调用 Task Engine 正常执行路径，第 19.4 章）
//	route.go     路由注册（Phase 1 落地）
//
// 业务错误码段（第 20.5 章冻结，编号随 Phase 1 按仓库规则分配；
// 前端 apps/evolyn-web/packages/utils/src/request/errorCodes.ts 同步对齐）：
//
//	WORKFLOW_DEFINITION_INVALID
//	WORKFLOW_VERSION_NOT_PUBLISHED
//	WORKFLOW_INSTANCE_ALREADY_RUNNING
//	WORKFLOW_INSTANCE_NOT_RUNNING
//	WORKFLOW_TASK_NOT_PENDING
//	WORKFLOW_TASK_FORBIDDEN
//	WORKFLOW_ACTION_NOT_ALLOWED
//	WORKFLOW_EXPRESSION_INVALID
//	WORKFLOW_ASSIGNEE_NOT_FOUND
//	WORKFLOW_FORM_VERSION_INVALID
//	WORKFLOW_FORM_FIELD_FORBIDDEN
//
// 权限资源规划（第 21 章）：workflows:* / workflow-instances:* /
// workflow-tasks:*；RBAC 决定操作能力，TaskActor / Runtime 负责实例级校验。
package workflow
