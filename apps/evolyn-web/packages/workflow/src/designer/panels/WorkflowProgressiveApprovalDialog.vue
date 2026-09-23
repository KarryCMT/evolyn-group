<script setup lang="ts">
import { ElButton, ElDialog, ElOption, ElRadio, ElRadioGroup, ElSelect } from 'element-plus';
import { reactive, watch } from 'vue';
import type {
  WorkflowProgressiveApprovalConfig,
  WorkflowProgressiveEndpoint,
} from '../../schema';

defineOptions({ name: 'WorkflowProgressiveApprovalDialog' });

const props = defineProps<{
  modelValue: boolean;
  value: WorkflowProgressiveApprovalConfig | undefined;
}>();

const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  confirm: [value: WorkflowProgressiveApprovalConfig];
}>();

const draft = reactive<WorkflowProgressiveApprovalConfig>({
  endpoint: 'starter_direct_manager',
  downwardLevels: 0,
});

const END_LEVEL_OPTIONS = Array.from({ length: 11 }, (_, level) => ({
  value: level,
  label: level === 0 ? '最高级部门主管' : `最高级部门向下${level}级主管`,
}));

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return;
    draft.endpoint = props.value?.endpoint ?? 'starter_direct_manager';
    draft.downwardLevels = props.value?.downwardLevels ?? 0;
  },
  { immediate: true },
);

function chooseEndpoint(endpoint: WorkflowProgressiveEndpoint) {
  draft.endpoint = endpoint;
  if (endpoint === 'starter_direct_manager') draft.downwardLevels = 0;
}

function close() {
  emit('update:modelValue', false);
}

function confirm() {
  emit('confirm', {
    endpoint: draft.endpoint,
    downwardLevels: draft.endpoint === 'organization_top_manager' ? draft.downwardLevels : 0,
  });
  close();
}
</script>

<template>
  <ElDialog
    :model-value="modelValue"
    title="设置逐级审批规则"
    width="760px"
    append-to-body
    class="workflow-progressive-dialog"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <p class="workflow-progressive-dialog__description">
      从发起人的直接部门主管开始，逐级向上审批，直至设置的审批终点。
      <a href="#" @click.prevent>帮助文档</a>
    </p>

    <h4>审批终点</h4>
    <ElRadioGroup
      :model-value="draft.endpoint"
      class="workflow-progressive-dialog__options"
      @update:model-value="chooseEndpoint"
    >
      <div class="workflow-progressive-dialog__option">
        <ElRadio value="starter_direct_manager">发起人的</ElRadio>
        <ElSelect model-value="direct" disabled aria-label="发起人审批终点">
          <ElOption value="direct" label="直接部门主管" />
        </ElSelect>
      </div>
      <div class="workflow-progressive-dialog__option">
        <ElRadio value="organization_top_manager">通讯录中的</ElRadio>
        <ElSelect
          :model-value="draft.downwardLevels ?? 0"
          :disabled="draft.endpoint !== 'organization_top_manager'"
          aria-label="通讯录审批终点"
          @update:model-value="(value: number) => (draft.downwardLevels = value)"
        >
          <ElOption
            v-for="option in END_LEVEL_OPTIONS"
            :key="option.value"
            :value="option.value"
            :label="option.label"
          />
        </ElSelect>
      </div>
    </ElRadioGroup>

    <template #footer>
      <ElButton @click="close">取消</ElButton>
      <ElButton type="primary" @click="confirm">确定</ElButton>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.workflow-progressive-dialog {
  &__description {
    margin: 4px 0 28px;
    color: var(--el-text-color-secondary);
    line-height: 1.8;

    a {
      margin-left: 8px;
      color: var(--el-color-primary);
    }
  }

  h4 {
    margin: 0 0 16px;
    color: var(--el-text-color-primary);
    font-size: 15px;
  }

  &__options {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 14px;
  }

  &__option {
    display: grid;
    grid-template-columns: 142px minmax(260px, 420px);
    align-items: center;

    :deep(.el-select) {
      width: 100%;
    }
  }
}
</style>
