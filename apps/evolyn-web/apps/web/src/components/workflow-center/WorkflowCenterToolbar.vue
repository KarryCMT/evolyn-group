<script setup lang="ts">
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import { RiRefreshLine, RiSearch2Line } from '@remixicon/vue';

defineOptions({ name: 'WorkflowCenterToolbar' });

const props = defineProps<{
  scope: WorkflowCenterScope;
  keyword: string;
  loading: boolean;
  pendingCount?: number;
  sortOrder: 'newest' | 'oldest';
  /** 嵌入应用壳时，范围由左侧个人导航切换，避免重复入口。 */
  showScopeNavigation?: boolean;
}>();

const emit = defineEmits<{
  updateScope: [scope: WorkflowCenterScope];
  updateKeyword: [keyword: string];
  refresh: [];
  updateSortOrder: [value: 'newest' | 'oldest'];
}>();

const scopes: ReadonlyArray<{ value: WorkflowCenterScope; label: string }> = [
  { value: 'pending', label: '我的待办' },
  { value: 'started', label: '我发起的' },
  { value: 'completed', label: '我处理的' },
  { value: 'cc-to-me', label: '抄送我的' },
];
</script>

<template>
  <header class="workflow-center-toolbar">
    <nav
      v-if="props.showScopeNavigation !== false"
      class="workflow-center-toolbar__scopes"
      aria-label="审批任务范围"
    >
      <button
        v-for="item in scopes"
        :key="item.value"
        class="workflow-center-toolbar__scope"
        :class="{ 'workflow-center-toolbar__scope--active': props.scope === item.value }"
        type="button"
        :aria-current="props.scope === item.value ? 'page' : undefined"
        @click="emit('updateScope', item.value)"
      >
        {{ item.label }}
      </button>
    </nav>

    <div v-if="props.scope === 'pending'" class="workflow-center-toolbar__filters" aria-label="待办分类">
      <button type="button" class="workflow-center-toolbar__pill workflow-center-toolbar__pill--active" aria-current="true" @click="emit('updateKeyword', '')">
        全部<span v-if="props.pendingCount !== undefined">（{{ props.pendingCount }}）</span>
      </button>
    </div>
    <div class="workflow-center-toolbar__tools">
      <label class="workflow-center-toolbar__search">
        <RiSearch2Line aria-hidden="true" />
        <input
          :value="props.keyword"
          placeholder="搜索已加载流程信息"
          aria-label="搜索已加载流程信息"
          @input="emit('updateKeyword', ($event.target as HTMLInputElement).value)"
        >
      </label>
      <el-select :model-value="props.sortOrder" class="workflow-center-toolbar__sort" aria-label="已加载任务排序" @update:model-value="emit('updateSortOrder', $event)">
        <el-option label="最新优先" value="newest" />
        <el-option label="最早优先" value="oldest" />
      </el-select>
      <el-button circle :loading="props.loading" aria-label="刷新待办" @click="emit('refresh')">
        <RiRefreshLine aria-hidden="true" />
      </el-button>
    </div>
  </header>
</template>

<style scoped lang="scss">
.workflow-center-toolbar {
  display: flex;
  min-height: 68px;
  padding: 16px 20px;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--el-space-xl);


  &__scopes,
  &__tools,
  &__search {
    display: flex;
    min-width: 0;
    align-items: center;
  }

  &__filters { display: flex; align-items: center; gap: 12px; }
  &__pill { padding: 8px 20px; border: 1px solid var(--el-border-color); border-radius: 24px; font: inherit; cursor: pointer; }
  &__pill--active { color: var(--el-color-white); background: var(--el-color-primary); border-color: var(--el-color-primary); font-weight: 600; }
  &__sort { width: 112px; flex-shrink: 0; }
  &__scopes {
    height: 100%;
    gap: var(--el-space-xl);
  }

  &__scope {
    height: 68px;
    padding: 0 var(--el-space-xs);
    border: 0;
    border-bottom: 3px solid transparent;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    font-size: var(--el-font-size-medium);
    font-weight: 600;

    &--active {
      border-bottom-color: var(--el-color-primary);
      color: var(--el-color-primary);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary-light-5);
      outline-offset: -3px;
    }
  }

  &__tools {
    margin-left: auto;
    flex: 0 1 460px;
    gap: var(--el-space-sm);
  }

  &__search {
    min-width: 0;
    flex: 1;
    gap: var(--el-space-sm);
    padding: 0 var(--el-space-md);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-placeholder);

    &:focus-within {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
    }
  }

  &__search input {
    width: 100%;
    height: 34px;
    min-width: 0;
    border: 0;
    outline: 0;
    color: var(--el-text-color-primary);
    background: transparent;
  }
}
</style>
