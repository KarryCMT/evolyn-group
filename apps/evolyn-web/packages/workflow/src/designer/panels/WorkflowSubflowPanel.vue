<script setup lang="ts">
import { ElAlert, ElForm, ElFormItem, ElInput } from 'element-plus';
import { computed } from 'vue';
import type { WorkflowNode } from '../../schema';

defineOptions({ name: 'WorkflowSubflowPanel' });

const props = defineProps<{ node: WorkflowNode }>();
const emit = defineEmits<{ updateConfig: [config: WorkflowNode['config']] }>();
const config = computed(() => props.node.config.subflow ?? { definitionCode: '' });

function setDefinitionCode(definitionCode: string) {
  emit('updateConfig', { ...props.node.config, subflow: { definitionCode } });
}
</script>

<template>
  <div class="workflow-subflow-panel">
    <ElAlert
      title="子流程运行能力正在接入；当前配置可保存为设计草稿，暂不可启用流程。"
      type="warning"
      :closable="false"
      show-icon
    />
    <ElForm label-position="top" @submit.prevent>
      <ElFormItem label="目标流程" required>
        <ElInput
          :model-value="config.definitionCode"
          placeholder="输入目标流程编码"
          @update:model-value="setDefinitionCode"
        />
      </ElFormItem>
      <ElFormItem label="版本策略">
        <ElInput model-value="发起时使用目标流程启用版本" disabled />
      </ElFormItem>
    </ElForm>
  </div>
</template>

<style scoped lang="scss">
.workflow-subflow-panel {
  display: grid;
  padding: 0 var(--el-space-md) var(--el-space-md);
  gap: var(--el-space-md);
}
</style>
