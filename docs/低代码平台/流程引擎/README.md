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

- **Phase 0（当前）**：架构、协议与语义冻结——全部 SPI 契约草案、DSL
  校验器、状态机迁移表、表达式沙箱及单元测试已落地；不产生 `wf_*`
  表写入逻辑。
- Phase 1 Definition Engine → Phase 4 完整人工 Task Engine 为 V1 核心
  （详见 v1.1 文档第 25 章）；`wf_*` 表结构自 Phase 1 起按迁移规范落地
  （迁移编号自 000048 顺延）。
- 前置排期：IAM 部门负责人（leader）/直属主管（reporting）字段随
  Phase 3 开工前补齐，此前对应 Resolver 保持禁用。
- 前端错误码：workflow 域 errCode 已在
  `apps/evolyn-web/packages/utils/src/request/errorCodes.ts` 预留分段。
