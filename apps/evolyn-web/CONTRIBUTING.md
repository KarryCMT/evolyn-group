# 贡献指南

感谢你对这个项目的关注！我们欢迎任何形式的贡献。

## 🚀 快速开始

### 环境要求

- Node.js >= 20，推荐使用 Node.js 22.18+ 或 24+ 以匹配最新构建工具链
- pnpm >= 9

### 本地开发

1. Fork 并克隆仓库

```bash
git clone https://github.com/your-username/vue3-turbo-component-lib-template.git
cd vue3-turbo-component-lib-template
```

2. 安装依赖

```bash
pnpm install
```

3. 启动开发环境

```bash
# 启动所有包的开发环境
pnpm dev

# 或者启动特定的应用
pnpm dev:docs    # 启动文档
pnpm dev:play    # 启动 playground
```

## 📝 开发规范

### 代码规范

项目使用以下工具确保代码质量：

- **ESLint**: JavaScript/TypeScript 代码检查
- **Prettier**: 代码格式化
- **Stylelint**: CSS/SCSS 样式检查
- **Commitlint**: Git 提交信息规范

运行代码检查：

```bash
pnpm lint:all
```

### 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

类型包括：

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

示例：

```
feat(ui): add new Button component
fix(utils): resolve array chunk function edge case
docs: update installation guide
```

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
pnpm test

# 运行测试并生成覆盖率报告
pnpm test:coverage

# 运行 UI 测试
pnpm test:ui
```

### 编写测试

- 为新组件添加测试文件：`ComponentName.spec.ts`
- 测试文件应该放在组件目录下的 `__tests__` 文件夹中
- 确保测试覆盖主要功能和边界情况

## 🎨 添加新组件

使用组件生成器快速创建新组件：

```bash
pnpm generate:component ComponentName
```

这会自动创建：

- 组件文件 (`ComponentName.vue`)
- 类型定义 (`ComponentName.types.ts`)
- 样式文件 (`ComponentName.scss`)
- 导出文件 (`index.ts`)
- 测试文件 (`ComponentName.spec.ts`)

## 📚 文档

### 更新文档

文档位于 `apps/docs` 目录：

- 中文文档：`apps/docs/zh/`
- 英文文档：`apps/docs/en/`

添加新组件文档时，请确保：

1. 提供清晰的使用示例
2. 列出所有 props 和 events
3. 包含中英文版本

### 本地预览文档

```bash
pnpm dev:docs
```

## 🔧 构建和发布

### 构建项目

```bash
# 构建所有包
pnpm build

# 聚合构建所有库包到根目录 dist，适合统一收集发布产物
pnpm build:gulp
```

`pnpm build` 会通过 Turborepo 调度各包自身构建任务；`pnpm build:gulp` 会执行 `build/gulpfile.ts`，将 `ui`、`utils`、`hooks`、`directives` 聚合输出到根目录 `dist/<包名>`。

### 版本管理

项目使用 [Changesets](https://github.com/changesets/changesets) 进行版本管理：

1. 添加变更记录：

```bash
pnpm changeset
```

2. 更新版本：

```bash
pnpm changeset:version
```

## 🐛 报告问题

在提交 issue 之前，请：

1. 搜索现有的 issues
2. 使用最新版本重现问题
3. 提供详细的重现步骤
4. 包含相关的错误信息和环境信息

## 💡 功能请求

我们欢迎功能建议！请：

1. 详细描述功能需求
2. 说明使用场景
3. 考虑是否符合项目目标

## 📋 Pull Request 流程

1. Fork 项目
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'feat: add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 创建 Pull Request

### PR 检查清单

- [ ] 代码通过所有测试
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] 遵循代码规范
- [ ] 提交信息符合规范

## 🤝 社区

- 在 GitHub Discussions 中讨论想法
- 在 Issues 中报告 bug
- 通过 PR 贡献代码

感谢你的贡献！🎉
