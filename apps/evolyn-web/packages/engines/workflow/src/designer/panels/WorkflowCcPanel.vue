<script setup lang="ts">
import { ElAlert } from 'element-plus';
import type { WorkflowActorOptions, WorkflowField, WorkflowNode } from '../../schema';
import WorkflowAssigneeEditor from './WorkflowAssigneeEditor.vue';

/** 抄送节点面板：仅配置抄送对象（recipients），节点瞬时完成不阻塞流转。 */
defineOptions({ name: 'WorkflowCcPanel' });

defineProps<{
  node: WorkflowNode;
  fields: readonly WorkflowField[];
  actorOptions: WorkflowActorOptions | undefined;
}>();

const emit = defineEmits<{
  updateConfig: [config: WorkflowNode['config']];
}>();
</script>

<template>
  <div class="workflow-cc-panel">
    <ElAlert
      class="workflow-cc-panel__tip"
      type="info"
      :closable="false"
      show-icon
      title="抄送仅通知，不影响流程流转：节点到达即写入抄送记录并继续推进"
    />
    <WorkflowAssigneeEditor
      :spec="node.config.recipients"
      :actor-options="actorOptions"
      :fields="fields"
      label="抄送对象"
      @update="(spec) => emit('updateConfig', { ...node.config, recipients: spec })"
    />
  </div>
</template>

<style scoped lang="scss">
.workflow-cc-panel {
  display: flex;
  padding: 0 var(--el-space-md);
  flex-direction: column;
  gap: var(--el-space-sm);

  &__tip {
    border-radius: var(--el-border-radius-base);
  }
}
</style>
