<script setup lang="ts">
import { RiDeleteBin6Fill } from '@remixicon/vue';
import { ElButton, ElInput, ElScrollbar, ElTag } from 'element-plus';
import { computed } from 'vue';
import type {
  WorkflowActorOptions,
  WorkflowDocument,
  WorkflowEdge,
  WorkflowField,
  WorkflowNode,
} from '../schema';
import WorkflowApprovalPanel from './panels/WorkflowApprovalPanel.vue';
import WorkflowCcPanel from './panels/WorkflowCcPanel.vue';
import WorkflowConditionPanel from './panels/WorkflowConditionPanel.vue';
import WorkflowEdgePanel from './panels/WorkflowEdgePanel.vue';
import WorkflowServicePanel from './panels/WorkflowServicePanel.vue';

/**
 * 属性面板壳：按选中对象（节点/连线）分发到对应配置面板。
 * 面板只发语义化补丁事件，文档变更统一由 WorkflowDesigner 落到 DSL。
 */
defineOptions({ name: 'WorkflowInspector' });

const props = defineProps<{
  document: WorkflowDocument;
  selectedNode: WorkflowNode | null;
  selectedEdge: WorkflowEdge | null;
  fields: readonly WorkflowField[];
  actorOptions: WorkflowActorOptions | undefined;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  updateNodeName: [nodeKey: string, name: string];
  updateNodeConfig: [nodeKey: string, config: WorkflowNode['config']];
  updateEdgeCondition: [edgeKey: string, expression: string | null];
  removeNode: [nodeKey: string];
  removeEdge: [edgeKey: string];
}>();

const TYPE_TAG_LABELS: Record<string, string> = {
  start: '发起节点',
  approval: '审批节点',
  condition: '条件分支',
  cc: '抄送节点',
  service: '服务调用',
  parallel: '并行网关',
  end: '结束节点',
};

const typeTag = computed(() =>
  props.selectedNode ? (TYPE_TAG_LABELS[props.selectedNode.type] ?? '节点') : '',
);

/** 名称直接回写 DSL 节点（start 名称用于发起语义展示，同样可改） */
const editableName = computed(() => props.selectedNode !== null);

function submitName(value: string) {
  if (!props.selectedNode) return;
  const trimmed = value.trim();
  if (trimmed) emit('updateNodeName', props.selectedNode.key, trimmed);
}

/** 条件表达式统一出口：null = 默认分支（清空 condition），字符串 = 条件分支 */
function applyCondition(edgeKey: string, expression: string | null) {
  emit(
    'updateEdgeCondition',
    edgeKey,
    expression !== null && expression.trim() === '' ? '' : expression,
  );
}
</script>

<template>
  <aside class="workflow-inspector" aria-label="流程属性">
    <ElScrollbar class="workflow-inspector__scroll">
      <template v-if="selectedNode">
        <div class="workflow-inspector__header">
          <ElTag class="workflow-inspector__type" size="small" effect="light">{{ typeTag }}</ElTag>
          <ElButton
            v-if="!readonly && selectedNode.type !== 'start'"
            type="danger"
            text
            size="small"
            :icon="RiDeleteBin6Fill"
            @click="emit('removeNode', selectedNode.key)"
          >
            删除节点
          </ElButton>
        </div>

        <div class="workflow-inspector__field">
          <label class="workflow-inspector__label" :for="`workflow-node-name-${selectedNode.key}`">
            节点名称
          </label>
          <ElInput
            :id="`workflow-node-name-${selectedNode.key}`"
            :model-value="selectedNode.name"
            :disabled="!editableName || readonly"
            @change="submitName"
          />
        </div>

        <WorkflowApprovalPanel
          v-if="selectedNode.type === 'approval'"
          :node="selectedNode"
          :fields="fields"
          :actor-options="actorOptions"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <WorkflowConditionPanel
          v-else-if="selectedNode.type === 'condition'"
          :node="selectedNode"
          :document="document"
          @update-edge-condition="applyCondition"
        />
        <WorkflowCcPanel
          v-else-if="selectedNode.type === 'cc'"
          :node="selectedNode"
          :fields="fields"
          :actor-options="actorOptions"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <WorkflowServicePanel
          v-else-if="selectedNode.type === 'service'"
          :node="selectedNode"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <p v-else class="workflow-inspector__hint">
          {{
            selectedNode.type === 'start'
              ? '流程入口：实例从该节点发起，配置请在流程设置中调整。'
              : selectedNode.type === 'parallel'
                ? '并行网关由 split/join 成对协作，V1 设计器暂不提供并行编排，请通过 DSL 配置。'
                : '流程终点：到达即实例完成。'
          }}
        </p>
      </template>

      <template v-else-if="selectedEdge">
        <WorkflowEdgePanel
          :edge="selectedEdge"
          :document="document"
          @update-condition="(expression) => applyCondition(selectedEdge!.key, expression)"
          @remove-edge="
            () => {
              if (selectedEdge) emit('removeEdge', selectedEdge.key);
            }
          "
        />
      </template>

      <p v-else class="workflow-inspector__hint">请选择流程节点或连线以设置属性。</p>
    </ElScrollbar>
  </aside>
</template>

<style scoped lang="scss">
.workflow-inspector {
  display: flex;
  min-width: 0;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-lighter);

  &__scroll {
    min-height: 0;
    flex: 1;
  }

  &__header {
    display: flex;
    padding: var(--el-space-md) var(--el-space-md) var(--el-space-xs);
    align-items: center;
    justify-content: space-between;
  }

  &__field {
    display: flex;
    padding: 0 var(--el-space-md) var(--el-space-sm);
    flex-direction: column;
    gap: var(--el-space-xs);
    border-bottom: 1px solid var(--el-border-color-lighter);
    margin-bottom: var(--el-space-sm);
  }

  &__label {
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
  }

  &__hint {
    padding: var(--el-space-lg) var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.8;
  }
}
</style>
