<script setup lang="ts">
import LogicFlow from '@logicflow/core';
import { register } from '@logicflow/vue-node-registry';
import {
  type ShallowRef,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
  watch,
} from 'vue';
import type { WorkflowDocument, WorkflowNode, WorkflowNodeType } from '../schema';
import WorkflowCanvasControls from './WorkflowCanvasControls.vue';
import WorkflowNodeCard from './WorkflowNodeCard.vue';
import { WorkflowVueNodeView } from './WorkflowVueNodeView';

defineOptions({ name: 'WorkflowCanvas' });

const props = defineProps<{
  document: WorkflowDocument;
  selectedNodeId: string | null;
}>();

const emit = defineEmits<{
  selectNode: [nodeId: string];
  updateNodePosition: [nodeId: string, position: WorkflowNode['position']];
}>();

const canvasRef = useTemplateRef<HTMLElement>('canvasRef');
const logicFlow: ShallowRef<LogicFlow | null> = shallowRef(null);
const zoomPercent = shallowRef(100);
const activeNodeId = shallowRef<string | null>(props.selectedNodeId);
let resizeObserver: ResizeObserver | null = null;

function renderGraph() {
  logicFlow.value?.render(toGraphData(props.document));
}

/**
 * 节点选择由 Vue 卡片自行呈现，LogicFlow 只高亮关联边。
 * 这样既保留清晰的选中反馈，也不会叠加引擎默认的深色节点轮廓。
 */
function syncSelection() {
  const instance = logicFlow.value;
  if (!instance) return;

  instance.graphModel.clearSelectElements();
  props.document.nodes.forEach((node) => {
    instance.setProperties(node.id, { selected: node.id === activeNodeId.value });
  });
  if (!activeNodeId.value) return;

  instance.getNodeEdges(activeNodeId.value).forEach((edge) => {
    instance.selectElementById(edge.id, true);
  });
}

function resizeCanvas() {
  const element = canvasRef.value;
  const instance = logicFlow.value;
  if (!element || !instance) return;
  instance.resize(element.clientWidth, element.clientHeight);
}

function selectNode(nodeId: string) {
  activeNodeId.value = nodeId;
  syncSelection();
  emit('selectNode', nodeId);
}

/** LogicFlow 将缩放状态保留在画布内，控制条只派发命令，避免引入第二份图状态。 */
function syncZoom() {
  const scale = logicFlow.value?.getTransform().SCALE_X ?? 1;
  zoomPercent.value = Math.round(scale * 100);
}

function undo() {
  logicFlow.value?.undo();
}

function redo() {
  logicFlow.value?.redo();
}

function zoomIn() {
  logicFlow.value?.zoom(true);
  syncZoom();
}

function zoomOut() {
  logicFlow.value?.zoom(false);
  syncZoom();
}

function fitView() {
  logicFlow.value?.fitView();
  syncZoom();
}

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;

  const instance = new LogicFlow({
    container: element,
    width: element.clientWidth,
    height: element.clientHeight,
    edgeType: 'polyline',
    grid: true,
    history: true,
    edgeSelectedOutline: false,
    hideAnchors: true,
    hoverOutline: false,
    nodeSelectedOutline: false,
    nodeTextEdit: false,
    textEdit: false,
    snapline: true,
    style: {
      rect: {
        radius: 10,
        fill: 'var(--el-bg-color)',
        stroke: 'var(--el-color-primary-light-5)',
        strokeWidth: 1.5,
      },
      circle: {
        fill: 'var(--el-bg-color)',
        stroke: 'var(--el-color-primary-light-5)',
        strokeWidth: 1.5,
      },
      polyline: { stroke: 'var(--el-color-primary-light-5)', strokeWidth: 2 },
      arrow: {
        fill: 'var(--el-color-primary-light-5)',
        stroke: 'var(--el-color-primary-light-5)',
        offset: 8,
        verticalLength: 5,
      },
      nodeText: { color: 'var(--el-text-color-primary)', fontSize: 15 },
    },
  });
  logicFlow.value = instance;
  register(
    { type: 'workflow-start', component: WorkflowNodeCard, view: WorkflowVueNodeView },
    instance,
  );
  register(
    { type: 'workflow-approval', component: WorkflowNodeCard, view: WorkflowVueNodeView },
    instance,
  );
  register(
    { type: 'workflow-end', component: WorkflowNodeCard, view: WorkflowVueNodeView },
    instance,
  );

  instance.on('node:click', ({ data }) => selectNode(String(data.id)));
  instance.on('node:drop', ({ data }) => {
    emit('updateNodePosition', String(data.id), { x: data.x, y: data.y });
  });
  instance.on('graph:transform', syncZoom);
  renderGraph();
  syncSelection();

  resizeObserver = new ResizeObserver(resizeCanvas);
  resizeObserver.observe(element);
});

watch(
  () => props.document,
  () => {
    renderGraph();
    syncSelection();
  },
);
watch(
  () => props.selectedNodeId,
  (nodeId) => {
    if (nodeId === activeNodeId.value) return;
    activeNodeId.value = nodeId;
    syncSelection();
  },
);

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  logicFlow.value?.destroy();
  logicFlow.value = null;
});

function toGraphData(document: WorkflowDocument): LogicFlow.GraphConfigData {
  return {
    nodes: document.nodes.map((node) => ({
      id: node.id,
      type: resolveLogicFlowNodeType(node.type),
      x: node.position.x,
      y: node.position.y,
      text: '',
      properties: {
        width: node.type === 'approval' ? 210 : 174,
        height: node.type === 'approval' ? 58 : 48,
        workflowType: node.type,
        label: node.name,
        selected: node.id === activeNodeId.value,
      },
    })),
    edges: document.edges.map((edge) => ({
      id: edge.id,
      type: 'polyline',
      sourceNodeId: edge.sourceNodeId,
      targetNodeId: edge.targetNodeId,
    })),
  };
}

function resolveLogicFlowNodeType(type: WorkflowNodeType) {
  return `workflow-${type}`;
}
</script>

<template>
  <div class="workflow-canvas" aria-label="流程画布">
    <div ref="canvasRef" class="workflow-canvas__graph" />
    <WorkflowCanvasControls
      :zoom-percent="zoomPercent"
      @fit-view="fitView"
      @redo="redo"
      @undo="undo"
      @zoom-in="zoomIn"
      @zoom-out="zoomOut"
    />
  </div>
</template>

<style scoped lang="scss">
.workflow-canvas {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: visible;
  background: var(--el-fill-color-lighter);

  &__graph {
    width: 100%;
    height: 100%;
    overflow: hidden;

    :deep(.lf-graph) {
      background: var(--el-fill-color-lighter);
    }

    :deep(.custom-vue-node-content) {
      width: 100%;
      height: 100%;
    }

    // LogicFlow 会在节点点击后聚焦 SVG 容器；浏览器焦点框和引擎 outline
    // 都是矩形，会与起止节点的胶囊轮廓形成截图中的双层边框。
    :deep(.lf-node:focus),
    :deep(.lf-edge:focus) {
      outline: none;
    }

    :deep(.lf-outline-node),
    :deep(.lf-outline-edge),
    :deep(.lf-outline > .lf-outline) {
      display: none;
    }

    :deep(.lf-edge-selected polyline) {
      stroke: var(--el-color-primary) !important;
      stroke-width: 2px;
    }

    :deep(.lf-edge-selected .lf-arrow path) {
      fill: var(--el-color-primary) !important;
      stroke: var(--el-color-primary) !important;
    }
  }
}
</style>
