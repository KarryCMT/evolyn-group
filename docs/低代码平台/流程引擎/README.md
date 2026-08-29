# 流程引擎（Evolyn Workflow Engine）

> Go 原生流程引擎，面向低代码表单与企业审批场景；不是 Go 版 Flowable，
> 不做 BPMN/DMN 兼容。架构定版结论见架构文档第 27 章 ADR-012。

## 文档索引

| 文档 | 状态 | 说明 |
| --- | --- | --- |
| [流程引擎阶段开发功能设计v1.1.md](./流程引擎阶段开发功能设计v1.1.md) | **现行基线** | V1 定版稿：架构分层、DSL v1 协议、状态机语义、事务边界、表单交互、审批人快照、事件/Job、API/错误码/权限、Phase 0~10 计划 |
| [流程引擎阶段开发功能设计v1.0.md](./流程引擎阶段开发功能设计v1.0.md) | 历史版本 | 已被 v1.1 取代（内核位置、DSL 单文档存储、事务与 Outbox 复用等均有修正），保留作演进记录 |

## 实现位置映射

```text
apps/evolyn-core/
├── internal/engine/workflow/     纯引擎内核（禁 gin/gorm/redis/http）
│   ├── model/                    DSL v1 协议类型与领域模型（唯一协议事实源）
│   ├── definition/               DSL 严格校验器 + 发布预编译器
│   ├── task/                     状态机迁移表（状态语义唯一事实源）
│   ├── expression/               Expr 白名单沙箱（expr-lang/expr）
│   ├── repository/               仓储 SPI（GORM 适配在 platform 层）
│   ├── provider/                 平台能力窄端口（Form/组织/身份/事件/时钟）
│   ├── executor/                 节点执行器 SPI
│   ├── assignment/               审批人解析 SPI（V1 能力矩阵）
│   ├── runtime/                  WorkflowContext / Navigator 契约
│   └── event/                    workflow.* 事件目录（冻结）
└── internal/platform/workflow/   平台适配层（Controller/Service/GORM/
                                  Provider Adapter/Job Worker，Phase 1+ 落地）
```

## 里程碑状态

- **Phase 0**：架构、协议与语义冻结——全部 SPI 契约草案、DSL 校验器、状态机
  迁移表、表达式沙箱及单元测试已落地（commit cba47b2）。
- **Phase 1**：Definition Engine 已落地——迁移 000048
  （wf_definition + wf_definition_version，含 down 与 scripts/db.sql 同步）、
  GORM 仓储、DefinitionService（CRUD/草稿乐观锁/严格校验/Expr 预编译/发布
  事务/版本查询）、/api/v1/workflows 全套接口（Swagger 中文注解）、
  workflows 资源基线管理员补授；联调可直接以 JSON DSL 创建并发布定义。
- **Phase 2（当前）**：最小 Runtime 已落地——迁移 000049（wf_instance/
  wf_execution/wf_node_instance/wf_task/wf_task_actor/wf_operation 六表 +
  运行实例业务幂等/请求幂等部分唯一索引）；内核 Runtime（推进环/执行器/
  Task Engine 行锁审批/节点完成判定/事件端口）；API：POST
  /workflow-instances（发起，双层幂等）、GET /workflow-instances/:id
  （详情时间线）、POST /workflow-tasks/:taskId/approve（同意，双击防护）；
  workflow-instances/workflow-tasks 资源补授（发起/查看/审批授全体成员，
  实例级由 TaskActor 校验兜底）；Phase 6 前事件经日志适配器落流水。
- **Phase 3（当前）**：Form + Expr + 动态审批人已落地——迁移 000050
  （departments.leader_member_id，IAM leader 前置能力，部门服务写入校验）；
  引擎内核：条件节点执行器、发布预编译产物进程内缓存（运行期禁重编译）、
  form.*/starter.* 表达式上下文经窄端口填充、role/form_field/
  department_manager Resolver（解析为空租户管理员兜底，禁止静默跳过）、
  发布校验器 Resolver 能力门、审批编辑字段权限过滤（editable/required
  之外整体拒绝 WORKFLOW_FORM_FIELD_FORBIDDEN）；平台层：identity/
  organization/form 三条窄端口适配器（JOIN 聚合路径显式租户条件）、
  form 域 WorkflowRecordStore 窄端口（合并值按冻结快照整体终审后同事务
  写回，FORM_RECORD_INVALID + fieldErrors 与提交同协议）、Approve 接口
  values 传值；发起人直属主管（reporting）IAM 无此语义，保持禁用。
- **Phase 4（当前）**：完整人工 Task Engine 已落地（V1 核心完成）——
  迁移 000051（wf_cc_record 抄送记录表）；任务级动作 Reject（terminate
  联动）/ReturnToStarter/Transfer（TRANSFERRED 回链 + 新任务）与实例级
  动作 Withdraw（撤回窗口：WithdrawAllowed 冻结规则）/Terminate（独立
  workflow-instances:update 权限）/Resubmit（WAITING_RESUBMIT → 从退回
  节点继续）；CC 节点执行器（独立记录表，不参与完成判定）；审批中心
  API：GET /workflow-tasks?scope=pending|completed|cc-to-me、GET
  /workflow-instances?scope=started-by-me、GET /workflow-tasks/:id
  （详情上下文：表单冻结快照 + 业务数据 + 字段权限 + AllowedActions +
  操作时间线）；稳定码 WORKFLOW_ACTION_NOT_ALLOWED。
- **Phase 5（当前）**：wf_job 延时任务已落地——迁移 000052（wf_job，
  status+execute_at 领取索引 + 任务维度部分索引）；DSL approval 节点新增
  `timeout`（seconds + approve/reject 动作）与 `reminder`（seconds）显式
  配置，校验器封顶 30 天；任务创建时按配置排期，任务/节点/实例终态联动
  取消在途 Job；WorkflowJobWorker 30s 轮询，`FOR UPDATE SKIP LOCKED`
  单条领取，claim+执行+回写同事务（crash 自动回滚为 PENDING），失败重试
  独立事务记账（未超限回队 + 60s 退避，超限 FAILED + last_error）；超时
  自动 approve/reject 强制经 Runtime → Task Engine 正常执行路径
  （AutoTimeout：操作人 0=系统、TIMEOUT 操作流水、事件不变），提醒落
  REMINDER 操作流水（Phase 6 接通知域后升级推送）。
- **Phase 6**（workflow.* 事件接入既有 Outbox）为下一里程碑。
- 前端错误码：workflow 域 errCode 已在
  `apps/evolyn-web/packages/utils/src/request/errorCodes.ts` 预留分段。

