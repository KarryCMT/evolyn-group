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
- 关键配置段：`server`（端口 8080、jwtSecret、rateLimits）、`db`（库名 `evolyn`；`migrations: true` 启动时应用版本化 SQL 迁移【生产路径】，`migrate: true` 为 GORM AutoMigrate【仅开发/测试】，两者互斥）、`tenant`（注销保留期/清理周期）、`redis`、`oauth`。

目录职责（域模块化定版，ADR-007）：

```text
cmd/api/              入口，只做装配
internal/
  config/             配置解析
  contextx/           通用 context 键值存取（租户上下文/操作者/请求元数据、DetachTenant）
  model/              跨域共享内核（PlatformBaseModel / TenantBaseModel）
  metrics/            监控指标
  infrastructure/     postgres/redis/pgx 客户端、GORM 租户 Callback、
                      SQL Migration 执行器（migrate.go）、统一事务
                      （tx.go：TxManager/ResolveDB，ctx 传播事务 session）、
                      生命周期
  testsupport/        集成测试基础设施（TEST_PG_DSN 按需建库/迁移/清理）
  utils/              ratelimit/request/set/trace
  version/            版本信息（Makefile ldflags 注入，路径勿动）
  platform/
    server/           HTTP 服务器装配与路由注册（依赖注入汇聚点）
    controller/       Controller 注册契约（RegisterRoute/Name；
                      PlatformController 标记平台运营域归属）与 AppConf
                      应用配置控制器（GET /app/conf：区号列表/能力开关，
                      公开引导端点直挂 api 组，不经租户域 RBAC）
    ginctx/           gin.Context 会话/上下文存取助手
    httpx/            统一响应封装
    middleware/       认证/租户/租户状态拦截/鉴权（平台与租户两条链）/
                      限流/CORS/日志/trace/监控
    auth/             认证域：JWT（claims 含 accountId/memberId/tenantId）、
                       OAuth（github/wechat）、短信验证码（sms/ 子包：Redis
                       存码 + 冷却/试错上限，场景 login/register；开发通道
                       provider=dev 固定码 666666 + devEcho 回显，其他
                       provider 启动拦截）、auth 控制器（POST /auth/user 为
                       注册向导第 1 步「手机号+验证码」注册即登录：免密注册，
                       服务端生成随机登录名/密码，已注册手机号等价短信登录；
                       POST /auth/tenant 自助开通租户：创建者即所有者并绑定
                       tenant-admin，编码服务端生成）
    iam/              身份域（域内 controller→service→repository 小三层）：
                       account 平台账号（PUT /accounts/me 自助资料含注册向导
                       「完善信息」的 onboarding JSONB 画像，昵称变更同事务同步
                       当前成员昵称；PUT /accounts/me/password 密码修改——免密
                       注册账号 password_initialized=false（迁移 000012）首设
                       免旧密码；迁移 000010/000011）/ user 租户成员 /
                       group / rbac / department；authorization/ 自研 RBAC 鉴权
    tenant/           租户域（小三层）：租户 CRUD 与生命周期（注销保留期/
                      Purge Worker）、plan 套餐与配额、QuotaService 配额执行
    audit/            审计域：业务操作审计日志（Recorder，追加写流水）
migrations/           版本化 SQL Migration（Schema 唯一事实来源，嵌入二进制；
                      命名 NNNNNN_name.(up|down).sql，版本号只增不复用）
scripts/              db.sql（终态快照，与迁移链一致）、cert.sh（本地证书）
```

常用命令（在 `apps/evolyn-core` 内执行）：

```bash
make run            # 本地运行（需 postgres/redis 就绪，读 config/app.yaml）
make build          # 构建到 bin/evolyn-core（注入版本 ldflags）
make test           # 单测 + 覆盖率（离线全绿，真库集测自动跳过）
make test-integration  # 复用本地 postgres（见下方「本地开发/测试数据库约定」）
                    # 以 TEST_PG_DSN 跑全部测试（含 SEC-TENANT-*/MIGRATE-INT-* 真库集成用例）
make vet            # go vet ./...
make fmt            # gofmt -s 格式化
make lint           # golangci-lint（需已安装）
make swagger        # 重新生成 swagger 到 ./docs（gitignored，本地查阅用）
make postgres       # 复用本地 postgres 容器并导入 scripts/db.sql
make redis          # 起本地 redis 容器
```

本地开发/测试数据库约定（固定，勿改）：统一复用本机已存在的 `postgres`
容器（`postgres:14-alpine`，`127.0.0.1:5432`，账号/密码 `postgres/postgres`）。
Makefile 的 `PG_CONTAINER`/`PG_IMAGE`/`PG_HOST`/`PG_PORT`/`TEST_PG_DSN`
默认已指向它，端口已有实例监听时直接复用，不要另建容器或拉其他镜像 tag。
集成测试 DSN 连 `postgres` 库（testsupport 按需建/清临时测试库，零残留）；
业务库 `evolyn` 由 `make postgres` 导入 `scripts/db.sql` 快照。该连接仅用于
开发与测试，生产连接串经 `config/app.example.yaml` 复制填写，不入库。

后端改动规则：

- 域模块化：新资源归属既有域（iam/tenant/auth/audit）或按第 27 章 ADR-007 的结构新
  建域，域内遵循 controller → service → repository 小三层，同步补各层
  `interface.go` 与 `internal/platform/server/server.go` 装配。
- 多步写流程必须走统一事务边界（整改 FIX-020/021）：Service 依赖 TxManager
  接口（`WithinTransaction`），Repository 一律经 `infrastructure.ResolveDB`
  取连接以加入 ctx 传播的事务；跨租户 Update/Delete 必须先加载再写，
  禁止「租户过滤 0 行影响却返回成功」；审计在事务提交后独立写入（best-effort）。
- 账号×成员拆分（ADR-006）是现行模型基线：登录身份（name/phone/password/
  OAuth 凭证）只挂 `accounts`；租户内身份（昵称/部门/分组/角色）挂 `users`。
  迁移期字段残留由启动幂等回填处理，新代码不要声明旧列。
- 路由双域隔离（整改 FIX-008）：`/api/v1/platform/*` 只走
  Authentication + PlatformAuthorization（无租户上下文），平台控制器实现
  `Platform() bool` 标记；其余租户域路由走 Tenant → TenantStatus →
  Authorization 链。关系绑定（角色/分组/部门）必须经 Service 层同租户校验，
  不允许裸 ID 盲写关系表。
- 数据库结构变更以 `migrations/` 版本化 SQL 为唯一事实来源（整改 FIX-009）：
  改 GORM Model 必须同时提交 up/down 迁移并同步 `scripts/db.sql` 快照；
  约束/索引名与迁移文件保持一致以保证快照库重放幂等。AutoMigrate 仅限
  开发/测试（`db.migrate`），禁止依赖其建生产 Schema。
- swagger 注释一律中文：新增/修改接口时 `@Tags`（模块名）、`@Summary`、
  `@Description` 等注释统一写中文，不得新写英文；模块 tag 沿用既有中文名
  （首页/认证/账号/成员/部门/分组/角色权限/平台管理），新模块也起中文名。
  改完执行 `make swagger` 重新生成文档；docs 包经 server.go 的 blank import
  编译进二进制，联调需重启服务生效。
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
  与 `src/pages/` 目录一一对应，新增页面在此登记；`meta.public` 标记公开页，
  全局守卫拦截未登录访问）；`src/pages/`（页面，`auth/` 为登录/注册/找回密码）；
  `src/api/`（HTTP 层：`http.ts` 统一 fetch 封装 + 按域拆分的接口模块，
  对齐后端统一响应 `{code,msg,data}`）；`src/components/auth/`（认证域业务组件：
  AuthLayout 骨架、登录/注册表单等）；`src/composables/`（auth 会话、dark 暗色）；
  `src/styles/`（element 主题覆盖，默认 `el` 命名空间）。布局与菜单组件
  （侧栏/顶栏）随后续前端业务批落地。

常用命令（在 `apps/evolyn-web` 内执行，需 pnpm >= 10）：

```bash
pnpm install                        # workspace 全量安装（only-allow pnpm）
pnpm -F @evolyn.do/web dev          # 主应用本地开发
pnpm -F @evolyn.do/web typecheck    # vue-tsc 类型检查
pnpm -F @evolyn.do/web build        # 生产构建
```

前端改动规则：

- 使用 Composition API（`<script setup>`）+ TypeScript；`typecheck` 必须通过。
- 样式统一引用 Element Plus 默认命名空间 CSS 变量（`--el-*`），不自定义 CSS
  token；类名采用 BEM（块__元素--修饰符），优先组件内 scoped 样式。
- 新增页面在 `src/pages/` 建文件并到 `src/router/index.ts` 登记（不用文件式
  自动路由）；布局/菜单组件落地后，菜单入口同步维护对应布局组件。
- 数据表格统一采用 VisActor VTable 生态（见架构文档第 3 章），新列表页不要
  引入其他表格库。
- 组件库改动在 `packages/ui` 内进行并通过 changeset 发版，主应用优先复用
  `@evolyn.do/*` workspace 包，不重复引第三方实现。

## 验证建议

- 后端：`cd apps/evolyn-core && go test ./...`，必要时 `make lint`、`make run`。
- 前端：`cd apps/evolyn-web && pnpm -F @evolyn.do/web typecheck && pnpm -F @evolyn.do/web build`；联调用 `deploy/docker-compose.yaml` 起依赖。
- 跨端接口：字段名、错误码、鉴权头在 evolyn-core 控制器与前端调用侧两侧核对（前端 HTTP 层已随登录批落地：`apps/web/src/api/`，开发期 Vite 将 `/api` 代理到本地 8080）。

## 文档约定

- `docs/低代码平台/`：平台架构设计专题，含设计基线（第 1–21 章选型、第 22 章起顶层定版）与 README 导航；重大技术决策以追加 ADR 方式记录在第 27 章，不回改已定版结论原文，被取代表述在原章节就地标注指向关系。
- 其他业务文档按子目录组织并配 `README.md` 导航，不散落根目录。
- 移动或改名文档用 `git mv`，同一次改动内修复引用；整理只调结构，不改写业务结论。

## 文档维护

- 新增子项目、脚本、关键目录或里程碑落地时，同步更新本文件与 `docs/低代码平台/README.md` 的实现位置映射。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
