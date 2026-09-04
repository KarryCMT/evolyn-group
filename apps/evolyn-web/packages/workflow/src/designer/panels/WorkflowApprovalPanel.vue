<script setup lang="ts">
import {
  ElCollapse,
  ElCollapseItem,
  ElForm,
  ElFormItem,
  ElInputNumber,
  ElRadioButton,
  ElRadioGroup,
  ElSwitch,
} from 'element-plus';
import { computed } from 'vue';
import {
  WORKFLOW_MAX_JOB_SECONDS,
  type WorkflowActorOptions,
  type WorkflowField,
  type WorkflowNode,
} from '../../schema';
import WorkflowAssigneeEditor from './WorkflowAssigneeEditor.vue';
import WorkflowFieldPermissions from './WorkflowFieldPermissions.vue';

/**
 * 审批节点配置面板：审批模式（单人/或签/会签）、审批人、超时自动动作、
 * 催办与字段权限。驳回策略 V1 固定 terminate（终止型），不提供配置项。
 */
defineOptions({ name: 'WorkflowApprovalPanel' });

const props = defineProps<{
  node: WorkflowNode;
  fields: readonly WorkflowField[];
  actorOptions: WorkflowActorOptions | undefined;
}>();

const emit = defineEmits<{
  updateConfig: [config: WorkflowNode['config']];
}>();

const config = computed(() => props.node.config);

/** 局部更新 config（浅合并，引用替换保证文档不可变更新链路） */
function patchConfig(patch: Partial<WorkflowNode['config']>) {
  emit('updateConfig', { ...config.value, ...patch });
}

const timeoutEnabled = computed(() => config.value.timeout !== undefined);
const reminderEnabled = computed(() => config.value.reminder !== undefined);

function toggleTimeout(enabled: string | number | boolean) {
  patchConfig(
    enabled ? { timeout: { seconds: 3 * 24 * 3600, action: 'approve' } } : { timeout: undefined },
  );
}

function toggleReminder(enabled: string | number | boolean) {
  patchConfig(enabled ? { reminder: { seconds: 24 * 3600 } } : { reminder: undefined });
}
</script>

<template>
  <div class="workflow-approval-panel">
    <ElForm label-position="top" size="default" @submit.prevent>
      <ElFormItem label="审批方式">
        <ElRadioGroup
          :model-value="config.approvalMode ?? 'single'"
          @update:model-value="
            (value) =>
              patchConfig({ approvalMode: value as WorkflowNode['config']['approvalMode'] })
          "
        >
          <ElRadioButton value="single">单人审批</ElRadioButton>
          <ElRadioButton value="or-sign">或签</ElRadioButton>
          <ElRadioButton value="countersign">会签</ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem
        v-if="config.approvalMode === 'countersign'"
        label="通过比例（满足 ⌈人数 × 比例⌉ 即通过）"
      >
        <ElInputNumber
          :model-value="config.passRatio ?? 1"
          :min="0.1"
          :max="1"
          :step="0.1"
          :precision="1"
          controls-position="right"
          @update:model-value="(value) => patchConfig({ passRatio: value ?? 1 })"
        />
      </ElFormItem>
    </ElForm>

    <WorkflowAssigneeEditor
      :spec="config.assignee"
      :actor-options="actorOptions"
      :fields="fields"
      label="审批人"
      @update="(spec) => patchConfig({ assignee: spec })"
    />

    <ElCollapse class="workflow-approval-panel__collapse">
      <ElCollapseItem title="超时处理" name="timeout">
        <div class="workflow-approval-panel__switch-row">
          <span>启用超时自动处理</span>
          <ElSwitch :model-value="timeoutEnabled" @update:model-value="toggleTimeout" />
        </div>
        <template v-if="config.timeout">
          <ElForm label-position="top" size="default" @submit.prevent>
            <ElFormItem label="超时时间（秒，上限 30 天）">
              <ElInputNumber
                :model-value="config.timeout.seconds"
                :min="1"
                :max="WORKFLOW_MAX_JOB_SECONDS"
                controls-position="right"
                @update:model-value="
                  (value) => patchConfig({ timeout: { ...config.timeout!, seconds: value ?? 1 } })
                "
              />
            </ElFormItem>
            <ElFormItem label="超时后动作">
              <ElRadioGroup
                :model-value="config.timeout.action"
                @update:model-value="
                  (value) =>
                    patchConfig({
                      timeout: { ...config.timeout!, action: value as 'approve' | 'reject' },
                    })
                "
              >
                <ElRadioButton value="approve">自动同意</ElRadioButton>
                <ElRadioButton value="reject">自动驳回</ElRadioButton>
              </ElRadioGroup>
            </ElFormItem>
          </ElForm>
        </template>
      </ElCollapseItem>

      <ElCollapseItem title="催办提醒" name="reminder">
        <div class="workflow-approval-panel__switch-row">
          <span>启用单次催办提醒</span>
          <ElSwitch :model-value="reminderEnabled" @update:model-value="toggleReminder" />
        </div>
        <ElForm v-if="config.reminder" label-position="top" size="default" @submit.prevent>
          <ElFormItem label="提醒时间（秒，任务创建后计时，上限 30 天）">
            <ElInputNumber
              :model-value="config.reminder.seconds"
              :min="1"
              :max="WORKFLOW_MAX_JOB_SECONDS"
              controls-position="right"
              @update:model-value="(value) => patchConfig({ reminder: { seconds: value ?? 1 } })"
            />
          </ElFormItem>
        </ElForm>
      </ElCollapseItem>

      <ElCollapseItem title="字段权限" name="permissions">
        <WorkflowFieldPermissions
          :fields="fields"
          :form-permissions="config.formPermissions"
          @update="(permissions) => patchConfig({ formPermissions: permissions })"
        />
      </ElCollapseItem>
    </ElCollapse>
  </div>
</template>

<style scoped lang="scss">
.workflow-approval-panel {
  display: flex;
  min-height: 0;
  flex-direction: column;

  &__collapse {
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 0;

    :deep(.el-collapse-item__content) {
      padding-bottom: var(--el-space-sm);
    }
  }

  &__switch-row {
    display: flex;
    margin-bottom: var(--el-space-sm);
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }
}
</style>
