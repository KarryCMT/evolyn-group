<script setup lang="ts">
import { ElAlert, ElForm, ElFormItem, ElInput } from 'element-plus';
import { computed } from 'vue';
import type { WorkflowNode, WorkflowPluginConfig } from '../../schema';

defineOptions({ name: 'WorkflowPluginPanel' });

const props = defineProps<{ node: WorkflowNode }>();
const emit = defineEmits<{ updateConfig: [config: WorkflowNode['config']] }>();
const config = computed<WorkflowPluginConfig>(
  () => props.node.config.plugin ?? { pluginCode: '', actionCode: '' },
);

function patchPlugin(patch: Partial<WorkflowPluginConfig>) {
  emit('updateConfig', { ...props.node.config, plugin: { ...config.value, ...patch } });
}
</script>

<template>
  <div class="workflow-plugin-panel">
    <ElAlert
      title="插件执行适配器正在接入；当前配置可保存为设计草稿，暂不可启用流程。"
      type="warning"
      :closable="false"
      show-icon
    />
    <ElForm label-position="top" @submit.prevent>
      <ElFormItem label="插件" required>
        <ElInput
          :model-value="config.pluginCode"
          placeholder="输入插件编码"
          @update:model-value="(value: string) => patchPlugin({ pluginCode: value })"
        />
      </ElFormItem>
      <ElFormItem label="插件动作" required>
        <ElInput
          :model-value="config.actionCode"
          placeholder="输入动作编码"
          @update:model-value="(value: string) => patchPlugin({ actionCode: value })"
        />
      </ElFormItem>
    </ElForm>
  </div>
</template>

<style scoped lang="scss">
.workflow-plugin-panel {
  display: grid;
  padding: 0 var(--el-space-md) var(--el-space-md);
  gap: var(--el-space-md);
}
</style>
