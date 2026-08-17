# 组件开发指南

本指南将帮助你在这个组件库中开发高质量的 Vue 组件。

## 🏗️ 组件结构

每个组件应该包含以下文件：

```
ComponentName/
├── ComponentName.vue          # 组件主文件
├── ComponentName.types.ts     # TypeScript 类型定义
├── ComponentName.scss         # 样式文件
├── index.ts                   # 导出文件
└── __tests__/
    └── ComponentName.spec.ts  # 测试文件
```

## 📝 组件模板

### 1. Vue 组件文件

```vue
<template>
  <div
    class="v-component-name"
    :class="[
      type ? `v-component-name--${type}` : '',
      size ? `v-component-name--${size}` : '',
      {
        'is-disabled': disabled,
        'is-active': active,
      },
    ]"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import type { ComponentNameEmits, ComponentNameProps } from './ComponentName.types';

defineOptions({
  name: 'VComponentName',
});

const props = withDefaults(defineProps<ComponentNameProps>(), {
  type: 'default',
  size: 'medium',
  disabled: false,
});

const emit = defineEmits<ComponentNameEmits>();

// 组件逻辑
</script>

<style lang="scss">
@use './ComponentName.scss' as *;
</style>
```

### 2. 类型定义文件

```typescript
export interface ComponentNameProps {
  /**
   * 组件类型
   */
  type?: 'default' | 'primary' | 'success' | 'warning' | 'danger';

  /**
   * 组件大小
   */
  size?: 'small' | 'medium' | 'large';

  /**
   * 是否禁用
   */
  disabled?: boolean;
}

export interface ComponentNameEmits {
  /**
   * 点击事件
   */
  click: [event: MouseEvent];

  /**
   * 值改变事件
   */
  change: [value: string];
}
```

### 3. 样式文件

```scss
.v-component-name {
  // 基础样式
  display: inline-block;
  position: relative;

  // 类型变体
  &--primary {
    // primary 样式
  }

  &--success {
    // success 样式
  }

  // 大小变体
  &--small {
    // small 样式
  }

  &--large {
    // large 样式
  }

  // 状态样式
  &.is-disabled {
    // 禁用状态
    opacity: 0.6;
    cursor: not-allowed;
  }

  &.is-active {
    // 激活状态
  }
}
```

### 4. 导出文件

```typescript
import ComponentName from './ComponentName.vue';
import { withInstall } from '../../_utils';

export const VComponentName = withInstall(ComponentName);
export default VComponentName;

export * from './ComponentName.types';
```

### 5. 测试文件

```typescript
import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ComponentName from '../ComponentName.vue';
import type { ComponentNameProps } from '../ComponentName.types';

describe('ComponentName', () => {
  it('renders correctly', () => {
    const wrapper = mount(ComponentName);
    expect(wrapper.classes()).toContain('v-component-name');
  });

  it('applies correct type class', () => {
    const wrapper = mount(ComponentName, {
      props: { type: 'primary' } as ComponentNameProps,
    });
    expect(wrapper.classes()).toContain('v-component-name--primary');
  });

  it('emits events correctly', async () => {
    const wrapper = mount(ComponentName);
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')).toBeTruthy();
  });
});
```

## 🎨 设计规范

### 命名规范

- **组件名**: 使用 PascalCase，如 `Button`、`DatePicker`
- **CSS 类名**: 使用 BEM 规范，如 `.v-button`, `.v-button--primary`, `.v-button__icon`
- **Props**: 使用 camelCase，如 `disabled`, `maxLength`
- **Events**: 使用 camelCase，如 `click`, `change`, `update:modelValue`

### CSS 变量

使用 CSS 变量来支持主题定制：

```scss
.v-button {
  --v-button-color: var(--color-text);
  --v-button-bg: var(--color-bg);
  --v-button-border: var(--color-border);

  color: var(--v-button-color);
  background-color: var(--v-button-bg);
  border-color: var(--v-button-border);
}
```

### 响应式设计

考虑不同屏幕尺寸的适配：

```scss
.v-component {
  // 移动端
  @media (max-width: 768px) {
    // 移动端样式
  }

  // 平板
  @media (min-width: 769px) and (max-width: 1024px) {
    // 平板样式
  }

  // 桌面端
  @media (min-width: 1025px) {
    // 桌面端样式
  }
}
```

## 🔧 开发最佳实践

### 1. Props 设计

- 提供合理的默认值
- 使用 TypeScript 类型约束
- 添加详细的 JSDoc 注释
- 考虑向后兼容性

### 2. 事件处理

- 使用描述性的事件名称
- 传递有用的事件参数
- 支持事件修饰符

### 3. 插槽设计

- 提供有意义的默认内容
- 使用作用域插槽传递数据
- 考虑插槽的可扩展性

### 4. 可访问性

- 添加适当的 ARIA 属性
- 支持键盘导航
- 确保颜色对比度符合标准
- 提供屏幕阅读器支持

### 5. 性能优化

- 使用 `v-memo` 优化重复渲染
- 合理使用 `computed` 和 `watch`
- 避免不必要的响应式数据
- 考虑组件的懒加载

## 🧪 测试策略

### 单元测试

- 测试组件的基本渲染
- 测试 props 的正确应用
- 测试事件的正确触发
- 测试边界情况

### 集成测试

- 测试组件间的交互
- 测试复杂的用户操作流程
- 测试数据流的正确性

### 可访问性测试

- 使用 `@testing-library/vue` 进行可访问性测试
- 测试键盘导航
- 测试屏幕阅读器兼容性

## 📚 文档编写

### API 文档

为每个组件编写详细的 API 文档：

```markdown
## Props

| 属性名 | 类型   | 默认值    | 说明     |
| ------ | ------ | --------- | -------- |
| type   | string | 'default' | 按钮类型 |
| size   | string | 'medium'  | 按钮大小 |

## Events

| 事件名 | 参数                | 说明       |
| ------ | ------------------- | ---------- |
| click  | (event: MouseEvent) | 点击时触发 |

## Slots

| 插槽名  | 说明     |
| ------- | -------- |
| default | 按钮内容 |
```

### 使用示例

提供清晰的使用示例：

```vue
<template>
  <VButton type="primary" @click="handleClick"> 点击我 </VButton>
</template>
```

## 🚀 发布流程

1. 确保所有测试通过
2. 更新组件文档
3. 添加 changeset 记录
4. 提交 Pull Request
5. 代码审查通过后合并

遵循这些指南将帮助你创建高质量、一致性的组件！
