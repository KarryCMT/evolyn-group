<script setup lang="ts">
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import WorkflowCenter from '~/components/workflow-center/WorkflowCenter.vue';

defineOptions({ name: 'WorkflowCenterPage' });

const route = useRoute();
const router = useRouter();
const validScopes = new Set<WorkflowCenterScope>(['pending', 'started', 'completed', 'cc-to-me']);

const scope = computed<WorkflowCenterScope>(() => {
  const value = String(route.query.scope ?? 'pending');
  return validScopes.has(value as WorkflowCenterScope) ? (value as WorkflowCenterScope) : 'pending';
});

function updateScope(nextScope: WorkflowCenterScope): void {
  if (nextScope === scope.value) return;
  void router.replace({ name: 'workflow-center', query: { scope: nextScope } });
}
</script>

<template>
  <div class="workflow-center-page">
    <TopNavigation
      back-to="/dashboard"
      title="审批中心"
      :show-default-navigation="false"
      surface="surface"
    />
    <WorkflowCenter :scope="scope" @update-scope="updateScope" />
  </div>
</template>

<style scoped lang="scss">
.workflow-center-page {
  display: flex;
  height: 100vh;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color-page);
}
</style>
