# Quick Start

## Introduction

evolyn.do-template is a component library and toolkit template project based on Vue3, consisting of the following parts:

- UI Component Library: Provides commonly used UI components
- Utility Functions: Offers common utility functions
- Hooks: Provides reusable composable functions
- Directives: Offers commonly used directives

## Build Modes

The project provides two build entries:

- `pnpm build`: uses Turborepo to run each package's own build task and keeps output in each package's `dist` directory.
- `pnpm build:gulp`: runs `build/gulpfile.ts` to aggregate `ui`, `utils`, `hooks`, and `directives` into root `dist/<package>`.

Base packages are built with Rolldown, the UI package is built with tsdown, and build configuration files use TypeScript.

## Installation

Install using a package manager:

::: code-group

```bash [npm]
npm install @evolyn.do/ui @evolyn.do/utils @evolyn.do/hooks @evolyn.do/directives
```

```bash [yarn]
yarn add @evolyn.do/ui @evolyn.do/utils @evolyn.do/hooks @evolyn.do/directives
```

```bash [pnpm]
pnpm add @evolyn.do/ui @evolyn.do/utils @evolyn.do/hooks @evolyn.do/directives
```

```bash [bun]
bun add @evolyn.do/ui @evolyn.do/utils @evolyn.do/hooks @evolyn.do/directives
```

:::

## Usage

### UI Components

```ts
// Global import
import { createApp } from 'vue';
import UI from '@evolyn.do/ui';
import '@evolyn.do/ui/style.css';
const app = createApp(App);
app.use(UI);
// Additionally, add the following configuration to tsconfig.json for type hints:
// "types": ["@evolyn.do/ui/global.d.ts"]

// Import on demand
import { Button } from '@evolyn.do/ui';
import '@evolyn.do/ui/style.css';
const app = createApp(App);
app.use(Button);
```

### Utility Functions

```ts
import { isString } from '@evolyn.do/utils';
console.log(isString('hello')); // true
```

### Hooks

```ts
import { useCounter } from '@evolyn.do/hooks';
const { count, increment, decrement } = useCounter();
```

### Directives

```ts
import { vFocus } from '@evolyn.do/directives';
// Global import
app.directive('focus', vFocus);

// Import on demand
import { vFocus } from '@evolyn.do/directives';
app.directive('focus', vFocus);
```
