<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, shallowRef, watch } from 'vue';
import { computeAutoLayout, resolveNodePositions } from '../adapters/graph';
import {
  type WorkflowActorOptions,
  type WorkflowDocument,
  type WorkflowField,
  type WorkflowIssue,
  type WorkflowNodeType,
  type WorkflowPosition,
  addEdge,
  addNode,
  collectWorkflowIssues,
  insertNodeOnEdge,
  removeEdge,
  removeNode,
  resolveIssueTargets,
  setNodePosition,
  updateEdge,
  updateNode,
} from '../schema';
import WorkflowCanvas from './WorkflowCanvas.vue';
import WorkflowInspector from './WorkflowInspector.vue';
import WorkflowPalette from './WorkflowPalette.vue';

/**
 * 流程设计器装配：素材面板 + LogicFlow 画布 + 属性面板。
 * DSL 文档（props.document）是唯一事实源，一切编辑经 schema/lifecycle
 * 的不可变操作落到新文档并整体上抛；画布坐标只写 settings.designer.layout。
 * issues 同时合并外部（后端发布校验回传）与本层即时校验，画布一次高亮。
 */
defineOptions({ name: 'WorkflowDesigner' });

const props = defineProps<{
  document: WorkflowDocument;
  fields: readonly WorkflowField[];
  actorOptions?: WorkflowActorOptions;
  /** 外部校验问题（后端 WORKFLOW_DEFINITION_INVALID 的 issues 负载） */
  issues?: readonly WorkflowIssue[];
  /** 只读预览（版本快照）：隐藏素材与属性面板，禁止一切编辑 */
  readonly?: boolean;
}>();

const emit = defineEmits<{
  updateDocument: [document: WorkflowDocument];
}>();

const selectedNodeKey = shallowRef<string | null>(props.document.nodes[0]?.key ?? null);
const selectedEdgeKey = shallowRef<string | null>(null);
const undoStack = shallowRef<WorkflowDocument[]>([]);
const redoStack = shallowRef<WorkflowDocument[]>([]);
let expectedDocument: WorkflowDocument | null = null;

const selectedNode = computed(
  () => props.document.nodes.find((node) => node.key === selectedNodeKey.value) ?? null,
);
const selectedEdge = computed(
  () => props.document.edges.find((edge) => edge.key === selectedEdgeKey.value) ?? null,
);

/** 即时校验 + 外部 issues 合并后的画布高亮定位 */
const allIssues = computed(() => [
  ...(props.issues ?? []),
  ...collectWorkflowIssues(props.document),
]);
const issueTarget = computed(() => resolveIssueTargets(props.document, allIssues.value));

watch(
  () => props.document,
  (document) => {
    if (expectedDocument === document) {
      expectedDocument = null;
    } else {
      // 版本切换/服务端重载属于新的编辑会话，不允许撤销回另一个版本。
      undoStack.value = [];
      redoStack.value = [];
    }
    if (
      selectedNodeKey.value &&
      !document.nodes.some((node) => node.key === selectedNodeKey.value)
    ) {
      selectedNodeKey.value = null;
    }
    if (
      selectedEdgeKey.value &&
      !document.edges.some((edge) => edge.key === selectedEdgeKey.value)
    ) {
      selectedEdgeKey.value = null;
    }
  },
);

function commitDocument(document: WorkflowDocument) {
  undoStack.value = [...undoStack.value.slice(-49), props.document];
  redoStack.value = [];
  expectedDocument = document;
  emit('updateDocument', document);
}

function undo() {
  if (props.readonly || undoStack.value.length === 0) return;
  const previous = undoStack.value.at(-1);
  if (!previous) return;
  undoStack.value = undoStack.value.slice(0, -1);
  redoStack.value = [...redoStack.value, props.document];
  expectedDocument = previous;
  emit('updateDocument', previous);
}

function redo() {
  if (props.readonly || redoStack.value.length === 0) return;
  const next = redoStack.value.at(-1);
  if (!next) return;
  redoStack.value = redoStack.value.slice(0, -1);
  undoStack.value = [...undoStack.value, props.document];
  expectedDocument = next;
  emit('updateDocument', next);
}

function selectNode(nodeKey: string) {
  selectedNodeKey.value = nodeKey;
  selectedEdgeKey.value = null;
}

function selectEdge(edgeKey: string) {
  selectedEdgeKey.value = edgeKey;
  selectedNodeKey.value = null;
}

/** 新增节点落点：选中的节点下方；无选中时取自动布局的兜底坐标区域 */
function positionForNewNode(): WorkflowPosition {
  const positions = resolveNodePositions(props.document);
  const anchorKey = selectedNodeKey.value;
  const anchor = anchorKey ? positions[anchorKey] : undefined;
  if (anchor) {
    return { x: anchor.x + 240, y: anchor.y + 150 };
  }
  const maxBottom = Object.values(positions).reduce(
    (max, position) => Math.max(max, position.y),
    0,
  );
  return { x: 420, y: maxBottom + 150 };
}

function handleAddNode(type: WorkflowNodeType) {
  if (props.readonly) return;
  const selectedKey = selectedNodeKey.value;
  const outgoing = selectedKey
    ? props.document.edges.filter((edge) => edge.source === selectedKey)
    : [];
  const selectedNode = props.document.nodes.find((node) => node.key === selectedKey);
  // 一条后继边代表线性路径，新业务节点直接插入该边，避免出现「发起 →
  // 结束」与新节点彼此断开的默认画布。多分支/结束节点保持手工连线，避免猜测语义。
  const result =
    type !== 'end' && selectedNode?.type !== 'end' && outgoing.length === 1
      ? insertNodeOnEdge(props.document, type, positionForNewNode(), outgoing[0].key)
      : addNode(props.document, type, positionForNewNode());
  const { document, node } = result;
  commitDocument(document);
  selectNode(node.key);
}

/** 从工具栏拖到画布时按落点创建独立节点，由用户通过锚点决定连接关系。 */
function handleDropNode(type: WorkflowNodeType, position: WorkflowPosition) {
  if (props.readonly) return;
  const { document, node } = addNode(props.document, type, position);
  commitDocument(document);
  selectNode(node.key);
}

function handleAddFromNode(
  nodeKey: string,
  type: WorkflowNodeType,
  direction: 'top' | 'right' | 'bottom' | 'left',
) {
  if (props.readonly) return;
  const positions = resolveNodePositions(props.document);
  const anchor = positions[nodeKey] ?? { x: 420, y: 240 };
  const offsets = {
    top: { x: 0, y: -150 },
    right: { x: 250, y: 0 },
    bottom: { x: 0, y: 150 },
    left: { x: -250, y: 0 },
  } as const;
  const offset = offsets[direction];
  const added = addNode(props.document, type, {
    x: anchor.x + offset.x,
    y: anchor.y + offset.y,
  });
  // 上/左方向表示新节点接入当前节点之前；下/右方向表示接在之后。
  const document = ['top', 'left'].includes(direction)
    ? addEdge(added.document, added.node.key, nodeKey)
    : addEdge(added.document, nodeKey, added.node.key);
  commitDocument(document);
  selectNode(added.node.key);
}

function handleMoveNode(nodeKey: string, position: WorkflowPosition) {
  if (props.readonly) return;
  commitDocument(setNodePosition(props.document, nodeKey, position));
}

/** 锚点连线：重复边静默忽略（自环已在画布层拦截） */
function handleConnectEdge(source: string, target: string) {
  if (props.readonly) return;
  if (props.document.edges.some((edge) => edge.source === source && edge.target === target)) return;
  commitDocument(addEdge(props.document, source, target));
}

function handleUpdateName(nodeKey: string, name: string) {
  if (props.readonly) return;
  commitDocument(updateNode(props.document, nodeKey, { name }));
}

function handleUpdateConfig(nodeKey: string, config: WorkflowDocument['nodes'][number]['config']) {
  if (props.readonly) return;
  commitDocument(updateNode(props.document, nodeKey, { config }));
}

/**
 * 条件出边统一写入口：null = 恢复默认分支（清空 condition）；
 * 空串保留为「未填表达式」状态交给校验提示，避免静默吞掉用户输入。
 */
function handleUpdateEdgeCondition(edgeKey: string, expression: string | null) {
  if (props.readonly) return;
  commitDocument(
    updateEdge(props.document, edgeKey, {
      condition: expression === null ? undefined : { expression },
    }),
  );
}

function handleRemoveNode(nodeKey: string) {
  if (props.readonly) return;
  commitDocument(removeNode(props.document, nodeKey));
}

function handleRemoveEdge(edgeKey: string) {
  if (props.readonly) return;
  commitDocument(removeEdge(props.document, edgeKey));
}

function deleteSelected() {
  if (selectedEdgeKey.value) {
    handleRemoveEdge(selectedEdgeKey.value);
    selectedEdgeKey.value = null;
    return;
  }
  if (selectedNode.value && !['start', 'end'].includes(selectedNode.value.type)) {
    handleRemoveNode(selectedNode.value.key);
    selectedNodeKey.value = null;
  }
}

/** 对齐/排列统一回到协议图的分层布局，避免 LogicFlow 自持坐标形成第二事实源。 */
function arrangeNodes() {
  if (props.readonly) return;
  const layout = computeAutoLayout(props.document);
  const arranged = props.document.nodes.reduce((document, node) => {
    const position = layout[node.key];
    return position ? setNodePosition(document, node.key, position) : document;
  }, props.document);
  commitDocument(arranged);
}

function handleKeyboard(event: KeyboardEvent) {
  if (props.readonly) return;
  const target = event.target as HTMLElement | null;
  if (target?.closest('input, textarea, [contenteditable="true"]')) return;
  const command = event.metaKey || event.ctrlKey;
  if (command && event.key.toLowerCase() === 'z') {
    event.preventDefault();
    event.shiftKey ? redo() : undo();
    return;
  }
  if (command && event.key.toLowerCase() === 'y') {
    event.preventDefault();
    redo();
    return;
  }
  if (event.key === 'Delete' || event.key === 'Backspace') {
    event.preventDefault();
    deleteSelected();
  }
}

onMounted(() => window.addEventListener('keydown', handleKeyboard));
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeyboard));
</script>

<template>
  <section
    class="workflow-designer"
    :class="{ 'workflow-designer--readonly': readonly }"
    aria-label="流程设计器"
  >
    <WorkflowPalette
      v-if="!readonly"
      class="workflow-designer__toolbar"
      :can-undo="undoStack.length > 0"
      :can-redo="redoStack.length > 0"
      :can-delete="Boolean(selectedEdge || (selectedNode && !['start', 'end'].includes(selectedNode.type)))"
      @add-node="handleAddNode"
      @undo="undo"
      @redo="redo"
      @align="arrangeNodes"
      @distribute="arrangeNodes"
      @delete-selected="deleteSelected"
    />
    <WorkflowCanvas
      class="workflow-designer__canvas"
      :document="document"
      :selected-node-key="selectedNodeKey"
      :selected-edge-key="selectedEdgeKey"
      :error-node-keys="issueTarget.nodeKeys"
      :error-edge-keys="issueTarget.edgeKeys"
      :readonly="readonly"
      @select-node="selectNode"
      @select-edge="selectEdge"
      @update-node-position="handleMoveNode"
      @connect-edge="handleConnectEdge"
      @drop-node="handleDropNode"
      @add-from-node="handleAddFromNode"
    />
    <WorkflowInspector
      v-if="!readonly"
      class="workflow-designer__inspector"
      :document="document"
      :selected-node="selectedNode"
      :selected-edge="selectedEdge"
      :fields="fields"
      :actor-options="actorOptions"
      @update-node-name="handleUpdateName"
      @update-node-config="handleUpdateConfig"
      @update-edge-condition="handleUpdateEdgeCondition"
      @remove-node="handleRemoveNode"
      @remove-edge="handleRemoveEdge"
    />
  </section>
</template>

<style scoped lang="scss">
.workflow-designer {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template: auto minmax(0, 1fr) / minmax(0, 1fr) 360px;

  &--readonly {
    grid-template: minmax(0, 1fr) / minmax(0, 1fr);
  }

  &__toolbar {
    grid-column: 1 / -1;
  }

  &__canvas,
  &__inspector {
    min-height: 0;
  }
}

@media (max-width: 900px) {
  .workflow-designer {
    grid-template-columns: minmax(0, 1fr) 300px;
  }
}

@media (max-width: 700px) {
  .workflow-designer {
    grid-template-columns: minmax(0, 1fr);

    &__inspector {
      display: none;
    }
  }
}
</style>
