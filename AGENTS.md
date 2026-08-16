# frog Group Agent Guide

本文件适用于整个仓库。进入子目录工作时，先检查该子目录是否另有更近的 `AGENTS.md`；若没有，则遵守本文件。

## 项目总览

这是一个多端 frog 业务项目，根目录主要负责聚合脚本，不是统一的 pnpm workspace。

- `frog-core/`: GoFrame v2 后端服务，包含 HTTP、队列、定时任务、WebSocket、权限、插件、代码生成和部署配置。
- `frog-web/`: Vue 3 + TypeScript + Vite 管理后台，使用 Naive UI、Pinia、Vue Router、Less、自动导入与组件生成。
- `frog-applet/`: uni-app + Vue 3 + Vite 小程序/H5 应用，主要面向微信小程序，也保留多平台构建脚本。
- `frog_app/`: Flutter 多端 App，Very Good CLI 风格，包含 development/staging/production flavor、bloc 相关依赖、l10n 与内置 mini app 资源。

根目录 `package.json` 只提供跨子应用的便捷命令。依赖安装、锁文件、构建产物都按子应用边界处理。

## 通用约束

- 不要跨子应用混装依赖。`frog-web` 和 `frog-applet` 各自有 `pnpm-lock.yaml`；进入对应目录后再安装或执行脚本。
- 不要提交或手改生成产物，除非任务明确要求。常见生成/构建产物包括 `node_modules/`、`dist/`、`build/`、`.dart_tool/`、Go 生成的 DAO/Entity/Service、`auto-imports.d.ts`、`components.d.ts`。
- 修改接口字段、路由、权限、菜单、模型时，要同步检查前端、小程序、App、后端是否存在同名 API 或模型映射。
- 不要把密钥、真实 token、生产数据库连接、对象存储凭证写入代码或文档。环境变量示例应使用占位值。
- 保持现有编码和风格。部分旧文档存在编码异常，不要为了整理文档而大面积重写业务文件。
- 遇到工作区已有未提交改动时，默认视为用户改动；只处理当前任务相关文件，不回滚无关文件。
- 改动的代码需要补充相关注释，简单逻辑简要注释，复杂逻辑详细注释
- 当前改动`frog_app、frog-applet` 下面任何项目都不允许执行build相关命令，可以执行typecheck和lint命令

## 常用根命令

在根目录可使用这些聚合脚本：

```bash
npm run web:dev
npm run web:build
npm run weixin:dev
npm run weixin:build
npm run go:dev
npm run go:build
npm run go:all
npm run go:dao
npm run app:dev
```

注意：`go:*` 脚本会进入 `frog-core` 并调用 `make`；Makefile 中包含类 Unix 命令，在 Windows 环境下可能需要 Git Bash、WSL 或兼容 shell。

## frog-core 后端

技术栈与入口：

- Go 1.24.4，模块名 `frog`。
- GoFrame v2.9.x，入口为 `main.go`。
- 配置主要位于 `manifest/config/`，运行资源位于 `resource/`，持久化/本地运行数据位于 `storage/`。
- 数据访问层、模型、服务接口大量依赖 GoFrame 代码生成。

主要目录职责：

- `api/`: API 入参/出参定义。
- `internal/controller/`: 控制器层。
- `internal/logic/`: 业务逻辑层。
- `internal/service/`: 服务接口与实现注册。
- `internal/dao/`: 数据访问对象，通常由 `gf gen dao` 生成。
- `internal/model/`: entity、input、output 等模型。
- `internal/router/`: 路由注册。
- `internal/queues/`: 队列任务。
- `internal/crons/`: 定时任务。
- `internal/websocket/`: WebSocket 相关逻辑。
- `manifest/`: 配置、部署、Docker、i18n。
- `resource/`: 静态资源、模板、生成器配置。
- `utility/`: 通用工具库。

常用命令：

```bash
cd frog-core
make deps
make http
make all
make queue
make cron
make build
make dao
make service
make lint
go test ./...
```

后端改动规则：

- 遵循 GoFrame 分层，不要把业务逻辑塞进 controller。
- 新增或变更数据库表后，优先通过 `make dao` 或 `gf gen dao` 生成 DAO/Entity，不要手写生成文件。
- 新增服务接口时遵循 `internal/service` 的现有注册模式，可用 `make service` 生成。
- 路由、权限、Casbin、菜单、演示模式限制通常是联动点；改后台接口时要同步检查。
- 提交前至少运行相关 Go 测试或说明未运行原因；格式化使用 `gofmt`。

## frog-web 管理后台

技术栈与入口：

- Vue 3 + TypeScript + Vite。
- UI 和生态包括 Naive UI、Pinia、Vue Router、VueUse、ECharts、CodeMirror、i18n。
- 入口为 `src/main.ts` 和 `src/App.vue`。
- Vite 别名：`@` 指向 `src/`，`/#/` 指向 `types/`。
- 样式预处理使用 Less，Vite 会注入 `src/styles/var.less`。

主要目录职责：

- `src/api/`: 按业务域拆分的后台 API。
- `src/views/`: 页面视图。
- `src/components/`: 复用组件。
- `src/router/`: 静态路由、动态路由生成、路由守卫。
- `src/store/`: Pinia 状态模块。
- `src/layout/`: 后台布局。
- `src/hooks/`: 组合式逻辑。
- `src/utils/`: 通用工具。
- `src/locale/`: 国际化。
- `build/`: Vite 插件、代理、构建后处理等工程配置。

常用命令：

```bash
cd frog-web
pnpm install
pnpm run dev
pnpm run build
pnpm run lint:eslint
pnpm run lint:prettier
pnpm run lint:stylelint
```

前端改动规则：

- 优先使用 `<script setup>`、组合式 API、现有 hooks 和 store 模式。
- API 文件按业务域放入 `src/api/<domain>/`，不要在页面中散落裸 `axios` 调用。
- 路由改动要检查 `src/router/base.ts`、动态路由生成、权限守卫和菜单来源。
- UI 风格应延续后台管理系统的密度与克制感，避免营销页式大视觉。
- 不要手工维护 `auto-imports.d.ts`、`components.d.ts`，它们由插件生成。
- 构建产物在 `dist/`，不要把构建输出作为源码修改。

## frog-applet 小程序/H5

技术栈与入口：

- uni-app + Vue 3 + Vite。
- 入口为 `src/main.js` 和 `src/App.vue`。
- 页面、分包、tabBar 由 `src/pages.json` 管理。
- Vite 别名：`@` 指向 `src/`。
- 样式包含 `src/uni.scss`、`src/styles/`，页面样式优先延续现有 SCSS/CSS 变量。

主要目录职责：

- `src/apis/`: 小程序端 API 封装。
- `src/pages/`: 页面目录，目前包含 `home`、`circle`、`login`、`merchants`、`publish`、`release`、`user` 等。
- `src/components/`: 业务和通用组件。
- `src/composables/`: 组合式逻辑。
- `src/config/`: 端侧配置。
- `src/constants/`: 常量。
- `src/static/`: 小程序静态资源。
- `src/utils/`: 工具函数。

常用命令：

```bash
cd frog-applet
pnpm install
pnpm run dev:h5
pnpm run dev:mp-weixin
pnpm run build:h5
pnpm run build:mp-weixin
```

小程序改动规则：

- 新增页面必须同步维护 `src/pages.json`，包括路径、分包和 tabBar。
- 跳转路径以 `pages.json` 中真实路径为准，避免复制旧路径导致运行时跳转失败。
- API 调用优先通过 `src/apis/` 的封装，不要在页面中分散处理 base URL、token、错误提示。
- 跨端代码要考虑 H5 与微信小程序能力差异，使用条件编译时保持范围最小。
- 静态资源放入 `src/static/`，避免引入过大的图片或未压缩资源。

## frog_app Flutter App

技术栈与入口：

- Flutter SDK 约束 `^3.41.0`，Dart SDK 约束 `^3.11.0`。
- 使用 Very Good Analysis、bloc/flutter_bloc、dio、webview_flutter、flutter_localizations。
- flavor 入口：
  - `lib/main_development.dart`
  - `lib/main_staging.dart`
  - `lib/main_production.dart`
- 根启动逻辑在 `lib/bootstrap.dart`。

主要目录职责：

- `lib/app/`: App 根组件与应用外壳。
- `lib/auth/`: 认证相关功能。
- `lib/network/`: 网络层。
- `lib/tenant/`: 租户相关功能。
- `lib/mini_app/`: mini app 容器或资源逻辑。
- `lib/l10n/`: 国际化配置与生成产物。
- `assets/mini_app/`: 内置 mini app 静态资源。

常用命令：

```bash
cd frog_app
flutter pub get
flutter run --flavor development --target lib/main_development.dart
flutter run --flavor staging --target lib/main_staging.dart
flutter run --flavor production --target lib/main_production.dart
flutter test
very_good test --coverage --test-randomize-ordering-seed random
dart run bloc_tools:bloc lint .
flutter gen-l10n --arb-dir="lib/l10n/arb"
```

App 改动规则：

- 新功能优先按 feature 目录组织，延续 bloc/Very Good 风格和 `very_good_analysis` 约束。
- 网络请求走 `lib/network/` 的现有封装，不要在页面中直接散落 Dio 初始化。
- 新增文案时维护 ARB 文件，并运行或触发 l10n 生成。
- flavor 相关配置要分别检查 development、staging、production 入口。
- 修改内置 mini app 资源时，确认 `pubspec.yaml` 的 assets 声明覆盖到目标文件。

## 验证建议

按修改范围选择最小但有效的验证：

- 后端逻辑：`cd frog-core && go test ./...`，必要时补充 `make lint` 或目标服务启动验证。
- 管理后台：`cd frog-web && pnpm run build`，样式或 lint 改动补充对应 lint 命令。
- 小程序：`cd frog-applet && pnpm run build:mp-weixin` 或目标平台构建；H5 改动可先用 `pnpm run dev:h5`。
- Flutter App：`cd frog_app && flutter test`，涉及 lint 或 bloc 规则时运行 `dart run bloc_tools:bloc lint .`。
- 跨端接口：后端接口、后台 API、小程序 API、App 网络层都要检查字段名、枚举值、错误码和鉴权逻辑。

## 文档目录规范

`docs/` 使用固定的信息架构，常规文档不得直接散落在根目录：

- `docs/README.md`：全局导航，新增、移动或删除文档时必须同步更新。
- `docs/01-产品资料/`：跨迭代的产品原型、业务模型和长期规划。
- `docs/02-迭代资料/`：明确归属某一期的产品、前端、后端、数据库及交付记录。
- `docs/03-专题资料/`：跨迭代或独立交付的功能专题。
- `docs/04-智乒社社团资料/`：智乒社社团资料，可参考社团信息、组织、宗旨、运营方式。

目录与文件命名规则：

- 迭代目录统一使用 `NN-第N期/`，并以 `README.md` 作为该期产品基线和阅读入口。
- 迭代和专题内部统一使用 `前端/`、`后端/`、`db/`、`交付记录/`；不要再创建 `前端设计/`、`后端设计/`、`数据库设计/` 等同义目录。
- 专题目录必须包含 `README.md`，说明范围、状态、子文档和正式实现位置。
- 文件名应在当前目录上下文中保持简短，例如 `B端设计.md`、`小程序端设计.md`、`功能设计.md`；不要重复添加“恩斯邁智乒社”等项目全称前缀。
- 品牌名称需要出现在标题或正文时统一写作“恩斯邁智乒社”；文件名写 `B端`，正文标题和自然语言写 `B 端`。
- 同一文档只能有一个权威位置。跨迭代复用时使用链接，不要复制一份到另一个目录。
- 验收、实现和交付类历史资料放入对应迭代的 `交付记录/`，保留原始结论，不与设计基线混放。

数据库文档规则：

- `docs/**/db/` 仅保存设计参考 SQL、执行顺序和数据库说明，不作为生产迁移的权威来源。
- 后端正式迁移只放在 `apps/frog-core/docs/migrations/`。设计 SQL 落地后，应在相关 README 或设计文档中链接正式迁移文件。
- 不要在 `docs/` 和后端迁移目录维护两份都宣称可直接上线的 SQL；必须明确“参考设计”和“正式迁移”的区别。

文档变更检查：

- 移动或改名优先使用 `git mv`，并在同一次改动中修复 Markdown 链接、反引号路径和导航入口。
- 调整目录后使用 `rg -n "docs/" docs` 检查旧路径残留，并逐一确认本地 Markdown 链接目标存在。
- 新增顶层分类前，先确认现有三个分类确实无法承载；确需新增时，同时更新 `docs/README.md` 和本节约束。
- 文档整理只调整结构、命名、导航和失效引用，不借机大面积改写业务结论。

## 文档维护

- 新增子应用、运行脚本、代码生成流程或关键目录时，同步更新本文件。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
- 文档应记录可执行的约束和命令，避免只写愿景或重复 README。
