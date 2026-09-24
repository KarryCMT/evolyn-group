<script setup lang="ts">
import LogicFlow, { PolylineEdge, PolylineEdgeModel } from '@logicflow/core';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import type {
  IntelligentActionType,
  IntelligentDesignerResources,
  IntelligentDocument,
  IntelligentNode,
  IntelligentPosition,
  IntelligentTrigger,
} from '../schema';
import { IntelligentContext } from '../context/IntelligentContext';
import { IntelligentNodeModel, IntelligentNodeView } from '../node/intelligentNodeView';
import { NODE_SIZES } from '../util';
import IntelligentInsertMenu from '../tool/insertMenu/index.vue';
import IntelligentPropertyPanel from '../tool/propertyPanel/index.vue';
import gridBackgroundBase64 from '../assets/img/grid.svg';

defineOptions({ name: 'LogicPanel' });

const props = defineProps<{
  document: IntelligentDocument;
  selectedNodeId: string | null;
  resources?: IntelligentDesignerResources;
}>();

const emit = defineEmits<{
  addNode: [actionType: IntelligentActionType];
  moveNode: [nodeId: string, position: IntelligentPosition];
  removeNode: [nodeId: string];
  selectNode: [nodeId: string | null];
  updateNode: [nodeId: string, patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>];
  updateTrigger: [patch: Partial<IntelligentTrigger>];
}>();

const panelRef = useTemplateRef<HTMLElement>('panelRef');
const canvasRef = useTemplateRef<HTMLElement>('canvasRef');
const logicFlow = shallowRef<LogicFlow | null>(null);
const zoomPercent = shallowRef(100);
const insertMenuVisible = shallowRef(false);
const insertMenuPosition = shallowRef<IntelligentPosition>({ x: 0, y: 0 });
const context = new IntelligentContext(props.document);
const gridBackgroundUrl = `data:image/svg+xml;base64,${gridBackgroundBase64}`;
let resizeObserver: ResizeObserver | null = null;
let centerFrame = 0;

const selectedNode = computed(
  () => props.document.nodes.find((node) => node.id === props.selectedNodeId) ?? null,
);

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
      draggable: node.type !== 'end',
      properties: {
        ...NODE_SIZES[node.type],
        actionType: node.actionType,
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

function openInsertMenu(position?: IntelligentPosition): void {
  const panel = panelRef.value;
  insertMenuPosition.value = position ?? {
    x: (panel?.clientWidth ?? 600) / 2,
    y: 40,
  };
  insertMenuVisible.value = true;
}

function chooseNode(actionType: IntelligentActionType): void {
  emit('addNode', actionType);
  context.events.emit('node:add', actionType);
}

function selectNode(nodeId: string | null): void {
  emit('selectNode', nodeId);
  context.events.emit('node:select', nodeId);
}

function resizeCanvas(): void {
  const element = canvasRef.value;
  if (!element || !logicFlow.value) return;
  logicFlow.value.resize(element.clientWidth, element.clientHeight);
}

defineExpose({ fitView, openInsertMenu, zoomIn, zoomOut, zoomPercent });

watch(() => props.document, renderGraph, { deep: false });
watch(() => props.document.nodes.length, scheduleCenterDocument);
watch(() => props.selectedNodeId, syncSelection);

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;
  const instance = new LogicFlow({
    container: element,
    width: element.clientWidth,
    height: element.clientHeight,
    edgeType: 'intelligent-edge',
    // 复用原项目的网格资源，避免 LogicFlow 内置网格在缩放时产生视觉跳动。
    grid: false,
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
      view: IntelligentNodeView,
      model: IntelligentNodeModel,
    });
  }
  instance.on('node:click', ({ data }) => {
    insertMenuVisible.value = false;
    selectNode(String(data.id));
  });
  instance.on('node:contextmenu', ({ e }) => {
    const bounds = panelRef.value?.getBoundingClientRect();
    openInsertMenu({ x: e.clientX - (bounds?.left ?? 0), y: e.clientY - (bounds?.top ?? 0) });
  });
  instance.on('blank:click', () => {
    insertMenuVisible.value = false;
    selectNode(null);
  });
  instance.on('blank:contextmenu', ({ e }) => {
    const bounds = panelRef.value?.getBoundingClientRect();
    openInsertMenu({ x: e.clientX - (bounds?.left ?? 0), y: e.clientY - (bounds?.top ?? 0) });
  });
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
  context.destroy();
});
</script>

<template>
  <section
    ref="panelRef"
    class="logic-panel"
    :style="{ backgroundImage: `url(${gridBackgroundUrl})` }"
  >
    <div ref="canvasRef" class="logic-panel__canvas" />
    <IntelligentInsertMenu
      v-model="insertMenuVisible"
      :position="insertMenuPosition"
      @select="chooseNode"
    />
    <IntelligentPropertyPanel
      :node="selectedNode"
      :trigger="document.trigger"
      :resources="resources"
      @close="selectNode(null)"
      @remove="emit('removeNode', $event)"
      @update="(nodeId, patch) => emit('updateNode', nodeId, patch)"
      @update-trigger="emit('updateTrigger', $event)"
    />
  </section>
</template>

<style scoped lang="scss">
.logic-panel {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background-color: #f7f8fa;
  background-repeat: repeat;

  &__canvas { width: 100%; height: 100%; }

  :deep(.lf-graph) { background: transparent; }
  :deep(.intelligent-node-content) { width: 100%; height: 100%; }
  :deep(.lf-node foreignObject) { overflow: visible; }
  :deep(.lf-node:focus),
  :deep(.lf-edge:focus) { outline: none; }
  :deep(.lf-outline-node),
  :deep(.lf-outline-edge) { display: none; }
}
</style>
