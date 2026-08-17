**English** | [简体中文](./README.zh-CN.md)

<h1 align="center">Vue3 Turbo Component Library Template 🚀</h1>

<p align="center">
  快速构建属于你的企业级 Vue 3 组件库，基于最新技术栈的 Monorepo 模板
</p>

<p align="center">
  <a href="https://github.com/huangmingfu/vue3-turbo-component-lib-template/stargazers">
    <img src="https://img.shields.io/github/stars/huangmingfu/vue3-turbo-component-lib-template" alt="GitHub stars">
  </a>
  <a href="https://github.com/huangmingfu/vue3-turbo-component-lib-template/issues">
    <img src="https://img.shields.io/github/issues/huangmingfu/vue3-turbo-component-lib-template" alt="GitHub issues">
  </a>
  <a href="https://github.com/huangmingfu/vue3-turbo-component-lib-template/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/huangmingfu/vue3-turbo-component-lib-template" alt="GitHub">
  </a>
  <a href="https://github.com/huangmingfu/vue3-turbo-component-lib-template/network/members">
    <img src="https://img.shields.io/github/forks/huangmingfu/vue3-turbo-component-lib-template" alt="GitHub forks">
  </a>
</p>

[![Vue.js](https://img.shields.io/badge/Vue.js-3.5-42b883?style=flat-square&logo=vue.js&logoColor=white)](https://v3.vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178c6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Turborepo](https://img.shields.io/badge/Turborepo-%23000000?style=flat-square&logo=turborepo&logoColor=white)](https://turbo.build/)
[![pnpm](https://img.shields.io/badge/pnpm-%23F69220?style=flat-square&logo=pnpm&logoColor=white)](https://pnpm.io/)
[![License](https://img.shields.io/github/license/huangmingfu/vue3-turbo-component-lib-template?style=flat-square)](LICENSE)

A modern component library template based on `Turborepo + Vue 3.5 + TypeScript`, using Monorepo architecture to manage multiple packages with pre-configured code standards and development toolchain. This template helps developers focus on component development without dealing with complicated underlying configurations, quickly building their own enterprise-level component library.

> Version note: the current main branch is the latest v2 line. The v1 line is kept on the [`v1`](https://github.com/huangmingfu/vue3-turbo-component-lib-template/tree/v1) branch. This v2 label belongs to the template project itself, while the four example library packages stay at `1.0.0` so users can rename the template and maintain their own initial package versions.

## ✨ Key Features

- 🚀 **Cutting-edge Tech Stack** - Built on Vue 3.5 + TypeScript 6, enjoy the latest features
- 📦 **Monorepo Architecture** - Managed with Turborepo for multiple packages, improving code reusability
- 🚫 **Enforced pnpm** - Resolves phantom dependencies, saves disk space, improves installation speed
- 🎨 **Complete Standards** - Integrated ESLint, Prettier, Stylelint, Commitlint and other code standards
- 📚 **Documentation Support** - Built with VitePress, supports internationalization
- 🔥 **On-demand Import** - Supports Tree Shaking to reduce final bundle size
- 🎯 **Full Type Support** - Comprehensive TypeScript type definitions for better DX
- 🛠️ **Rich Toolset** - Built-in common Hooks, utility functions and directives
- 🔄 **HMR Support** - Real-time preview during development, boosts productivity
- 🔧 **One-click Renaming** - Quickly replace @evolyn.do with your custom package name
- ⚡️ **Fast Library Builds** - Uses Rolldown for base packages and tsdown for the UI package
- 🧩 **Optional Aggregate Build** - Keeps `build:gulp` for collecting all library outputs under root `dist`
- 🧪 **Modern Playground Dependency** - Uses antdv-next and @antdv-next/icons in playground
- 📝 **Version Management** - Uses Changeset for multi-package versioning and release workflow

## 📁 Project Structure

```
├── apps/
│   └── docs/           # Component library documentation based on VitePress
├── build/              # Gulp aggregate build entry powered by Rolldown and tsdown
├── packages/
│   ├── ui/             # UI component library
│   ├── hooks/          # Custom Hooks
│   ├── directives/     # Custom directives
│   ├── utils/          # Utility functions
│   └── lint-configs/   # Code standard configurations
├── playground/         # Component demo environment
└── scripts/            # Helper scripts
```

## 🚀 Quick Start

```bash
# 1. Clone the template
git clone https://github.com/huangmingfu/vue3-turbo-component-lib-template.git

# 2. Install dependencies
pnpm install

# 3. Start development environment
pnpm dev

# 4. Build the project
pnpm build
```

### Install the component library in your project

```bash
# Replace @evolyn.do with your own package name
pnpm add @evolyn.do/ui @evolyn.do/utils @evolyn.do/hooks @evolyn.do/directives

# Example installation
pnpm add @hmflib/ui @hmflib/utils @hmflib/hooks @hmflib/directives
```

## 🧰 Development Commands

```bash
# Development
pnpm dev              # Start development environment for all packages
pnpm dev:docs         # Start documentation application
pnpm dev:play         # Start playground

# Building
pnpm build            # Build all packages
pnpm build:docs       # Build documentation application
pnpm build:gulp       # Aggregate library packages into root dist for unified output collection

# Testing
pnpm test             # Test all packages
pnpm test:ui          # Test UI package
pnpm test:coverage    # Test coverage for all packages

# Code Quality
pnpm lint:all         # Check all code standards
pnpm deps:check       # Check for dependency updates
pnpm deps:update      # Update all dependencies

# Others
pnpm clean            # Clean build artifacts
pnpm rename-pkg       # One-click package renaming
pnpm generate:component # Generate new component
```

## 🎯 Why Choose This Template?

### Advantages over other component library templates:

1. **Cutting-edge Technology** - Based on the latest Vue 3.5 and TypeScript 6, always in sync with technological development
2. **Ready to Use** - Pre-configured complete development toolchain, start developing without additional setup
3. **Flexible Architecture** - Monorepo structure for managing multiple packages while maintaining module independence
4. **Enterprise-grade Standards** - Integrated complete code and commit standards to ensure code quality
5. **Comprehensive Documentation** - Built-in documentation system with internationalization support
6. **Fast Build Pipeline** - Base packages use Rolldown, the UI package uses tsdown, and `build:gulp` aggregates library outputs
7. **Easy Customization** - One-click renaming script to quickly customize to your own component library

## 📸 Preview

### Documentation System

> Supports internationalization

![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202412291431548.png)
![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202411271629728.png)
![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202411271629672.png)

### Playground

> Real-time preview and testing of components

![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202411271630381.png)
![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202411271631563.png)

## 🔧 Tech Stack

- [Vue 3](https://vuejs.org/) - Progressive JavaScript framework
- [TypeScript](https://www.typescriptlang.org/) - Superset of JavaScript with type safety
- [Turborepo](https://turbo.build/repo) - High-performance Monorepo build system
- [VitePress](https://vitepress.dev/) - Vite-powered static site generator
- [Vite](https://vitejs.dev/) - Next-generation frontend build tool
- [Rolldown](https://rolldown.rs/) - High-performance JavaScript/TypeScript bundler
- [tsdown](https://tsdown.dev/) - TypeScript library bundler powered by Rolldown
- [antdv-next](https://github.com/antdv-next/antdv-next) - Vue component library used by playground
- [pnpm](https://pnpm.io/) - Fast, disk space efficient package manager
- [ESLint](https://eslint.org/) - JavaScript/TypeScript code quality tool
- [Prettier](https://prettier.io/) - Code formatter
- [Stylelint](https://stylelint.io/) - CSS/SCSS/Less code linter
- [Commitlint](https://commitlint.js.org/) - Commit message linting tool
- [husky](https://typicode.github.io/husky/) - Git hooks manager
- [lint-staged](https://github.com/lint-staged/lint-staged) - Run linters on git staged files
- [Changesets](https://github.com/changesets/changesets) - Versioning and release management tool

## 📚 Related Links

> Some code and structure design references [Vben5](https://github.com/vbenjs/vue-vben-admin)

- [Vue 3 Official Documentation](https://vuejs.org/)
- [TypeScript Official Documentation](https://www.typescriptlang.org/)
- [Turborepo Official Documentation](https://turbo.build/repo)
- [VitePress Official Documentation](https://vitepress.dev/)
- [Vben Admin](https://github.com/vbenjs/vue-vben-admin)

## ❓ FAQ

### 1. Why recommend pnpm?

pnpm has the following advantages over npm and yarn:

- Solves phantom and doppelganger dependencies issues
- Saves significant disk space
- Faster installation speed
- Ensures consistent node_modules structure

### 2. How to customize package names?

The project uses `@evolyn.do` as the default package name prefix. You can replace it with one command:

```bash
# Replace @evolyn.do with your custom package name, e.g. @yourname
pnpm rename-pkg "@evolyn.do" "@yourname"
```

### 3. How to generate new components?

Use the built-in script to quickly generate new components:

```bash
pnpm generate:component
```

### 4. What to do if command execution fails?

If you encounter issues executing `rm -rf` or other shell commands, please use Git Bash terminal to run the commands.
![](https://huangmingfu.github.io/drawing-bed/images/pic-go/202412251542234.png)

If `pnpm run dev` fails, first execute the build command:

```bash
pnpm run build
pnpm run dev
```

### 5. What is the difference between `pnpm build` and `pnpm build:gulp`?

`pnpm build` uses Turborepo to run each package's own build task and keeps output in each package's `dist` directory. `pnpm build:gulp` runs `build/gulpfile.ts` and aggregates `ui`, `utils`, `hooks`, and `directives` into root `dist/<package>`, which is useful when you want one collected library output directory.

## 🤝 Contributing

Contributions of any kind are welcome!

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

[MIT](LICENSE)

---

⭐ If you like this project, please give it a star! Your support is our motivation for continuous improvement!
