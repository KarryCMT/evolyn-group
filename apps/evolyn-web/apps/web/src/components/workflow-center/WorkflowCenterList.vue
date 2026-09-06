<script setup lang="ts">
import type { WorkflowCenterListItem } from '~/composables/useWorkflowCenter';
import { RiNodeTree, RiUser3Fill } from '@remixicon/vue';

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
        <span class="workflow-center-list__header">
          <strong class="workflow-center-list__title">{{ item.title }}</strong>
          <span class="workflow-center-list__subtitle"><RiNodeTree aria-hidden="true" />{{ item.subtitle }}</span>
          <el-tag v-if="item.status !== 'PENDING'" size="small" effect="plain">{{ item.status }}</el-tag>
          <time>{{ formatDate(item.createdAt).slice(0, 16) }}</time>
        </span>
        <span class="workflow-center-list__body">
          <span class="workflow-center-list__starter">
            <span class="workflow-center-list__avatar"><RiUser3Fill aria-hidden="true" /></span>
            {{ item.starterName || '发起人信息未提供' }}
          </span>
          <span class="workflow-center-list__fields">
            <span v-for="(field, index) in item.summaryFields" :key="index" class="workflow-center-list__field">
              <span class="workflow-center-list__label">{{ field.label }}：</span>
              <span class="workflow-center-list__value">{{ field.value }}</span>
            </span>
            <span class="workflow-center-list__field">
              <span class="workflow-center-list__label">流程单号：</span>{{ item.instanceNo || '—' }}
            </span>
          </span>
        </span>
      </button>
    </template>

    <div v-if="props.hasMore" class="workflow-center-list__more">
      <el-button text type="primary" @click="emit('loadMore')">
        加载更多
      </el-button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.workflow-center-list {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 0 20px 24px;

  &__loading, &__empty { min-height: 320px; }
  &__item {
    display: flex;
    flex-direction: column;
    width: 100%;
    margin-bottom: 16px;
    padding: 0;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-bg-color);
    box-shadow: 0 1px 4px rgb(0 0 0 / 6%);
    color: var(--el-text-color-primary);
    font: inherit;
    text-align: left;
    cursor: pointer;
    overflow: hidden;
    &:hover:not(:disabled) { border-color: var(--el-color-primary-light-5); }
    &:disabled { cursor: default; }
    &:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 2px; }
  }
  &__header {
    display: flex;
    align-items: center;
    gap: 20px;
    width: 100%;
    min-height: 54px;
    padding: 12px 20px;
    box-sizing: border-box;
    border-bottom: 1px solid var(--el-border-color-lighter);
    time { margin-left: auto; flex-shrink: 0; color: var(--el-text-color-secondary); font-size: 13px; }
  }
  &__title { font-size: 16px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__subtitle { display: flex; align-items: center; gap: 6px; color: var(--el-text-color-secondary); }
  &__subtitle svg { width: 16px; height: 16px; flex-shrink: 0; }
  &__body { display: flex; align-items: center; width: 100%; min-height: 114px; padding: 16px 36px; box-sizing: border-box; gap: 28px; }
  &__starter { display: flex; align-items: center; gap: 14px; width: 190px; flex-shrink: 0; }
  &__avatar { display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; flex-shrink: 0; border-radius: 50%; background: var(--el-color-primary-light-8); color: var(--el-color-primary); }
  &__avatar svg { width: 20px; height: 20px; }
  &__fields { display: flex; min-width: 0; flex-direction: column; gap: 8px; }
  &__field { display: flex; min-width: 0; line-height: 22px; }
  &__label { flex-shrink: 0; color: var(--el-text-color-secondary); }
  &__value { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__more { display: flex; padding: 16px; justify-content: center; }
  @media (max-width: 760px) {
    padding: 0 12px 16px;
    &__header { flex-wrap: wrap; gap: 8px 14px; padding: 12px; }
    &__body { padding: 16px; flex-direction: column; align-items: flex-start; gap: 16px; }
    &__fields { width: 100%; }
  }
}
</style>
