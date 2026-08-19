# 低代码平台架构设计（专题）

## 范围

本专题承载 evolyn-group 从 Kubernetes 管理平台（weave 二次开发）向**企业级低代码平台**（对标简道云形态）转型的整体架构设计，覆盖技术选型、七大核心引擎、数据模型、顶层工程拓扑与演进路线。

## 状态

- **设计基线，持续更新**：第一部分（第 1–21 章）为技术选型与引擎能力设计；第二部分（第 22 章起）为 2026-08 定版的顶层设计收敛稿。七项关键 ADR 结论（HTTP+OpenAPI、Redis Stream、前端 TypeScript、版本粒度仅表单与流程、多租户一级能力、数据表格 VTable、应用模板）已并入现行章节，其中多租户完整定版（原顶层设计第 25 章）恢复并扩展为第 26 章，含与 M0 现状的差距及 P0–P2 落地阶段。
- 实现自 M0 起步；M1（账号×成员拆分与租户体系补齐）后端全链路已落地，后续 ADR 自第 27 章集中登记（ADR-006 账号×成员拆分、ADR-007 后端工程结构定版）。

## 子文档

- [企业级低代码平台技术架构设计.md](./企业级低代码平台技术架构设计.md)：唯一设计基线，阅读入口。
- [第一期/第一期SaaS平台底座整改文档.md](./第一期/第一期SaaS平台底座整改文档.md)：第一期 SaaS 底座收口整改（FIX-001~019：租户内唯一约束、跨租户校验、租户状态拦截、平台/租户路由隔离、SQL Migration、成员闭环/配额/注销/审计、模型分层），含 FIX-018/019 评审结论（第 12 章）。
- [M1-账号成员拆分与租户体系补齐-任务清单.md](./M1-账号成员拆分与租户体系补齐-任务清单.md)：M1 在办任务跟踪（对标简道云差距落地），完成项勾选推进。
- [登录页公共配置-需求整理.md](./登录页公共配置-需求整理.md)：对标简道云 get_public_configuration 的登录页初始化配置需求归集（字段语义、采纳/暂缓/裁剪、响应模型意向），只含需求不含实现任务。

## 正式实现位置

| 规划目录 | 来源 | 说明 |
| --- | --- | --- |
| `apps/evolyn-core/` | 现有目录演进（不改名） | Go + Gin 平台主体（cmd/api、internal/{platform,engine,infrastructure}；域模块化结构已按 ADR-007 落地，含 iam/tenant/auth/audit 四域、migrations/ 版本化 SQL，engine 随 M2 起） |
| `apps/evolyn-web/` | 现有目录演进（不改名） | pnpm + Turborepo monorepo：`apps/web/` Vue3 + TypeScript 主应用（脚手架阶段）、`packages/` 共享库 |
| `services/workflow/` | 新建 | Java + Spring Boot + Flowable 流程服务 |
| `packages/openapi/` | 新建 | 全平台 API 契约唯一事实源 |

> 本目录存放设计文档；后端 swagger 生成物位于 `apps/evolyn-core/docs/`（gitignored，`make swagger` 本地查阅），不入库。
