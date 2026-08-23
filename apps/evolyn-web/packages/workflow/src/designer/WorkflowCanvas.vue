<script setup lang="ts">
import LogicFlow, { type GraphConfigData } from '@logicflow/core';
import {
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
  watch,
  type ShallowRef,
} from 'vue';
import type { WorkflowDocument, WorkflowNode, WorkflowNodeType } from '../schema';

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
let resizeObserver: ResizeObserver | null = null;

function renderGraph() {
  logicFlow.value?.render(toGraphData(props.document));
}

function resizeCanvas() {
  const element = canvasRef.value;
  const instance = logicFlow.value;
  if (!element || !instance) return;
  instance.resize(element.clientWidth, element.clientHeight);
}

function selectNode(nodeId: string) {
  emit('selectNode', nodeId);
}

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;

  const instance = new LogicFlow({
    container: element,
    width: element.clientWidth,
    height: element.clientHeight,
    edgeType: 'polyline',
    grid: {
      visible: true,
      size: 20,
      type: 'dot',
      config: { color: 'var(--el-border-color-lighter)', thickness: 1 },
    },
    history: true,
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

  instance.on('node:click', ({ data }) => selectNode(String(data.id)));
  instance.on('node:drop', ({ data }) => {
    emit('updateNodePosition', String(data.id), { x: data.x, y: data.y });
  });
  renderGraph();

  resizeObserver = new ResizeObserver(resizeCanvas);
  resizeObserver.observe(element);
});

watch(() => props.document, renderGraph);
watch(
  () => props.selectedNodeId,
  (nodeId) => {
    if (nodeId) selectNode(nodeId);
  },
);

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  logicFlow.value?.destroy();
  logicFlow.value = null;
});

function toGraphData(document: WorkflowDocument): GraphConfigData {
  return {
    nodes: document.nodes.map((node) => ({
      id: node.id,
      type: resolveLogicFlowNodeType(node.type),
      x: node.position.x,
      y: node.position.y,
      text: node.name,
      properties: {
        width: node.type === 'approval' ? 210 : 174,
        height: node.type === 'approval' ? 58 : 48,
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
  return type === 'approval' ? 'rect' : 'ellipse';
}
</script>

<template>
  <div ref="canvasRef" class="workflow-canvas" aria-label="流程画布" />
</template>

<style scoped lang="scss">
.workflow-canvas {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--el-fill-color-lighter);

  :deep(.lf-graph) {
    background: var(--el-fill-color-lighter);
  }
}
</style>
