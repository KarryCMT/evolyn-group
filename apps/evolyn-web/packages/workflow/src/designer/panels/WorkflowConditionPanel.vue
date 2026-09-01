<script setup lang="ts">
import { ElAlert, ElInput, ElRadioButton, ElRadioGroup } from 'element-plus';
import { computed } from 'vue';
import type { WorkflowDocument, WorkflowNode } from '../../schema';

/**
 * 条件分支节点面板：节点自身无配置，分支语义全部在出边上。
 * 每条出边二选一——默认分支（无条件兜底）或携带 Expr 表达式；
 * 表达式上下文：form.*（表单字段）/ starter.*（发起人）/ variables.*（服务节点写入）。
 */
defineOptions({ name: 'WorkflowConditionPanel' });

const props = defineProps<{
  node: WorkflowNode;
  document: WorkflowDocument;
}>();

const emit = defineEmits<{
  updateEdgeCondition: [edgeKey: string, expression: string | null];
}>();

/** 条件出边 + 目标节点名（编辑期定位提示；运行时按声明顺序首个命中） */
const outEdges = computed(() =>
  props.document.edges
    .filter((edge) => edge.source === props.node.key)
    .map((edge) => ({
      key: edge.key,
      expression: edge.condition?.expression ?? '',
      isDefault: edge.condition === undefined || edge.condition === null,
      targetName:
        props.document.nodes.find((node) => node.key === edge.target)?.name ?? edge.target,
    })),
);

/**
 * 切换分支类型：设为默认时清空表达式（该边无条件）；
 * 切回条件时置空串，由即时校验提示补填。
 */
function switchMode(edgeKey: string, mode: 'default' | 'expr') {
  if (mode === 'default') {
    emit('updateEdgeCondition', edgeKey, null);
  } else {
    const edge = outEdges.value.find((item) => item.key === edgeKey);
    emit('updateEdgeCondition', edgeKey, edge?.expression ?? '');
  }
}
</script>

<template>
  <div class="workflow-condition-panel">
    <ElAlert
      class="workflow-condition-panel__tip"
      type="info"
      :closable="false"
      show-icon
      title="按出边声明顺序依次匹配，首个命中的分支生效；全部未命中走默认分支"
    />
    <p class="workflow-condition-panel__hint">
      表达式可用上下文：<code>form.字段名</code>（表单数据）、<code>starter.字段</code>（发起人）、<code>variables.变量</code>（服务节点写入）。
    </p>

    <div v-for="edge in outEdges" :key="edge.key" class="workflow-condition-panel__branch">
      <div class="workflow-condition-panel__branch-head">
        <span class="workflow-condition-panel__branch-target" :title="edge.targetName">
          → {{ edge.targetName }}
        </span>
        <ElRadioGroup
          :model-value="edge.isDefault ? 'default' : 'expr'"
          size="small"
          @update:model-value="(value) => switchMode(edge.key, value as 'default' | 'expr')"
        >
          <ElRadioButton value="expr">条件</ElRadioButton>
          <ElRadioButton value="default">默认</ElRadioButton>
        </ElRadioGroup>
      </div>
      <ElInput
        v-if="!edge.isDefault"
        :model-value="edge.expression"
        placeholder="例如：form.amount > 10000"
        @update:model-value="(value) => emit('updateEdgeCondition', edge.key, value)"
      />
    </div>

    <p v-if="outEdges.length === 0" class="workflow-condition-panel__empty">
      当前节点没有出边：请在画布上从节点锚点拖出连线，再回到这里配置分支条件。
    </p>
  </div>
</template>

<style scoped lang="scss">
.workflow-condition-panel {
  display: flex;
  padding: 0 var(--el-space-md);
  flex-direction: column;
  gap: var(--el-space-sm);

  &__tip {
    border-radius: var(--el-border-radius-base);
  }

  &__hint {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;

    code {
      padding: 0 4px;
      font-family: var(--el-font-family);
      background: var(--el-fill-color-light);
      border-radius: var(--el-border-radius-small);
    }
  }

  &__branch {
    display: flex;
    padding: var(--el-space-sm);
    flex-direction: column;
    gap: var(--el-space-xs);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__branch-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-sm);
  }

  &__branch-target {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__empty {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }
}
</style>
