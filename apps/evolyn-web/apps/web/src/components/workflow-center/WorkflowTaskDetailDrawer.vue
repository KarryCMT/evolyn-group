<script setup lang="ts">
import type { WorkflowTaskDetailDto } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import {
  approveWorkflowTask,
  getWorkflowTask,
  rejectWorkflowTask,
  returnWorkflowTaskToStarter,
} from '~/api/workflow';

defineOptions({ name: 'WorkflowTaskDetailDrawer' });

const props = defineProps<{ taskId: number | null }>();

const emit = defineEmits<{
  close: [];
  changed: [];
}>();

type DetailStatus = 'idle' | 'loading' | 'ready' | 'error';
type TaskAction = 'approve' | 'reject' | 'return-to-starter';

const status = shallowRef<DetailStatus>('idle');
const detail = shallowRef<WorkflowTaskDetailDto | null>(null);
const actionLoading = shallowRef<TaskAction | null>(null);
const errorMessage = shallowRef('审批详情加载失败，请稍后重试');

const canApprove = computed(() => detail.value?.allowedActions.includes('approve') ?? false);
const canReject = computed(() => detail.value?.allowedActions.includes('reject') ?? false);
const canReturn = computed(
  () => detail.value?.allowedActions.includes('return-to-starter') ?? false,
);
const formValueText = computed(() => JSON.stringify(detail.value?.formValues ?? {}, null, 2));

watch(
  () => props.taskId,
  async (taskId, _previous, onCleanup) => {
    const controller = new AbortController();
    onCleanup(() => controller.abort());
    detail.value = null;
    if (!taskId) {
      status.value = 'idle';
      return;
    }
    status.value = 'loading';
    try {
      const result = await getWorkflowTask(taskId);
      if (controller.signal.aborted) return;
      detail.value = result;
      status.value = 'ready';
    } catch {
      if (controller.signal.aborted) return;
      errorMessage.value = '审批详情加载失败，请稍后重试';
      status.value = 'error';
    }
  },
  { immediate: true },
);

async function runAction(action: TaskAction): Promise<void> {
  const taskId = props.taskId;
  if (!taskId || actionLoading.value) return;

  const actionText: Record<TaskAction, string> = {
    approve: '同意',
    reject: '驳回',
    'return-to-starter': '退回发起人',
  };
  try {
    const { value } = await ElMessageBox.prompt(
      `确认${actionText[action]}该流程？`,
      actionText[action],
      {
        confirmButtonText: actionText[action],
        cancelButtonText: '取消',
        inputPlaceholder: '审批意见（可选）',
        inputValue: '',
      },
    );
    actionLoading.value = action;
    if (action === 'approve') {
      await approveWorkflowTask(taskId, { comment: value });
    } else if (action === 'reject') {
      await rejectWorkflowTask(taskId, { comment: value });
    } else {
      await returnWorkflowTaskToStarter(taskId, { comment: value });
    }
    ElMessage.success(`${actionText[action]}成功`);
    emit('changed');
    emit('close');
  } catch (error) {
    // Element Plus 取消 prompt 的 rejection 不是业务错误，保持安静即可。
    if (error === 'cancel' || error === 'close') return;
    if (error instanceof ApiError && error.errCode === 'WORKFLOW_TASK_NOT_PENDING') {
      ElMessage.warning('该待办已被处理，列表已刷新');
      emit('changed');
      emit('close');
      return;
    }
    ElMessage.error(`${actionText[action]}失败，请稍后重试`);
  } finally {
    actionLoading.value = null;
  }
}
</script>

<template>
  <el-drawer
    :model-value="props.taskId !== null"
    size="min(620px, 92vw)"
    title="流程审批"
    @close="emit('close')"
    @update:model-value="(visible) => !visible && emit('close')"
  >
    <section v-if="status === 'loading'" v-loading="true" class="workflow-task-detail__state" />

    <el-result
      v-else-if="status === 'error'"
      class="workflow-task-detail__state"
      icon="error"
      title="加载失败"
      :sub-title="errorMessage"
    />

    <section v-else-if="detail" class="workflow-task-detail">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="任务编号">
          #{{ detail.task.id }}
        </el-descriptions-item>
        <el-descriptions-item label="当前节点">
          {{ detail.nodeKey }}
        </el-descriptions-item>
        <el-descriptions-item label="流程状态">
          {{ detail.instance.status }}
        </el-descriptions-item>
        <el-descriptions-item label="发起时间">
          {{
            detail.instance.createdAt
          }}
        </el-descriptions-item>
      </el-descriptions>

      <section class="workflow-task-detail__section">
        <h3 class="workflow-task-detail__section-title">
          表单数据
        </h3>
        <!-- 结构化数据以文本节点输出，绝不通过 v-html 渲染业务字段。 -->
        <pre class="workflow-task-detail__values">{{ formValueText }}</pre>
      </section>

      <section class="workflow-task-detail__section">
        <h3 class="workflow-task-detail__section-title">
          流转记录
        </h3>
        <el-timeline v-if="detail.operations.length">
          <el-timeline-item
            v-for="operation in detail.operations"
            :key="operation.id"
            :timestamp="operation.createdAt"
          >
            {{ operation.type }}
          </el-timeline-item>
        </el-timeline>
        <el-empty v-else :image-size="72" description="暂无流转记录" />
      </section>
    </section>

    <template #footer>
      <div class="workflow-task-detail__actions">
        <el-button @click="emit('close')">
          关闭
        </el-button>
        <el-button
          v-if="canReturn"
          :loading="actionLoading === 'return-to-starter'"
          @click="runAction('return-to-starter')"
        >
          退回
        </el-button>
        <el-button
          v-if="canReject"
          type="danger"
          plain
          :loading="actionLoading === 'reject'"
          @click="runAction('reject')"
        >
          驳回
        </el-button>
        <el-button
          v-if="canApprove"
          type="primary"
          :loading="actionLoading === 'approve'"
          @click="runAction('approve')"
        >
          同意
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<style scoped lang="scss">
.workflow-task-detail {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-xl);

  &__state {
    min-height: 300px;
  }

  &__section {
    min-width: 0;
  }

  &__section-title {
    margin: 0 0 var(--el-space-md);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
  }

  &__values {
    max-height: 280px;
    margin: 0;
    padding: var(--el-space-md);
    overflow: auto;
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    font-family: var(--el-font-family);
    font-size: var(--el-font-size-small);
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
  }

  &__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--el-space-sm);
  }
}
</style>
