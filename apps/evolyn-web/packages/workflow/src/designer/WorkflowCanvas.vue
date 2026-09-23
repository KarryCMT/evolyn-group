<script setup lang="ts">
import LogicFlow, { PolylineEdge, PolylineEdgeModel } from '@logicflow/core';
import { RiGitBranchFill, RiPuzzle2Fill, RiSendPlaneFill, RiUser3Fill } from '@remixicon/vue';
import { register } from '@logicflow/vue-node-registry';
import {
  type ShallowRef,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
  watch,
} from 'vue';
import { toGraphData } from '../adapters/graph';
import type { WorkflowDocument, WorkflowNodeType, WorkflowPosition } from '../schema';
import WorkflowCanvasControls from './WorkflowCanvasControls.vue';
import WorkflowNodeCard from './WorkflowNodeCard.vue';
import WorkflowViewModeToggle from './WorkflowViewModeToggle.vue';
import { WorkflowVueNodeView } from './WorkflowVueNodeView';

defineOptions({ name: 'WorkflowCanvas' });

const props = defineProps<{
  document: WorkflowDocument;
  selectedNodeKey: string | null;
  selectedEdgeKey: string | null;
  errorNodeKeys?: ReadonlySet<string>;
  errorEdgeKeys?: ReadonlySet<string>;
  /** 只读模式：禁止拖拽节点/连线锚点，仅保留浏览（版本预览） */
  readonly?: boolean;
}>();

const emit = defineEmits<{
  selectNode: [nodeKey: string];
  selectEdge: [edgeKey: string];
  updateNodePosition: [nodeKey: string, position: WorkflowPosition];
  connectEdge: [source: string, target: string];
  dropNode: [type: WorkflowNodeType, position: WorkflowPosition];
  addFromNode: [nodeKey: string, type: WorkflowNodeType, direction: AddDirection];
}>();

type AddDirection = 'top' | 'right' | 'bottom' | 'left';

const rootRef = useTemplateRef<HTMLElement>('rootRef');
const canvasRef = useTemplateRef<HTMLElement>('canvasRef');
const logicFlow: ShallowRef<LogicFlow | null> = shallowRef(null);
const zoomPercent = shallowRef(100);
const viewMode = shallowRef<'compact' | 'detailed'>('compact');
const nodePicker = shallowRef<{
  nodeKey: string;
  direction: AddDirection;
  left: number;
  top: number;
} | null>(null);
let resizeObserver: ResizeObserver | null = null;
// 节点拖拽后 LogicFlow 已持有最新坐标。等待父级把该坐标写回 DSL 的期间，
// 不可用旧 props 全量 render，否则会把节点拉回拖拽前的位置。
let pendingNodePosition: { nodeKey: string; position: WorkflowPosition } | null = null;

/** 全部协议节点类型注册为 Vue 卡片节点（parallel 协议层支持，一并注册） */
const NODE_TYPES: WorkflowNodeType[] = [
  'start',
  'approval',
  'condition',
  'cc',
  'subflow',
  'plugin',
  'service',
  'parallel',
  'end',
];

/**
 * 流程连线模型：校验错误态整条描红（properties.error 由图数据投影注入），
 * 其余样式沿用 LogicFlow 主题；颜色统一走主题 CSS 变量适配暗色模式。
 */
class WorkflowEdgeModel extends PolylineEdgeModel {
  override getEdgeStyle() {
    const style = super.getEdgeStyle();
    if ((this.properties as { error?: boolean }).error) {
      style.stroke = 'var(--el-color-danger)';
    } else {
      style.stroke = 'var(--el-color-primary-light-5)';
    }
    style.strokeWidth = 2;
    return style;
  }
}

function renderGraph() {
  logicFlow.value?.render(
    toGraphData(props.document, {
      selectedNodeKey: props.selectedNodeKey,
      selectedEdgeKey: props.selectedEdgeKey,
      errorNodeKeys: props.errorNodeKeys,
      errorEdgeKeys: props.errorEdgeKeys,
      viewMode: viewMode.value,
    }),
  );
}

/** 消费当前拖拽写回：坐标抵达 DSL 后无需再次 render，LogicFlow 当前图即正确状态。 */
function consumePendingNodePosition(): boolean {
  const pending = pendingNodePosition;
  if (!pending) return false;
  const applied = props.document.settings.designer?.layout?.[pending.nodeKey];
  if (applied?.x === pending.position.x && applied.y === pending.position.y) {
    pendingNodePosition = null;
  }
  // 只要仍处于拖拽写回链路，都避免以旧文档覆写画布的即时位置。
  return true;
}

function resizeCanvas() {
  const element = canvasRef.value;
  const instance = logicFlow.value;
  if (!element || !instance) return;
  instance.resize(element.clientWidth, element.clientHeight);
}

/** LogicFlow 将缩放状态保留在画布内，控制条只派发命令，避免引入第二份图状态 */
function syncZoom() {
  const scale = logicFlow.value?.getTransform().SCALE_X ?? 1;
  zoomPercent.value = Math.round(scale * 100);
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

function setViewMode(mode: 'compact' | 'detailed') {
  if (viewMode.value === mode) return;
  viewMode.value = mode;
  renderGraph();
}

function allowNodeDrop(event: DragEvent) {
  if (props.readonly) return;
  event.preventDefault();
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy';
}

function dropNode(event: DragEvent) {
  if (props.readonly) return;
  event.preventDefault();
  const raw =
    event.dataTransfer?.getData('application/x-workflow-node-type') ||
    event.dataTransfer?.getData('text/plain');
  if (!NODE_TYPES.includes(raw as WorkflowNodeType) || ['start', 'end'].includes(raw)) return;
  const point = logicFlow.value?.getPointByClient({ x: event.clientX, y: event.clientY });
  if (!point) return;
  emit('dropNode', raw as WorkflowNodeType, point.canvasOverlayPosition);
}

function openNodePicker(event: Event) {
  if (props.readonly) return;
  const custom = event as CustomEvent<{
    nodeKey: string;
    direction: AddDirection;
    clientX: number;
    clientY: number;
  }>;
  const root = rootRef.value;
  if (!root || !custom.detail) return;
  const bounds = root.getBoundingClientRect();
  nodePicker.value = {
    nodeKey: custom.detail.nodeKey,
    direction: custom.detail.direction,
    left: Math.min(Math.max(custom.detail.clientX - bounds.left + 12, 12), bounds.width - 230),
    top: Math.min(Math.max(custom.detail.clientY - bounds.top + 12, 12), bounds.height - 270),
  };
}

function chooseNode(type: WorkflowNodeType) {
  const picker = nodePicker.value;
  if (!picker) return;
  emit('addFromNode', picker.nodeKey, type, picker.direction);
  nodePicker.value = null;
}

onMounted(() => {
  const element = canvasRef.value;
  if (!element) return;

  const instance = new LogicFlow({
    container: element,
    width: element.clientWidth,
    height: element.clientHeight,
    // 连线类型即自定义流程边（错误态描红）；节点视觉由 Vue 卡片承担
    edgeType: 'workflow-edge',
    grid: true,
    history: false, // DSL 文档是唯一事实源，撤销/重做由应用层按需实现
    edgeSelectedOutline: false,
    // 编辑态悬停展示连接锚点；只读态完全隐藏
    hideAnchors: props.readonly === true,
    adjustNodePosition: props.readonly !== true,
    adjustEdge: false,
    hoverOutline: false,
    nodeSelectedOutline: false,
    nodeTextEdit: false,
    textEdit: false,
    snapline: true,
    style: {
      polyline: { stroke: 'var(--el-color-primary-light-5)', strokeWidth: 2 },
      arrow: {
        fill: 'var(--el-color-primary-light-5)',
        stroke: 'var(--el-color-primary-light-5)',
        offset: 8,
        verticalLength: 5,
      },
      edgeText: {
        color: 'var(--el-text-color-secondary)',
        fontSize: 12,
        textWidth: 120,
        background: { fill: 'var(--el-fill-color-lighter)' },
      },
    },
  });
  logicFlow.value = instance;
  element.addEventListener('workflow-node-add', openNodePicker);

  instance.register({
    type: 'workflow-edge',
    view: PolylineEdge,
    model: WorkflowEdgeModel,
  });
  for (const type of NODE_TYPES) {
    register(
      { type: `workflow-${type}`, component: WorkflowNodeCard, view: WorkflowVueNodeView },
      instance,
    );
  }

  instance.on('node:click', ({ data }) => emit('selectNode', String(data.id)));
  instance.on('node:dragstart', ({ data }) => {
    const nodeKey = String(data.id);
    // LogicFlow 会将已多选节点作为一个拖拽组移动。产品侧只允许单节点拖动，
    // 因此在首个拖拽位移发生前收敛引擎内部选区，避免其他节点被联动移动。
    instance.clearSelectElements();
    instance.selectElementById(nodeKey, false);
    nodePicker.value = null;
    emit('selectNode', nodeKey);
  });
  instance.on('edge:click', ({ data }) => emit('selectEdge', String(data.id)));
  instance.on('blank:click', () => {
    nodePicker.value = null;
    if (!pendingNodePosition) renderGraph();
  });
  instance.on('node:drop', ({ data }) => {
    const nodeKey = String(data.id);
    const position = { x: data.x, y: data.y };
    pendingNodePosition = { nodeKey, position };
    emit('updateNodePosition', nodeKey, position);
  });
  // 锚点连线完成：以 DSL 结构层接管（生成协议边 key），临时边由重渲染替换
  instance.on('edge:connect', ({ data }) => {
    if (props.readonly) return;
    const source = String(data.sourceNodeId);
    const target = String(data.targetNodeId);
    if (source && target && source !== target) emit('connectEdge', source, target);
  });
  instance.on('graph:transform', syncZoom);
  renderGraph();

  resizeObserver = new ResizeObserver(resizeCanvas);
  resizeObserver.observe(element);
});

watch(
  () => [
    props.document,
    props.selectedNodeKey,
    props.selectedEdgeKey,
    props.errorNodeKeys,
    props.errorEdgeKeys,
  ],
  () => {
    if (consumePendingNodePosition()) return;
    renderGraph();
  },
  { deep: false },
);

onBeforeUnmount(() => {
  canvasRef.value?.removeEventListener('workflow-node-add', openNodePicker);
  resizeObserver?.disconnect();
  resizeObserver = null;
  logicFlow.value?.destroy();
  logicFlow.value = null;
});
</script>

<template>
  <div
    ref="rootRef"
    class="workflow-canvas"
    aria-label="流程画布"
    @dragover="allowNodeDrop"
    @drop="dropNode"
  >
    <div ref="canvasRef" class="workflow-canvas__graph" />
    <WorkflowViewModeToggle :mode="viewMode" @update-mode="setViewMode" />
    <div
      v-if="nodePicker"
      class="workflow-canvas__node-picker"
      :style="{ left: `${nodePicker.left}px`, top: `${nodePicker.top}px` }"
      role="menu"
      aria-label="选择节点类型"
    >
      <button type="button" role="menuitem" @click="chooseNode('approval')">
        <RiUser3Fill />流程节点
      </button>
      <button type="button" role="menuitem" @click="chooseNode('cc')">
        <RiSendPlaneFill />抄送节点
      </button>
      <button type="button" role="menuitem" @click="chooseNode('subflow')">
        <RiGitBranchFill />子流程
      </button>
      <button type="button" role="menuitem" @click="chooseNode('plugin')">
        <RiPuzzle2Fill />插件节点
      </button>
    </div>
    <WorkflowCanvasControls
      :zoom-percent="zoomPercent"
      @fit-view="fitView"
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

    :deep(.lf-node foreignObject) {
      overflow: visible;
    }

    // LogicFlow 会在节点点击后聚焦 SVG 容器；浏览器焦点框和引擎 outline
    // 都是矩形，会与起止节点的胶囊轮廓形成双层边框。
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

    :deep(.lf-anchor) {
      fill: var(--el-color-primary);
      stroke: var(--el-bg-color);
    }
  }

  &__node-picker {
    position: absolute;
    z-index: 10;
    display: flex;
    width: 224px;
    padding: 10px;
    flex-direction: column;
    gap: 8px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    box-shadow: var(--el-box-shadow);

    button {
      display: flex;
      height: 42px;
      padding: 0 14px;
      align-items: center;
      gap: 9px;
      color: var(--el-text-color-primary);
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color);
      border-radius: 7px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover {
        color: var(--el-color-primary);
        border-color: var(--el-color-primary);
      }

      svg { width: 20px; height: 20px; }
      &:nth-child(1) svg { color: var(--el-color-primary); }
      &:nth-child(2) svg { color: var(--el-color-success); }
      &:nth-child(3) svg { color: var(--el-color-warning); }
      &:nth-child(4) svg { color: #7c5ce7; }
    }
  }
}
</style>
