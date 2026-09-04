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

    <ElInput
      v-model="search"
      class="data-toolbar__search"
      :placeholder="placeholder ?? '搜索数据'"
      clearable
    />
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

  &__search {
    width: min(100%, 280px);
    flex: 0 1 280px;
  }
}

@media (max-width: 900px) {
  .data-toolbar {
    align-items: stretch;
    flex-direction: column-reverse;

    &__search {
      width: 100%;
      flex-basis: auto;
    }
  }
}
</style>
