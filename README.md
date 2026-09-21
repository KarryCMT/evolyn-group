# evolyn-group

企业级低代码平台（对标灵衍云形态）的单仓多应用工程，由 Kubernetes 管理平台 weave 二次演进而来。

## 仓库结构

| 目录 | 说明 |
| --- | --- |
| `apps/evolyn-core/` | Go 1.25 + Gin 后端（认证、组织、RBAC、应用/表单与 Go 原生流程引擎） |
| `apps/evolyn-web/` | Vue 3 + TypeScript + Element Plus 前端（应用工作区、表单设计/运行时与流程设计器） |
| `services/` | 预留的独立服务目录；当前不承载工作流，流程引擎已落地在 `evolyn-core` |
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

平台按 M0–M7 里程碑演进，当前处于 **M1 后段**：账号×成员拆分、租户体系、
应用菜单、表单资产/运行时与 Go 原生流程引擎（含流程设计器）均已落地。

现行实现索引见 [低代码平台文档导航](docs/低代码平台/README.md) 与
[流程引擎文档](docs/低代码平台/流程引擎/README.md)。
历史架构草案保留在
[企业级低代码平台技术架构设计.md](docs/低代码平台/企业级低代码平台技术架构设计.md)，
其中早期 Java/Flowable 方案已由 ADR-012 取代，不应作为现行实现依据。

## 开发约定

各子项目的详细约束见 [AGENTS.md](AGENTS.md)；CI 在 `.github/workflows/ci.yml`（后端 build/vet/test/gofmt，前端 build）。
