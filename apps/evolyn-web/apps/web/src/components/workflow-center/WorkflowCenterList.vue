<script setup lang="ts">
import type { WorkflowCenterListItem } from '~/composables/useWorkflowCenter';
import { RiArrowRightSLine, RiFileList3Line } from '@remixicon/vue';

defineOptions({ name: 'WorkflowCenterList' });

const props = defineProps<{
  items: readonly WorkflowCenterListItem[];
  loading: boolean;
  hasMore: boolean;
}>();

const emit = defineEmits<{
  openTask: [taskId: number];
  loadMore: [];
}>();

function formatDate(value: string): string {
  return value || '--';
}
</script>

<template>
  <section class="workflow-center-list" aria-label="流程任务列表">
    <div v-if="props.loading" v-loading="true" class="workflow-center-list__loading" />

    <el-empty
      v-else-if="props.items.length === 0"
      class="workflow-center-list__empty"
      description="暂无流程任务"
    />

    <template v-else>
      <button
        v-for="item in props.items"
        :key="`${item.source}-${item.id}`"
        class="workflow-center-list__item"
        type="button"
        :disabled="item.source !== 'task'"
        @click="item.taskId && emit('openTask', item.taskId)"
      >
        <span class="workflow-center-list__icon" aria-hidden="true"><RiFileList3Line /></span>
        <span class="workflow-center-list__content">
          <strong class="workflow-center-list__title">{{ item.title }}</strong>
          <span class="workflow-center-list__subtitle">{{ item.subtitle }}</span>
        </span>
        <span class="workflow-center-list__meta">
          <el-tag size="small" effect="plain">{{ item.status }}</el-tag>
          <time>{{ formatDate(item.createdAt) }}</time>
        </span>
        <RiArrowRightSLine v-if="item.source === 'task'" class="workflow-center-list__arrow" />
      </button>

      <div v-if="props.hasMore" class="workflow-center-list__more">
        <el-button text type="primary" @click="emit('loadMore')">
          加载更多
        </el-button>
      </div>
    </template>
  </section>
</template>

<style scoped lang="scss">
.workflow-center-list {
  min-height: 0;
  flex: 1;
  overflow: auto;

  &__loading,
  &__empty {
    min-height: 320px;
  }

  &__item {
    display: flex;
    width: 100%;
    min-height: 86px;
    padding: var(--el-space-md) var(--el-space-xl);
    border: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
    color: inherit;
    text-align: left;
    cursor: pointer;
    background: transparent;
    align-items: center;
    gap: var(--el-space-md);

    &:hover:not(:disabled) {
      background: var(--el-fill-color-lighter);
    }

    &:disabled {
      cursor: default;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary-light-5);
      outline-offset: -2px;
    }
  }

  &__icon {
    display: inline-flex;
    width: 36px;
    height: 36px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-size: 20px;
  }

  &__content {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    gap: var(--el-space-xs);
  }

  &__title,
  &__subtitle,
  &__meta time {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__title {
    color: var(--el-text-color-primary);
  }

  &__subtitle,
  &__meta time {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__meta {
    display: flex;
    width: 180px;
    flex: 0 0 auto;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--el-space-xs);
  }

  &__arrow {
    flex: 0 0 auto;
    color: var(--el-text-color-placeholder);
    font-size: 22px;
  }

  &__more {
    display: flex;
    padding: var(--el-space-lg);
    justify-content: center;
  }
}
</style>
