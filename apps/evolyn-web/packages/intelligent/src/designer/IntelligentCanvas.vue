<script setup lang="ts">
import LogicFlow, { PolylineEdge, PolylineEdgeModel } from '@logicflow/core';
import { onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import type { IntelligentDocument, IntelligentNodeType, IntelligentPosition } from '../schema';
import { IntelligentNodeModel, IntelligentVueNodeView } from './IntelligentVueNodeView';

defineOptions({ name: 'IntelligentCanvas' });

const props = defineProps<{
  document: IntelligentDocument;
  selectedNodeId: string | null;
}>();

const emit = defineEmits<{
  selectNode: [nodeId: string | null];
  moveNode: [nodeId: string, position: IntelligentPosition];
}>();

const canvasRef = useTemplateRef<HTMLElement>('canvasRef');
const logicFlow = shallowRef<LogicFlow | null>(null);
const zoomPercent = shallowRef(100);
let resizeObserver: ResizeObserver | null = null;
let centerFrame = 0;

const NODE_SIZES: Record<IntelligentNodeType, { width: number; height: number }> = {
  trigger: { width: 286, height: 88 },
  action: { width: 286, height: 88 },
  end: { width: 122, height: 54 },
};

class IntelligentEdgeModel extends PolylineEdgeModel {
  override getEdgeStyle() {
    return { ...super.getEdgeStyle(), stroke: '#63728a', strokeWidth: 1.6 };
  }
}

function graphData(): LogicFlow.GraphConfigData {
  return {
    nodes: props.document.nodes.map((node) => ({
      id: node.id,
      type: `intelligent-${node.type}`,
      x: node.position.x,
      y: node.position.y,
      text: '',
      draggable: true,
      properties: {
        ...NODE_SIZES[node.type],
        assistantType: node.type,
        label: node.name,
        description: node.description,
        selected: node.id === props.selectedNodeId,
      },
    })),
    edges: props.document.edges.map((edge) => ({
      id: edge.id,
      type: 'intelligent-edge',
      sourceNodeId: edge.source,
      targetNodeId: edge.target,
    })),
  };
}

function renderGraph(): void {
  logicFlow.value?.render(graphData());
}

/** 保持 100% 缩放，仅将当前文档的几何中心放到可视区中央。 */
function centerDocument(): void {
  const instance = logicFlow.value;
  const nodes = props.document.nodes;
  if (!instance || nodes.length === 0) return;
  const xs = nodes.map((node) => node.position.x);
  const ys = nodes.map((node) => node.position.y);
  instance.focusOn({
    x: (Math.min(...xs) + Math.max(...xs)) / 2,
    y: (Math.min(...ys) + Math.max(...ys)) / 2,
  });
}

function scheduleCenterDocument(): void {
  cancelAnimationFrame(centerFrame);
  centerFrame = requestAnimationFrame(centerDocument);
}

function syncSelection(): void {
  const instance = logicFlow.value;
  if (!instance) return;
  for (const node of instance.graphModel.nodes) {
    instance.setProperties(node.id, { selected: node.id === props.selectedNodeId });
  }
  instance.clearSelectElements();
  if (props.selectedNodeId && instance.getNodeModelById(props.selectedNodeId)) {
    instance.selectElementById(props.selectedNodeId, false, false);
  }
}

function syncZoom(): void {
  zoomPercent.value = Math.round((logicFlow.value?.getTransform().SCALE_X ?? 1) * 100);
}

function zoomIn(): void {
  logicFlow.value?.zoom(true);
  syncZoom();
}

function zoomOut(): void {
  logicFlow.value?.zoom(false);
  syncZoom();
}

function fitView(): void {
  logicFlow.value?.fitView();
  syncZoom();
}

function resizeCanvas(): void {
  const element = canvasRef.value;
  if (!element || !logicFlow.value) return;
  logicFlow.value.resize(element.clientWidth, element.clientHeight);
}

defineExpose({ fitView, zoomIn, zoomOut, zoomPercent });

watch(() => props.document, renderGraph, { deep: false });
watch(
  () => props.document.nodes.length,
  scheduleCenterDocument,
);
watch(() => props.selectedNodeId, syncSelection);

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;
  const instance = new LogicFlow({
    container: element,
    width: element.clientWidth,
    height: element.clientHeight,
    edgeType: 'intelligent-edge',
    grid: { visible: true, size: 20, type: 'dot', config: { color: '#dfe4eb' } },
    history: false,
    adjustEdge: false,
    hideAnchors: true,
    hoverOutline: false,
    nodeSelectedOutline: false,
    nodeTextEdit: false,
    textEdit: false,
    keyboard: { enabled: false },
    style: {
      arrow: { fill: '#63728a', stroke: '#63728a', offset: 8, verticalLength: 5 },
      polyline: { stroke: '#63728a', strokeWidth: 1.6 },
    },
  });
  logicFlow.value = instance;
  instance.register({ type: 'intelligent-edge', view: PolylineEdge, model: IntelligentEdgeModel });
  for (const type of ['trigger', 'action', 'end'] as const) {
    instance.register({
      type: `intelligent-${type}`,
      view: IntelligentVueNodeView,
      model: IntelligentNodeModel,
    });
  }
  instance.on('node:click', ({ data }) => emit('selectNode', String(data.id)));
  instance.on('blank:click', () => emit('selectNode', null));
  instance.on('node:drop', ({ data }) => {
    emit('moveNode', String(data.id), { x: data.x, y: data.y });
  });
  instance.on('graph:transform', syncZoom);
  renderGraph();
  scheduleCenterDocument();

  resizeObserver = new ResizeObserver(resizeCanvas);
  resizeObserver.observe(element);
});

onBeforeUnmount(() => {
  cancelAnimationFrame(centerFrame);
  resizeObserver?.disconnect();
  resizeObserver = null;
  logicFlow.value?.destroy();
  logicFlow.value = null;
});
</script>

<template>
  <div class="intelligent-canvas">
    <div ref="canvasRef" class="intelligent-canvas__graph" />
  </div>
</template>

<style scoped lang="scss">
.intelligent-canvas {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: #f7f8fa;

  &__graph {
    width: 100%;
    height: 100%;

    :deep(.lf-graph) {
      background: #f7f8fa;
    }

    :deep(.intelligent-vue-node-content) {
      width: 100%;
      height: 100%;
    }

    :deep(.lf-node foreignObject) {
      overflow: visible;
    }

    :deep(.lf-node:focus),
    :deep(.lf-edge:focus) {
      outline: none;
    }

    :deep(.lf-outline-node),
    :deep(.lf-outline-edge) {
      display: none;
    }
  }
}
</style>
