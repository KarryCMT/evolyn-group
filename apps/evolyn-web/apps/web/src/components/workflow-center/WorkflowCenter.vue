<script setup lang="ts">
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import type { WorkflowPendingTaskSummaryDto } from '~/types';
import { computed, shallowRef, watch } from 'vue';
import { useWorkflowCenter } from '~/composables/useWorkflowCenter';
import WorkflowCenterList from './WorkflowCenterList.vue';
import WorkflowCenterToolbar from './WorkflowCenterToolbar.vue';
import WorkflowTaskDetailDrawer from './WorkflowTaskDetailDrawer.vue';

defineOptions({ name: 'WorkflowCenter' });

const props = defineProps<{
  scope: WorkflowCenterScope;
  /** 嵌入应用工作区时，个人范围由左侧导航承担。 */
  embedded?: boolean;
  /** 左侧待办菜单选中的流程表单；空串代表全部待办。 */
  formCode?: string;
}>();

const emit = defineEmits<{
  updateScope: [scope: WorkflowCenterScope];
  pendingSummary: [summary: WorkflowPendingTaskSummaryDto | null];
}>();

// 路由是个人视图的唯一事实源；容器只将其投影为数据源输入，不自行维护副本。
const scopeRef = computed(() => props.scope);
const formCodeRef = computed(() => props.formCode ?? '');
const selectedTaskId = shallowRef<number | null>(null);
const {
  visibleItems,
  keyword,
  sortOrder,
  nextCursor,
  status,
  errorMessage,
  pendingSummary,
  refreshAll,
  loadMore,
} = useWorkflowCenter(scopeRef, formCodeRef);

watch(pendingSummary, (summary) => emit('pendingSummary', summary), { immediate: true });

function openTask(taskId: number): void {
  selectedTaskId.value = taskId;
}

function closeTask(): void {
  selectedTaskId.value = null;
}
</script>

<template>
  <main class="workflow-center" aria-label="审批中心">
    <WorkflowCenterToolbar
      :scope="props.scope"
      :keyword="keyword"
      :sort-order="sortOrder"
      :pending-count="formCode ? pendingSummary?.formCounts.find((item) => item.formCode === formCode)?.count : pendingSummary?.total"
      :loading="status === 'loading'"
      :show-scope-navigation="!props.embedded"
      @update-scope="emit('updateScope', $event)"
      @update-keyword="keyword = $event"
      @update-sort-order="sortOrder = $event"
      @refresh="refreshAll"
    />

    <el-result
      v-if="status === 'error'"
      class="workflow-center__result"
      icon="error"
      title="审批任务加载失败"
      :sub-title="errorMessage"
    >
      <template #extra>
        <el-button type="primary" @click="refreshAll">
          重新加载
        </el-button>
      </template>
    </el-result>

    <WorkflowCenterList
      v-else
      :items="visibleItems"
      :loading="status === 'loading'"
      :has-more="Boolean(nextCursor)"
      @open-task="openTask"
      @load-more="loadMore"
    />

    <WorkflowTaskDetailDrawer
      :task-id="selectedTaskId"
      @close="closeTask"
      @changed="refreshAll"
    />
  </main>
</template>

<style scoped lang="scss">
.workflow-center {
  display: flex;
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color);

  &__result {
    flex: 1;
  }
}
</style>
