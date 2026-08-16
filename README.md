# evolyn-group

企业级低代码平台（对标简道云形态）的单仓多应用工程，由 Kubernetes 管理平台 weave 二次演进而来。

## 仓库结构

| 目录 | 说明 |
| --- | --- |
| `apps/evolyn-core/` | Go 1.25 + Gin 后端（认证、组织、RBAC；低代码引擎按里程碑演进） |
| `apps/evolyn-web/` | Vue 3 + Element Plus 前端（设计器与运行态，TypeScript 渐进迁移中） |
| `services/` | 规划中的独立服务（Java + Flowable 工作流，M4 落地） |
| `packages/` | 规划中的共享契约（OpenAPI 唯一事实源） |
| `deploy/` | 本地/部署编排（docker-compose 起 PostgreSQL/Redis/MinIO） |
| `docs/` | 设计文档与架构基线 |

## 快速开始

```bash
# 1. 启动本地依赖（PostgreSQL / Redis / MinIO）
docker compose -f deploy/docker-compose.yaml up -d

# 2. 后端（默认读取 apps/evolyn-core/config/app.yaml）
npm run dev:core

# 3. 前端（/api 代理到 http://localhost:8080）
npm run dev:web
```

数据库初始化脚本见 `apps/evolyn-core/scripts/db.sql`（compose 首次启动自动导入）。

## 架构与路线图

整体设计见 [docs/低代码平台/企业级低代码平台技术架构设计.md](docs/低代码平台/企业级低代码平台技术架构设计.md)：
七大核心引擎（Schema / Query / Permission / Relation / Formula / Dashboard / DDL）+ 多租户 + 应用模板，
按 M0–M7 里程碑演进，当前处于 M0（工程重构）阶段。

## 开发约定

各子项目的详细约束见 [AGENTS.md](AGENTS.md)；CI 在 `.github/workflows/ci.yml`（后端 build/vet/test/gofmt，前端 build）。
