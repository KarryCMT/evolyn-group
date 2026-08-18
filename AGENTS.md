# evolyn-group Agent Guide

本文件适用于整个仓库。进入子目录工作时，先检查该子目录是否另有更近的 `AGENTS.md`；若没有，则遵守本文件。

## 项目总览

本仓库是企业级低代码平台（对标简道云形态）的单仓工程，由 Kubernetes 管理平台 weave 二次演进而来。整体架构与演进路线见 `docs/低代码平台/企业级低代码平台技术架构设计.md`（M0–M7 里程碑；当前处于 M1 后段：账号×成员拆分与租户体系补齐的后端全链路已落地，见第 26/27 章与 ADR-006/007）。

- `apps/evolyn-core/`: Go 1.25 + Gin 后端，模块名保持 `evolyn`。当前承载认证（JWT + OAuth）、平台账号/租户成员、部门、分组、自定义 RBAC、租户与套餐配额；低代码引擎（Schema/Query/Permission 等）按里程碑逐步落地。
- `apps/evolyn-web/`: pnpm + Turborepo 前端 monorepo（Vue 3.5 + TypeScript）。主应用在 `apps/web/`（脚手架阶段），共享库在 `packages/`，文档站在 `apps/docs/`。
- `services/`、`packages/`（仓库根）: 规划目录（Java/Flowable 工作流、OpenAPI 契约），落地前不要创建同名内容。
- `deploy/`: `docker-compose.yaml` 一键起 PostgreSQL 16/Redis 7/MinIO。
- 根目录 `.golangci.yml`、`.staticcheck-version` 等：仓库级 lint 配置；后端域内另有 `apps/evolyn-core/.golangci.yaml`。

分支约定：当前固定在 `1.0.0` 分支开发，未明确要求前不要切换到其他分支；`2.0.0` 为基线分支，`3.0.0` 为后续规划分支。

## 通用约束

- 不要跨子项目混装依赖。前端统一用 pnpm（`apps/evolyn-web` 为 workspace，`pnpm-lock.yaml` 为准，Node >= 22）；Go 命令在 `apps/evolyn-core` 内执行（根 `go.work` 已纳入）。
- 不要提交或手改生成/构建产物：`node_modules/`、`dist/`、`.turbo/`、`bin/`、`cover.out`、`coverage.txt`、swagger 生成的 `apps/evolyn-core/docs/`、`components.d.ts`。
- 证书与私钥不入库（`.gitignore` 已拦截 `certs/`、`*.key` 等）；本地证书用 `apps/evolyn-core/scripts/cert.sh` 生成。
- 真实密钥、token、生产连接串不写入代码或文档。`config/app.yaml` 是本地开发默认值，生产配置通过 `config/app.example.yaml` 复制后填写，不入库。
- 后端模块导入路径是 `evolyn/...`，应用名 evolyn 只是目录名，不要重命名 go module 或批量改导入路径。
- 业务代码一律放 `internal/`（编译器强制外部不可导入）；顶层 `pkg/` 已解散（ADR-007），不要再新建。后续引擎代码放 `internal/engine/`，禁止 import gin/gorm。
- 修改 API 字段、路由、权限模型时，同步检查 `apps/evolyn-web/apps/web/src`（`router/index.ts`、`pages/`）与后端 swagger。
- 遇到工作区已有未提交改动时，默认视为用户改动；只处理当前任务相关文件，不回滚无关文件。
- 改动的代码需要补充相关注释，简单逻辑简要注释，复杂逻辑详细注释。注释可使用中文。

## evolyn-core 后端

技术栈与入口：

- Go 1.25，Gin + GORM（Postgres）+ go-redis + swaggo。
- 入口 `cmd/api/main.go`，默认配置 `config/app.yaml`（可用 `--config` 覆盖）。
- 关键配置段：`server`（端口 8080、jwtSecret、rateLimits）、`db`（库名 `evolyn`，`migrate: true` 表示启动时 GORM 自动迁移）、`redis`、`oauth`。

目录职责（域模块化定版，ADR-007）：

```text
cmd/api/              入口，只做装配
internal/
  config/             配置解析
  contextx/           通用 context 键值存取（含租户上下文）
  model/              跨域共享内核（BaseModel：TenantID + 时间三件套）
  metrics/            监控指标
  infrastructure/     postgres/redis/pgx 客户端、GORM 租户 Callback、生命周期
  utils/              ratelimit/request/set/trace
  version/            版本信息（Makefile ldflags 注入，路径勿动）
  platform/
    server/           HTTP 服务器装配与路由注册（依赖注入汇聚点）
    controller/       Controller 注册契约（RegisterRoute/Name）与 index
    ginctx/           gin.Context 会话/上下文存取助手
    httpx/            统一响应封装
    middleware/       认证/租户/鉴权/限流/CORS/日志/trace/监控
    auth/             认证域：JWT（claims 含 accountId/memberId/tenantId）、
                       OAuth（github/wechat）、auth 控制器
    iam/              身份域（域内 controller→service→repository 小三层）：
                       account 平台账号 / user 租户成员 / group / rbac /
                       department；authorization/ 自定义 RBAC 鉴权（后续接 Casbin）
    tenant/           租户域（小三层）：租户 CRUD 与生命周期、plan 套餐与配额
scripts/              db.sql（冷启动初始化）、cert.sh（本地证书）
```

常用命令（在 `apps/evolyn-core` 内执行）：

```bash
make run            # 本地运行（需 postgres/redis 就绪，读 config/app.yaml）
make build          # 构建到 bin/evolyn-core（注入版本 ldflags）
make test           # 单测 + 覆盖率
make fmt            # gofmt -s 格式化
make lint           # golangci-lint（需已安装）
make swagger        # 重新生成 swagger 到 ./docs（gitignored，本地查阅用）
make postgres       # 起本地 postgres 容器并导入 scripts/db.sql
make redis          # 起本地 redis 容器
```

后端改动规则：

- 域模块化：新资源归属既有域（iam/tenant/auth）或按第 27 章 ADR-007 的结构新
  建域，域内遵循 controller → service → repository 小三层，同步补各层
  `interface.go` 与 `internal/platform/server/server.go` 装配。
- 账号×成员拆分（ADR-006）是现行模型基线：登录身份（name/phone/password/
  OAuth 凭证）只挂 `accounts`；租户内身份（昵称/部门/分组/角色）挂 `users`。
  迁移期字段残留由启动幂等回填处理，新代码不要声明旧列。
- 运营域路由（`/api/v1/platform`）无租户上下文，查询 tenants 时注意避免租户
  Callback 自我过滤（见 `internal/platform/tenant/repository/tenant.go` 注释）。
- 数据库结构变更依赖 GORM migrate（model 定义），`scripts/db.sql` 仅作冷启动
  初始化，两者保持一致。
- 提交前至少运行 `go test ./...`（或说明未运行原因），格式化使用 `make fmt`。

## evolyn-web 前端

技术栈与结构（pnpm + Turborepo monorepo，TypeScript 全量）：

- 主应用 `apps/web/`（`@evolyn.do/web`）：Vue 3.5 + Vite + Element Plus
  （`unplugin-vue-components` 按需自动导入，`components.d.ts` 勿手改）+ Sass，
  路径别名 `~/` 指向 `src/`。
- 共享库 `packages/`：`ui`（组件库）、`utils`、`hooks`、`directives`、
  `lint-configs`（eslint/prettier/stylelint/commitlint/typescript 配置）。
- 文档站 `apps/docs/`（VitePress）。依赖版本走 `pnpm-workspace.yaml` 的
  `catalog:` 统一管理。
- 主应用目录：`src/main.ts`（入口）；`src/router/index.ts`（**手动路由表**，
  与 `src/pages/` 目录一一对应，新增页面在此登记）；`src/pages/`（页面）；
  `src/composables/`（如 dark 暗色模式）；`src/styles/`（element 主题覆盖）。
  布局与菜单组件（侧栏/顶栏）随首个前端业务批落地。

常用命令（在 `apps/evolyn-web` 内执行，需 pnpm >= 10）：

```bash
pnpm install                        # workspace 全量安装（only-allow pnpm）
pnpm -F @evolyn.do/web dev          # 主应用本地开发
pnpm -F @evolyn.do/web typecheck    # vue-tsc 类型检查
pnpm -F @evolyn.do/web build        # 生产构建
```

前端改动规则：

- 使用 Composition API（`<script setup>`）+ TypeScript；`typecheck` 必须通过。
- 新增页面在 `src/pages/` 建文件并到 `src/router/index.ts` 登记（不用文件式
  自动路由）；布局/菜单组件落地后，菜单入口同步维护对应布局组件。
- 数据表格统一采用 VisActor VTable 生态（见架构文档第 3 章），新列表页不要
  引入其他表格库。
- 组件库改动在 `packages/ui` 内进行并通过 changeset 发版，主应用优先复用
  `@evolyn.do/*` workspace 包，不重复引第三方实现。

## 验证建议

- 后端：`cd apps/evolyn-core && go test ./...`，必要时 `make lint`、`make run`。
- 前端：`cd apps/evolyn-web && pnpm -F @evolyn.do/web typecheck && pnpm -F @evolyn.do/web build`；联调用 `deploy/docker-compose.yaml` 起依赖。
- 跨端接口：字段名、错误码、鉴权头在 evolyn-core 控制器与前端调用侧两侧核对（前端 HTTP 层随首个业务批落地）。

## 文档约定

- `docs/低代码平台/`：平台架构设计专题，含设计基线（第 1–21 章选型、第 22 章起顶层定版）与 README 导航；重大技术决策以追加 ADR 方式记录在第 27 章，不回改已定版结论原文，被取代表述在原章节就地标注指向关系。
- 其他业务文档按子目录组织并配 `README.md` 导航，不散落根目录。
- 移动或改名文档用 `git mv`，同一次改动内修复引用；整理只调结构，不改写业务结论。

## 文档维护

- 新增子项目、脚本、关键目录或里程碑落地时，同步更新本文件与 `docs/低代码平台/README.md` 的实现位置映射。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
