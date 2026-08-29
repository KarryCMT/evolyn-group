// Package workflow Evolyn Workflow Engine 纯引擎内核（流程引擎 V1.1 定版，
// 见 docs/低代码平台/流程引擎/流程引擎阶段开发功能设计v1.1.md 第 5 章）。
//
// 分层与依赖规则（七条核心架构原则，文档 3.1）：
//
//   - 本包是纯 Go 流程引擎内核，禁止依赖 gin、gorm、redis、http 及任何
//     internal/platform/* 具体实现（含 form/notification 具体仓储）；
//     平台能力一律经本包 provider/ 中声明的窄端口接口由适配层注入；
//   - 所有事务经平台层既有 TxManager / ResolveDB 统一建立（唯一事务边界），
//     内核仅通过 repository/ 接口感知原子操作，不感知 GORM；
//   - PostgreSQL 是 Workflow Runtime 状态的唯一事实源；
//   - Workflow 不直接更新低代码表单数据，必须经 BusinessDataProvider
//     （平台层 Form Domain Adapter）完成校验与写回；
//   - 流程定义发布后版本冻结，运行实例固定绑定 Definition Version 与
//     Form Version / Snapshot，不随设计态变更自动升级；
//   - 外部副作用经既有 EventPublisher / Outbox 异步处理，人工审批事务
//     不因通知失败回滚；
//   - 平台适配层位于 internal/platform/workflow（Controller/Service/GORM
//     Repository Adapter/Provider Adapter/Worker），依赖方向只允许
//     platform/workflow → engine/workflow。
//
// Phase 0（架构、协议与语义冻结）交付：DSL v1 协议（model/dsl.go）、
// DSL 严格校验器（definition/）、状态机迁移表（task/state_machine.go）、
// 全部 SPI 契约草案（repository/provider/executor/assignment/expression/
// event）；不产生任何核心 wf_* 表写入逻辑（随 Phase 1/2 落地）。
package workflow
