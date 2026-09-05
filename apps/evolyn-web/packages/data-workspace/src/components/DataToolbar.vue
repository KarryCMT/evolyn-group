<script setup lang="ts">
import { ElInput } from 'element-plus';
import type { DataAction } from '../types.js';

defineOptions({ name: 'DataToolbar' });

const search = defineModel<string>('search', { required: true });

defineProps<{
  actions: readonly DataAction[];
  placeholder?: string;
}>();

const emit = defineEmits<{
  action: [key: string];
}>();
</script>

<template>
  <header class="data-toolbar">
    <div class="data-toolbar__actions" aria-label="数据操作">
      <button
        v-for="action in actions"
        :key="action.key"
        class="data-toolbar__action"
        :class="{
          'data-toolbar__action--primary': action.tone === 'primary',
          'data-toolbar__action--danger': action.tone === 'danger',
        }"
        type="button"
        :disabled="action.disabled"
        @click="emit('action', action.key)"
      >
        <component :is="action.icon" v-if="action.icon" />
        <span>{{ action.label }}</span>
      </button>
    </div>

    <div class="data-toolbar__suffix">
      <!-- 工作台内建能力（如列设置）的挂载位，与业务动作区隔离 -->
      <slot name="suffix" />
      <ElInput
        v-model="search"
        class="data-toolbar__search"
        :placeholder="placeholder ?? '搜索数据'"
        clearable
      />
      <!-- 搜索框之后的工具型入口挂载位（如筛选），保持与参考形态同序：搜索 → 筛选 -->
      <slot name="suffix-end" />
    </div>
  </header>
</template>

<style scoped lang="scss">
.data-toolbar {
  min-height: 58px;
  padding: 8px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__actions {
    display: flex;
    min-width: 0;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
  }

  &__action {
    min-height: 34px;
    padding: 0 10px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    // 与「列设置」按钮同因：压缩时 CJK 文字会逐字折行，强制单行
    white-space: nowrap;
    color: var(--el-text-color-regular);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    font-size: 14px;
    font-weight: 550;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 17px;
      height: 17px;
    }

    &:hover:not(:disabled) {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:disabled {
      color: var(--el-text-color-disabled);
      cursor: not-allowed;
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover:not(:disabled) {
        color: var(--el-color-white);
        background: var(--el-color-primary-light-3);
      }
    }

    &--danger {
      color: var(--el-color-danger);

      &:hover:not(:disabled) {
        color: var(--el-color-danger);
        background: var(--el-color-danger-light-9);
      }
    }
  }

  &__suffix {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 8px;
  }

  &__search {
    width: min(100%, 280px);
    flex: 0 1 280px;
    // 列设置按钮锁定不缩后，suffix 区的收缩集中在搜索框；给出下限，
    // 极窄时改为挤压左侧动作区让其内部换行，而不是把搜索框压扁
    min-width: 160px;
  }
}

@media (max-width: 900px) {
  .data-toolbar {
    align-items: stretch;
    flex-direction: column-reverse;

    &__suffix {
      width: 100%;
    }

    &__search {
      width: 100%;
      flex-basis: auto;
    }
  }
}
</style>
