<script setup lang="ts">
import { RiDeleteBin6Fill } from '@remixicon/vue';
import { ElAlert, ElButton, ElInput, ElPopover, ElScrollbar, ElTag } from 'element-plus';
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
import WorkflowNodeSettingsTabs from './panels/WorkflowNodeSettingsTabs.vue';
import WorkflowPluginPanel from './panels/WorkflowPluginPanel.vue';
import WorkflowServicePanel from './panels/WorkflowServicePanel.vue';
import WorkflowSubflowPanel from './panels/WorkflowSubflowPanel.vue';

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
  subflow: '子流程',
  plugin: '插件节点',
  service: '服务调用',
  parallel: '并行网关',
  end: '结束节点',
};

const typeTag = computed(() =>
  props.selectedNode ? (TYPE_TAG_LABELS[props.selectedNode.type] ?? '节点') : '',
);
const selectedNodeID = computed(() => {
  if (!props.selectedNode) return '';
  const index = props.document.nodes.findIndex((node) => node.key === props.selectedNode?.key);
  return index >= 0 ? index : '';
});

/** 名称直接回写 DSL 节点（start 名称用于发起语义展示，同样可改） */
const editableName = computed(() => props.selectedNode !== null);

const connectionWarning = computed(() => {
  const node = props.selectedNode;
  if (!node) return '';
  const hasIncoming = props.document.edges.some((edge) => edge.target === node.key);
  const hasOutgoing = props.document.edges.some((edge) => edge.source === node.key);
  if (node.type === 'start') return hasOutgoing ? '' : '发起节点尚未连接后续节点';
  if (node.type === 'end') return hasIncoming ? '' : '结束节点尚未连接前置节点';
  return hasIncoming && hasOutgoing ? '' : '节点尚未正确连接';
});

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
    <div class="workflow-inspector__primary-tabs" role="tablist" aria-label="属性范围">
      <button type="button" class="is-active" role="tab" aria-selected="true">节点属性</button>
      <button type="button" role="tab" aria-selected="false" disabled>流程属性</button>
    </div>
    <ElScrollbar class="workflow-inspector__scroll">
      <template v-if="selectedNode">
        <ElAlert
          v-if="connectionWarning"
          class="workflow-inspector__connection-warning"
          type="warning"
          :closable="false"
          show-icon
        >
          <template #title>
            {{ connectionWarning }}
            <ElPopover placement="bottom" :width="250" trigger="click">
              <template #reference>
                <button type="button" class="workflow-inspector__help-link">查看连接方式</button>
              </template>
              <strong>连接节点</strong>
              <p class="workflow-inspector__help-copy">
                将鼠标移到节点边缘，从连接锚点拖到目标节点；业务节点需要同时连接前置与后续节点。
              </p>
            </ElPopover>
          </template>
        </ElAlert>
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
          <div class="workflow-inspector__field-heading">
            <label class="workflow-inspector__label" :for="`workflow-node-name-${selectedNode.key}`">
              <i>*</i>节点名称
            </label>
            <span>节点ID：{{ selectedNodeID }}</span>
          </div>
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
        <WorkflowSubflowPanel
          v-else-if="selectedNode.type === 'subflow'"
          :node="selectedNode"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <WorkflowPluginPanel
          v-else-if="selectedNode.type === 'plugin'"
          :node="selectedNode"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <WorkflowNodeSettingsTabs
          v-if="selectedNode.type === 'approval'"
          :key="selectedNode.key"
          :node="selectedNode"
          :fields="fields"
          @update-config="
            (config) => {
              if (selectedNode) emit('updateNodeConfig', selectedNode.key, config);
            }
          "
        />
        <p
          v-if="['start', 'parallel', 'end'].includes(selectedNode.type)"
          class="workflow-inspector__hint"
        >
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

  &__primary-tabs {
    display: grid;
    flex: 0 0 58px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    grid-template-columns: repeat(2, 1fr);

    button {
      position: relative;
      color: var(--el-text-color-primary);
      background: transparent;
      border: 0;
      cursor: pointer;
      font: inherit;
      font-size: 15px;

      &.is-active {
        color: var(--el-color-primary);
        font-weight: 600;

        &::after {
          position: absolute;
          right: 0;
          bottom: -1px;
          left: 0;
          height: 2px;
          background: var(--el-color-primary);
          content: '';
        }
      }

      &:disabled { cursor: not-allowed; opacity: 1; }
    }
  }

  &__connection-warning {
    margin: var(--el-space-sm) var(--el-space-md) 0;
  }

  &__help-link {
    padding: 0;
    margin-left: 6px;
    color: var(--el-color-primary);
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
  }

  &__help-copy {
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
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

    i { color: var(--el-color-danger); font-style: normal; }
  }

  &__field-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;

    > span {
      padding: 4px 12px;
      color: var(--el-text-color-regular);
      background: var(--el-fill-color-lighter);
      border: 1px solid var(--el-border-color);
      border-radius: 4px;
      font-size: 12px;
    }
  }

  &__hint {
    padding: var(--el-space-lg) var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.8;
  }
}
</style>
