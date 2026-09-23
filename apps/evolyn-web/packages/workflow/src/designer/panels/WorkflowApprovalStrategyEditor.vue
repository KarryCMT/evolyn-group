<script setup lang="ts">
import { RiEditBoxLine } from '@remixicon/vue';
import { ElButton, ElForm, ElFormItem, ElOption, ElSelect } from 'element-plus';
import { computed, ref } from 'vue';
import type {
  WorkflowApprovalStrategy,
  WorkflowNode,
  WorkflowProgressiveApprovalConfig,
} from '../../schema';
import WorkflowProgressiveApprovalDialog from './WorkflowProgressiveApprovalDialog.vue';

defineOptions({ name: 'WorkflowApprovalStrategyEditor' });

const props = defineProps<{ config: WorkflowNode['config'] }>();
const emit = defineEmits<{ updateConfig: [config: WorkflowNode['config']] }>();

const dialogVisible = ref(false);
const strategy = computed(() => props.config.approvalStrategy ?? 'regular');

function switchStrategy(value: WorkflowApprovalStrategy) {
  const patch: WorkflowNode['config'] = { ...props.config, approvalStrategy: value };
  if (value === 'regular') {
    patch.progressiveApproval = undefined;
  } else {
    patch.progressiveApproval ??= { endpoint: 'starter_direct_manager', downwardLevels: 0 };
    dialogVisible.value = true;
  }
  emit('updateConfig', patch);
}

function saveRule(value: WorkflowProgressiveApprovalConfig) {
  emit('updateConfig', { ...props.config, approvalStrategy: 'progressive', progressiveApproval: value });
}
</script>

<template>
  <ElForm class="workflow-approval-strategy" label-position="top" @submit.prevent>
    <ElFormItem label="节点负责人" required>
      <ElSelect
        :model-value="strategy"
        @update:model-value="(value: WorkflowApprovalStrategy) => switchStrategy(value)"
      >
        <ElOption value="regular" label="常规审批" />
        <ElOption value="progressive" label="逐级审批" />
      </ElSelect>
    </ElFormItem>

    <ElButton
      v-if="strategy === 'progressive'"
      class="workflow-approval-strategy__rule"
      @click="dialogVisible = true"
    >
      <span>{{ config.progressiveApproval ? '已设置逐级审批规则' : '设置逐级审批规则' }}</span>
      <RiEditBoxLine />
    </ElButton>
  </ElForm>

  <WorkflowProgressiveApprovalDialog
    v-model="dialogVisible"
    :value="config.progressiveApproval"
    @confirm="saveRule"
  />
</template>

<style scoped lang="scss">
.workflow-approval-strategy {
  :deep(.el-select) {
    width: 100%;
  }

  &__rule {
    display: flex;
    width: 100%;
    margin: -6px 0 18px;
    justify-content: space-between;

    svg {
      width: 18px;
    }
  }
}
</style>
