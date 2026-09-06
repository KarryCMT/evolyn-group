<script setup lang="ts">
import LogicFlow, { PolylineEdge, PolylineEdgeModel } from '@logicflow/core';
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
}>();

const canvasRef = useTemplateRef<HTMLElement>('canvasRef');
const logicFlow: ShallowRef<LogicFlow | null> = shallowRef(null);
const zoomPercent = shallowRef(100);
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
    }),
  );
}

/** 消费当前拖拽写回：坐标抵达 DSL 后无需再次 render，LogicFlow 当前图即正确状态。 */
function consumePendingNodePosition(): boolean {
  const pending = pendingNodePosition;
  if (!pending) return false;
  const applied = props.document.settings.designer?.layout?.[pending.nodeKey];
  if (
    applied?.x === pending.position.x &&
    applied.y === pending.position.y
  ) {
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
  instance.on('edge:click', ({ data }) => emit('selectEdge', String(data.id)));
  instance.on('blank:click', () => {
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
  resizeObserver?.disconnect();
  resizeObserver = null;
  logicFlow.value?.destroy();
  logicFlow.value = null;
});
</script>

<template>
  <div class="workflow-canvas" aria-label="流程画布">
    <div ref="canvasRef" class="workflow-canvas__graph" />
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
}
</style>
