# 低代码平台架构设计（专题）

## 范围

本专题承载 evolyn-group 从 Kubernetes 管理平台（weave 二次开发）向**企业级低代码平台**（对标简道云形态）转型的整体架构设计，覆盖技术选型、七大核心引擎、数据模型、顶层工程拓扑与演进路线。

## 状态

- **设计基线，持续更新**：第一部分（第 1–21 章）为技术选型与引擎能力设计；第二部分（第 22–32 章）为 2026-08 定版的顶层设计，含七项关键 ADR（HTTP+OpenAPI、Redis Stream、前端 TypeScript、版本粒度仅表单与流程、多租户一级能力、数据表格 VTable、应用模板）。
- 实现尚未开始，按第 29 章路线图从 M0 起步。

## 子文档

- [企业级低代码平台技术架构设计.md](./企业级低代码平台技术架构设计.md)：唯一设计基线，阅读入口。

## 正式实现位置

| 规划目录 | 来源 | 说明 |
| --- | --- | --- |
| `apps/server/` | 由 `apps/evolyn-core` 演进 | Go + Gin 平台主体（cmd/api、cmd/worker、internal/{platform,engine,infrastructure}） |
| `apps/web/` | 由 `apps/evolyn-web` 演进 | Vue3 + TypeScript 设计器与运行态 |
| `services/workflow/` | 新建 | Java + Spring Boot + Flowable 流程服务 |
| `packages/openapi/` | 新建 | 全平台 API 契约唯一事实源 |

> 本目录存放设计文档；仓库根 `docs/` 下的 `docs.go`、`swagger.*` 为旧后端 swagger 生成产物，与本专题无关。
