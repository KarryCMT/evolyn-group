# evolyn-group Agent Guide

本文件适用于整个仓库。进入子目录工作时，先检查该子目录是否另有更近的 `AGENTS.md`；若没有，则遵守本文件。

## 项目总览

本仓库是企业级低代码平台（对标简道云形态）的单仓工程，由 Kubernetes 管理平台 weave 二次演进而来。整体架构与演进路线见 `docs/低代码平台/企业级低代码平台技术架构设计.md`（M0–M7 里程碑，当前处于 M0 完成后的阶段）。

- `apps/evolyn-core/`: Go 1.25 + Gin 后端，模块名保持 `github.com/qingwave/weave`。当前承载认证（JWT + OAuth）、用户/分组、自定义 RBAC；低代码引擎（Schema/Query/Permission 等）按里程碑逐步落地。
- `apps/evolyn-web/`: Vue 3 + Vite 前端，Element Plus + Tailwind，JavaScript 存量 + TypeScript 渐进迁移（`vue-tsc` 已接入）。
- `services/`、`packages/`: 规划目录（Java/Flowable 工作流、OpenAPI 契约），落地前不要创建同名内容。
- `deploy/`: `docker-compose.yaml` 一键起 PostgreSQL/Redis/MinIO。
- `.github/workflows/ci.yml`: CI（后端 build/vet/test/gofmt，前端 typecheck/build）。

分支约定：`2.0.0` 为基线分支，`3.0.0` 为当前开发分支。

## 通用约束

- 不要跨子项目混装依赖。前端以 `apps/evolyn-web/package-lock.json` 为准使用 npm；Go 命令在 `apps/evolyn-core` 内执行（根 `go.work` 已纳入）。
- 不要提交或手改生成/构建产物：`node_modules/`、`dist/`、`bin/`、`cover.out`、`coverage.txt`、swagger 生成的 `apps/evolyn-core/docs/`、`auto-imports.d.ts`、`components.d.ts`。
- 证书与私钥不入库（`.gitignore` 已拦截 `certs/`、`*.key` 等）；本地证书用 `apps/evolyn-core/scripts/cert.sh` 生成。
- 真实密钥、token、生产连接串不写入代码或文档。`config/app.yaml` 是本地开发默认值，生产配置通过 `config/app.example.yaml` 复制后填写，不入库。
- 后端模块导入路径是 `github.com/qingwave/weave/...`，应用名 evolyn 只是目录名，不要重命名 go module 或批量改导入路径。
- 业务代码一律放 `internal/`（编译器强制外部不可导入）；`pkg/` 只放可复用的非业务库（authentication、common、utils、version）。后续引擎代码放 `internal/engine/`，禁止 import gin/gorm。
- 修改 API 字段、路由、权限模型时，同步检查 `evolyn-web/src`（router、views、axios）与后端 swagger。
- 遇到工作区已有未提交改动时，默认视为用户改动；只处理当前任务相关文件，不回滚无关文件。
- 改动的代码需要补充相关注释，简单逻辑简要注释，复杂逻辑详细注释。注释可使用中文。

## evolyn-core 后端

技术栈与入口：

- Go 1.25，Gin + GORM（Postgres）+ go-redis + swaggo。
- 入口 `cmd/api/main.go`，默认配置 `config/app.yaml`（可用 `--config` 覆盖）。
- 关键配置段：`server`（端口 8080、jwtSecret、rateLimits）、`db`（库名 `evolyn`，`migrate: true` 表示启动时 GORM 自动迁移）、`redis`、`oauth`。

目录职责：

```text
cmd/api/            入口，只做装配
internal/
  config/           配置解析
  server/           HTTP 服务器装配与路由注册
  controller/       HTTP 控制器（auth/user/group/rbac），业务下沉 service
  service/          业务逻辑，接口在 interface.go
  repository/       数据访问（GORM），接口在 interface.go
  model/            实体与请求/响应模型
  middleware/       认证/鉴权/限流/CORS/日志/trace
  database/         postgres/redis 客户端
  authorization/    自定义 RBAC 鉴权（M1 将替换为 Casbin）
  metrics/          监控指标
pkg/
  authentication/   JWT 与 OAuth（github/wechat）
  common/           通用响应/上下文/分页
  utils/            ratelimit/request/set/trace
  version/          版本信息（Makefile ldflags 注入，路径勿动）
scripts/            db.sql（冷启动初始化）、cert.sh（本地证书）
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
make ui             # 起前端 ../evolyn-web
```

后端改动规则：

- 遵循 controller → service → repository 分层，新增资源同步补各层 `interface.go` 与装配（`internal/server/server.go`）。
- 数据库结构变更依赖 GORM migrate（model 定义），`scripts/db.sql` 仅作冷启动初始化，两者保持一致。
- 提交前至少运行 `go test ./...`（或说明未运行原因），格式化使用 `make fmt`。

## evolyn-web 前端

技术栈与入口：

- Vue 3 + Vite，Element Plus 按需自动导入（`unplugin-*`，勿手改生成文件），Tailwind CSS + Sass。
- 入口 `src/main.js`；环境变量前缀 `WEAVE_`：`WEAVE_PORT`（默认 8081）、`WEAVE_SERVER`（`/api` 代理目标）、`WEAVE_MOCK`（mock 数据）、`WEAVE_BASE`（子路径部署）。
- TypeScript 渐进迁移中：存量 JS 允许保留（`allowJs`），新文件一律写 TS，`npm run typecheck` 必须通过。

目录职责：

- `src/axios/index.js`: 统一 axios 封装，页面不要散落裸 axios。
- `src/views/`: 页面（user/auth/home/others 等；设计器页面随里程碑新增）。
- `src/store/`: Pinia 模块。`src/router/index.js`: 路由与守卫。
- `src/mock/`: mockjs 数据源。`src/components/`、`src/utils/`、`src/config.js`。

常用命令（在 `apps/evolyn-web` 内执行）：

```bash
npm install
npm run dev             # 本地开发，/api 代理到 WEAVE_SERVER
npm run mock            # mock 数据开发
npm run typecheck       # vue-tsc 类型检查（CI 强制）
npm run build           # 生产构建
```

前端改动规则：

- 使用 Composition API（`<script setup>`）；新代码用 TS，存量 JS 不强制回改。
- 新增页面同步维护 `src/router/index.js` 与 `src/views/home/Menu.vue` 菜单。
- 数据表格将统一采用 VisActor VTable 生态（见架构文档第 31 章），新列表页不要引入其他表格库。

## 验证建议

- 后端：`cd apps/evolyn-core && go test ./...`，必要时 `make lint`、`make run`。
- 前端：`cd apps/evolyn-web && npm run typecheck && npm run build`；联调用 `deploy/docker-compose.yaml` 起依赖。
- 跨端接口：字段名、错误码、鉴权头在 evolyn-core 与 `evolyn-web/src/axios` 两侧核对。

## 文档约定

- `docs/低代码平台/`：平台架构设计专题，含设计基线（第 1–21 章选型、第 22–32 章顶层定版）与 README 导航；重大技术决策以追加 ADR 方式记录，不回改已定版结论。
- 其他业务文档按子目录组织并配 `README.md` 导航，不散落根目录。
- 移动或改名文档用 `git mv`，同一次改动内修复引用；整理只调结构，不改写业务结论。

## 文档维护

- 新增子项目、脚本、关键目录或里程碑落地时，同步更新本文件与 `docs/低代码平台/README.md` 的实现位置映射。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
