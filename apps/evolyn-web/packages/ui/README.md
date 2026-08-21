# @evolyn.do/ui

Vue 3 组件库，基于 Vue 3 + TypeScript 构建的现代化组件库。

## 特性

- 🚀 基于 Vue 3 + TypeScript 构建
- 📦 支持按需引入
- 💪 使用 Monorepo + pnpm 工作区管理
- 📝 完整的类型定义
- 🔧 完善的开发工具链

## 安装

```bash
npm install @evolyn.do/ui

yarn add @evolyn.do/ui

pnpm add @evolyn.do/ui
```

## 快速开始

### 全局引入

```ts
// main.ts
import { createApp } from 'vue';
import App from './App.vue';

import VUI from '@evolyn.do/ui';
import '@evolyn.do/ui/style.css';

const app = createApp(App);
app.use(VUI);
app.mount('#app');
```

### 按需引入

```ts
// main.ts
import { createApp } from 'vue';
import App from './App.vue';

import { Button } from '@evolyn.do/ui';
import '@evolyn.do/ui/style.css';

const app = createApp(App);
app.use(Button);
app.mount('#app');
```

## 使用示例

```vue
<template>
  <VButton @click="open = true">弹窗</VButton>
  <VButton type="primary">按钮</VButton>
  <VButton type="success">按钮</VButton>
  <VButton type="warning">按钮</VButton>
  <VButton type="danger">按钮</VButton>
  <VButton type="info">按钮</VButton>
  <VDialog v-model:open="open">
    <div>弹窗测试2222</div>
  </VDialog>
</template>

<script setup lang="ts">
import { VButton, VDialog } from '@evolyn.do/ui';
import { ref } from 'vue';
const open = ref(false);
</script>
```

## EvolynTable 数据表格

基于 [VisActor VTable](https://visactor.io/vtable) 的 ListTable 封装，视觉默认跟随
Element Plus 主题（运行时读取 `--el-*` CSS 变量，暗色切换后更新 `theme` prop 即可）。
平台数据列表统一使用本组件，不要再引入其他表格库。

```vue
<template>
  <EvolynTable :columns="columns" :records="records" @click-cell="onCellClick" />
</template>

<script setup lang="ts">
import { EvolynTable, type EvolynTableColumn } from '@evolyn.do/ui';

const columns: EvolynTableColumn[] = [
  { field: 'name', title: '名称', sortable: true },
  // 声明式富单元格：文字/图片/圆形等元素组合（色值需传具体值，画布不识别 CSS 变量）
  {
    field: 'name',
    title: '成员',
    customRender: () => ({
      expectedWidth: 160,
      expectedHeight: 24,
      elements: [
        { type: 'circle', x: 12, y: 12, radius: 12, fill: '#409eff' },
        { type: 'text', x: 32, y: 12, text: '张三', textBaseline: 'middle' },
      ],
    }),
  },
  { field: 'created_at', title: '创建时间', format: (record) => String(record['created_at']) },
];

const records = [{ name: '张三', created_at: '2026-01-01 00:00:00' }];

function onCellClick(args: unknown) {
  console.log(args);
}
</script>
```

要点：

- **数据更新分两级**：`records` 数组引用变化走 `setRecords` 增量刷新；`columns` /
  `options` 变化才全量 `updateOption`。请始终以新数组引用替换数据（`list = [...list]`
  或重新赋值），原地 push 同一数组不会触发刷新。
- **options 逃生舱**：`options` prop 接收 VTable `ListTableConstructorOptions` 的其余
  字段（冻结列、树表、编辑等），`columns`/`records`/`theme` 由组件统一管理。
- **事件**：VTable 事件以烤串命名转发（`click_cell` → `@click-cell`），完整清单见
  `src/components/EvolynTable/events.ts`。
- **实例**：`ref` 获取组件后调用 `getTable()` 可拿到底层 ListTable 实例（导出、滚动
  定位等长尾能力）。
- 组件高度默认 `100%`，父容器需提供确定高度（flex 布局记得配 `min-height: 0`）。
