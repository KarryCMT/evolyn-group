# evolyn-group Agent Guide

本文件适用于整个仓库。进入子目录工作时，先检查该子目录是否另有更近的 `AGENTS.md`；若没有，则遵守本文件。

## 项目总览

这是一个基于开源项目 [weave](https://github.com/qingwave/weave)（Kubernetes 管理平台）二次开发的多容器单仓项目，前后端分别位于 `apps/` 下：

- `apps/evolyn-core/`: Go 1.25 + Gin 后端服务，模块名保持 `github.com/qingwave/weave`。提供认证（JWT + GitHub OAuth）、自定义 RBAC、用户/分组/文章管理、Kubernetes 资源管理与 Pod 终端（WebSocket）、Docker 集成、速率限制与 swagger 文档。
- `apps/evolyn-web/`: Vue 3 + Vite 前端（JavaScript，非 TS），使用 Element Plus（unplugin 自动导入）、Tailwind CSS、Pinia、Vue Router、ECharts、CodeMirror、xterm.js，内置 mockjs 数据源。

根目录聚合内容：

- `go.work` / `go.work.sum`: Go workspace，当前只纳入 `apps/evolyn-core`。所有 `go` 命令应在 `apps/evolyn-core` 内执行。
- `package.json`: 跨子项目的便捷命令。
- `deploy.sh`: SSH 部署脚本（见「已知失效内容」）。
- `docs/`: 后端 swagger 产物（`docs.go`、`swagger.json`、`swagger.yaml`）与 `rbac.png` 架构图。

分支约定：`2.0.0` 为基线分支，`3.0.0` 为当前开发分支。

## 已知失效内容（不要盲目执行）

以下内容继承自旧结构（frog 项目）或上游 weave，与当前目录布局不符，执行前先确认：

- `deploy.sh` 与根 `package.json` 的 `deploy` 脚本：引用 `apps/frog-web`、`apps/frog-core` 和 `gf build`，这些路径与命令在本分支已不存在。
- 根 `package.json` 的 `dev:all`（`make all`）和 `dev:dao`（`make dao`）：`apps/evolyn-core/Makefile` 中没有 `all` 和 `dao` 目标。可用目标见下文。
- `apps/evolyn-core/Makefile` 的 `ui`、`docker-build-ui`、`docker-build-ui-mock`：执行 `cd web`，但 `evolyn-core` 下没有 `web` 目录，前端实际在 `apps/evolyn-web`。
- 后端源码中存在上游拼写遗留：`pkg/service/interfacae.go`、`pkg/repository/repositroy.go`。保持上游命名，除非任务明确要求，不要顺手改名（会引入无谓 diff）。

## 通用约束

- 不要跨子项目混装依赖。前端以 `apps/evolyn-web/package-lock.json` 为准使用 npm；不要在子项目外执行安装。
- 不要提交或手改生成/构建产物：`node_modules/`、`dist/`、`bin/`、`cover.out`、`coverage.txt`、swagger 生成的 `docs/` 内容、`auto-imports.d.ts`、`components.d.ts`。
- 不要把密钥、真实 token、生产凭证写入代码或文档。`apps/evolyn-core/config/app.yaml` 中的 jwtSecret、OAuth clientSecret、数据库/Redis 密码目前为本地开发默认值，接入真实环境时通过配置覆盖，不要把真实值写回文件。
- 后端模块导入路径是 `github.com/qingwave/weave/...`，应用名 evolyn 只是目录名，不要重命名 go module 或批量改导入路径。
- 修改 API 字段、路由、权限模型时，同步检查 `evolyn-web/src/axios/index.js`、views 与 store 中的调用映射。
- 遇到工作区已有未提交改动时，默认视为用户改动；只处理当前任务相关文件，不回滚无关文件。
- 改动的代码需要补充相关注释，简单逻辑简要注释，复杂逻辑详细注释。注释可使用中文。

## evolyn-core 后端

技术栈与入口：

- Go 1.25，Gin（`gin-gonic/gin` v1.11）+ GORM（Postgres）+ go-redis v9 + gorilla/websocket + swaggo。
- 入口 `main.go`，默认配置 `config/app.yaml`（可用 `--config` 覆盖）。
- 关键配置段：`server`（端口 8080、jwtSecret、rateLimits）、`docker`（docker.sock）、`kubernetes`（watchResources）、`db`（库名 `evolyn`，`migrate: true` 表示启动时 GORM 自动迁移）、`redis`、`oauth.github`。

分层与目录职责（均在 `pkg/` 下）：

- `controller/`: HTTP 控制器，按资源拆分（auth、user、group、post、rbac、container、kubecontroller 等），只做参数绑定与响应，业务逻辑下沉 service。
- `service/`: 业务逻辑层，接口定义在 `interfacae.go`。
- `repository/`: 数据访问层，接口定义在 `interface.go`，基于 GORM。
- `model/`: 请求/响应与实体模型。
- `authentication/`: JWT 签发校验与 GitHub OAuth。
- `authorization/`: 自定义 RBAC 鉴权（非 Casbin）。
- `library/kubernetes`、`library/docker`: K8s 客户端、informer watch 与 Docker 封装。
- `middleware/`、`server/`: 中间件与服务器装配。
- `scripts/db.sql`: Postgres 初始化脚本；`cert.sh` 生成本地证书。
- `.golangci.yaml`: lint 配置（gofmt、goimports、gocyclo 等）。

常用命令（在 `apps/evolyn-core` 内执行）：

```bash
make run            # 本地运行（需要 kubeconfig/docker/postgres/redis 就绪）
make build          # 构建到 bin/weave（注入版本 ldflags）
make test           # 单测 + 覆盖率
make fmt            # gofmt -s 格式化
make swagger        # swag init 重新生成 API 文档
make init           # 安装 swag、起 postgres/redis 容器并导入 db.sql、装 git hooks
make postgres       # 起本地 postgres 容器并导入 scripts/db.sql
make redis          # 起本地 redis 容器
make exec-db        # 进入 postgres 容器的 evolyn 库
make docker-build-server
golangci-lint run   # lint（已装 golangci-lint 时）
```

后端改动规则：

- 遵循 controller → service → repository 分层，新增资源时同步补 `interface.go`/`interfacae.go` 中的接口定义与注册。
- 新增或修改 API 后运行 `make swagger` 同步文档；路由与 RBAC 权限通常联动，改接口要检查 `authorization` 与前端调用。
- 数据库结构变更优先依赖 GORM migrate（model 定义），`scripts/db.sql` 仅作为冷启动初始化参考；两者保持一致。
- 提交前至少运行 `go test ./...`（或说明未运行原因），格式化使用 `make fmt`。

## evolyn-web 前端

技术栈与入口：

- Vue 3 + Vite 7，JavaScript（非 TypeScript），Element Plus 按需自动导入（`unplugin-auto-import`/`unplugin-vue-components`，勿手改生成文件）。
- 样式以 Tailwind CSS（含 typography 插件）+ Sass 为主。
- 入口 `src/main.js`；环境变量前缀 `WEAVE_`：`WEAVE_PORT`（默认 8081）、`WEAVE_SERVER`（默认 http://localhost:8080，`/api` 代理目标，含 WebSocket）、`WEAVE_MOCK`（启用 mock 数据）、`WEAVE_BASE`（子路径部署，如 `/weave/`）。

目录职责：

- `src/axios/index.js`: 统一 axios 封装与后端 API 调用，不要在页面里散落裸 axios。
- `src/views/`: 页面，按域分目录：`kube`（Pod/终端/Ingress/Namespace/Service）、`container`、`post`、`user`、`auth`、`doc`、`home` 等。
- `src/store/`: Pinia 模块（`kube.js`、`post.js`）。
- `src/router/index.js`: 路由与守卫。
- `src/mock/`: mockjs 数据源，`npm run mock` 或 `build-with-mock` 使用。
- `src/components/`、`src/utils/`、`src/config.js`: 复用组件、工具与端侧配置。

常用命令（在 `apps/evolyn-web` 内执行）：

```bash
npm install
npm run dev             # 本地开发，/api 代理到 WEAVE_SERVER
npm run mock            # 使用 mock 数据开发
npm run build           # 生产构建
npm run build-with-mock # 构建 mock 版本
npm run preview
```

前端改动规则：

- 使用 Composition API（`<script setup>`），延续现有 JS 风格，不引入 TS 迁移。
- API 调用统一走 `src/axios/index.js`，新增接口先在后端 swagger 确认字段。
- Pod 终端相关改动注意 WebSocket 代理配置（vite proxy `ws: true` 与后端 `pkg/controller/kubecontroller`）。
- 部署在子路径时检查 `WEAVE_BASE` 与 `nginx.conf`。

## 验证建议

按修改范围选择最小但有效的验证：

- 后端逻辑：`cd apps/evolyn-core && go test ./...`，必要时 `make fmt`、`golangci-lint run` 或 `make run` 启动验证。
- API 文档：接口变更后 `make swagger`，确认 `docs/` 更新且编译通过。
- 前端：`cd apps/evolyn-web && npm run build`；本地联调可先起 `npm run dev` 配合后端 `make run`。
- 跨端接口：字段名、错误码、鉴权头要在 `evolyn-core` swagger 与 `evolyn-web/src/axios/index.js` 两侧核对。

## 文档约定

- `docs/` 当前只存放后端 swagger 产物和 `rbac.png`，属于生成物，不要手工编辑；业务设计文档如需新增，先在 `docs/` 下建子目录并配 `README.md` 导航，不要散落根目录。
- 移动或改名文档时用 `git mv`，同一次改动内修复引用。
- 文档整理只调整结构、命名和失效引用，不借机改写业务结论。

## 文档维护

- 新增子项目、运行脚本、代码生成流程或关键目录时，同步更新本文件。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
- 修复「已知失效内容」中列出的脚本或 Makefile 目标时，记得同步更新本文件对应条目。
