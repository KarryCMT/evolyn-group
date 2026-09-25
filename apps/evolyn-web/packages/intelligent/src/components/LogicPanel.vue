<script setup lang="ts">
import LogicFlow from '@logicflow/core';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import {
  type IntelligentActionType,
  type IntelligentDesignerResources,
  type IntelligentDocument,
  type IntelligentNode,
  type IntelligentPosition,
  type IntelligentTrigger,
  getIntelligentNodeConfigurationState,
} from '../schema';
import { IntelligentContext } from '../context/IntelligentContext';
import { IntelligentEdgeModel, IntelligentEdgeView } from '../node/intelligentEdgeView';
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
  addNode: [actionType: IntelligentActionType, sourceEdgeId?: string];
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
const isRightDragging = shallowRef(false);
const insertMenuPosition = shallowRef<IntelligentPosition>({ x: 0, y: 0 });
const pendingInsertEdgeId = shallowRef<string | null>(null);
const context = new IntelligentContext(props.document);
const gridBackgroundUrl = `data:image/svg+xml;base64,${gridBackgroundBase64}`;
let resizeObserver: ResizeObserver | null = null;
let centerFrame = 0;
let rightDragState: { pointerId: number; x: number; y: number } | null = null;

const selectedNode = computed(
  () => props.document.nodes.find((node) => node.id === props.selectedNodeId) ?? null,
);
const panelClasses = computed(() => ({
  'logic-panel': true,
  'logic-panel--right-dragging': isRightDragging.value,
}));

function graphData(): LogicFlow.GraphConfigData {
  return {
    nodes: props.document.nodes.map((node) => {
      const configuration = getIntelligentNodeConfigurationState(node, props.document.trigger);
      return {
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
          description:
            node.type === 'action' && !configuration.configured && configuration.empty
              ? '未设置'
              : node.description,
          configured: configuration.configured,
          selected: node.id === props.selectedNodeId,
        },
      };
    }),
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

function openInsertMenu(position?: IntelligentPosition, sourceEdgeId?: string): void {
  const panel = panelRef.value;
  insertMenuPosition.value = position ?? {
    x: (panel?.clientWidth ?? 600) / 2,
    y: 40,
  };
  pendingInsertEdgeId.value = sourceEdgeId ?? null;
  insertMenuVisible.value = true;
}

function chooseNode(actionType: IntelligentActionType): void {
  const sourceEdgeId = props.document.edges.some(
    (edge) => edge.id === pendingInsertEdgeId.value,
  )
    ? pendingInsertEdgeId.value ?? undefined
    : undefined;
  emit('addNode', actionType, sourceEdgeId);
  context.events.emit('node:add', actionType);
  pendingInsertEdgeId.value = null;
}

function closeInsertMenu(): void {
  insertMenuVisible.value = false;
  pendingInsertEdgeId.value = null;
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

function finishRightDrag(): void {
  const state = rightDragState;
  rightDragState = null;
  isRightDragging.value = false;
  if (state && canvasRef.value?.hasPointerCapture(state.pointerId)) {
    canvasRef.value.releasePointerCapture(state.pointerId);
  }
}

function handleRightPointerDown(event: PointerEvent): void {
  if (event.button !== 2) return;
  const element = canvasRef.value;
  if (!element) return;

  // 右键仅用于平移画布，不参与 LogicFlow 的节点选择与上下文菜单事件。
  event.preventDefault();
  event.stopPropagation();
  closeInsertMenu();
  rightDragState = {
    pointerId: event.pointerId,
    x: event.clientX,
    y: event.clientY,
  };
  isRightDragging.value = true;
  element.setPointerCapture(event.pointerId);
}

function handleRightPointerMove(event: PointerEvent): void {
  const state = rightDragState;
  if (!state || state.pointerId !== event.pointerId) return;
  if ((event.buttons & 2) === 0) {
    finishRightDrag();
    return;
  }

  event.preventDefault();
  event.stopPropagation();
  const deltaX = event.clientX - state.x;
  const deltaY = event.clientY - state.y;
  state.x = event.clientX;
  state.y = event.clientY;
  if (deltaX !== 0 || deltaY !== 0) {
    logicFlow.value?.translate(deltaX, deltaY);
  }
}

function handleRightPointerEnd(event: PointerEvent): void {
  if (rightDragState?.pointerId !== event.pointerId) return;
  event.preventDefault();
  event.stopPropagation();
  finishRightDrag();
}

function handleCanvasContextMenu(event: MouseEvent): void {
  // 屏蔽浏览器和 LogicFlow 的右键菜单，避免再次打开节点选择浮窗。
  event.preventDefault();
  event.stopPropagation();
}

defineExpose({ fitView, openInsertMenu, zoomIn, zoomOut, zoomPercent });

watch(() => props.document, renderGraph, { deep: false });
watch(() => props.document.nodes.length, scheduleCenterDocument);
watch(() => props.selectedNodeId, syncSelection);

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;
  element.addEventListener('pointerdown', handleRightPointerDown, true);
  element.addEventListener('pointermove', handleRightPointerMove, true);
  element.addEventListener('pointerup', handleRightPointerEnd, true);
  element.addEventListener('pointercancel', handleRightPointerEnd, true);
  element.addEventListener('contextmenu', handleCanvasContextMenu, true);
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
      arrow: { fill: '#cfd6e0', stroke: '#cfd6e0', offset: 8, verticalLength: 5 },
      polyline: { stroke: '#cfd6e0', strokeWidth: 1.6 },
    },
  });
  logicFlow.value = instance;
  instance.register({
    type: 'intelligent-edge',
    view: IntelligentEdgeView,
    model: IntelligentEdgeModel,
  });
  for (const type of ['trigger', 'action', 'end'] as const) {
    instance.register({
      type: `intelligent-${type}`,
      view: IntelligentNodeView,
      model: IntelligentNodeModel,
    });
  }
  instance.on('node:click', ({ data }) => {
    closeInsertMenu();
    selectNode(String(data.id));
  });
  instance.on('edge:click', ({ data, position }) => {
    selectNode(null);
    openInsertMenu(position.domOverlayPosition, String(data.id));
  });
  instance.on('blank:click', () => {
    closeInsertMenu();
    selectNode(null);
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
  const element = canvasRef.value;
  element?.removeEventListener('pointerdown', handleRightPointerDown, true);
  element?.removeEventListener('pointermove', handleRightPointerMove, true);
  element?.removeEventListener('pointerup', handleRightPointerEnd, true);
  element?.removeEventListener('pointercancel', handleRightPointerEnd, true);
  element?.removeEventListener('contextmenu', handleCanvasContextMenu, true);
  finishRightDrag();
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
    :class="panelClasses"
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
      :nodes="document.nodes"
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

  &--right-dragging,
  &--right-dragging :deep(*) {
    cursor: grabbing !important;
  }

  &__canvas {
    width: 100%;
    height: 100%;
  }

  :deep(.lf-graph) {
    background: transparent;
  }
  :deep(.intelligent-node-content) {
    width: 100%;
    height: 100%;
  }
  :deep(.lf-node foreignObject) {
    overflow: visible;
  }
  :deep(.lf-edge foreignObject) {
    overflow: visible;
  }
  :deep(.intelligent-edge-insert__mount) {
    display: flex;
    width: 32px;
    height: 32px;
    align-items: center;
    justify-content: center;
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
</style>
