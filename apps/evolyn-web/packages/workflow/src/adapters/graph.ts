/**
 * DSL ↔ LogicFlow 适配层（Phase 9「LogicFlow Graph != Workflow Runtime Model」
 * 的落点）：DSL 文档是唯一事实源；LogicFlow 只消费投影出的图数据。
 * 画布坐标不属于 DSL 节点结构，持久化在 settings.designer.layout，
 * 缺失坐标的节点由分层自动布局兜底。
 */
import type LogicFlow from '@logicflow/core';
import type { WorkflowDocument, WorkflowPosition } from '../schema';

/** 画布常量：层间距/同层间距与各类型节点尺寸（与节点卡片样式同口径） */
const LAYER_GAP = 150;
const SIBLING_GAP = 260;
const CANVAS_TOP = 90;
const CANVAS_CENTER_X = 420;

const NODE_SIZES: Record<string, { width: number; height: number }> = {
  start: { width: 174, height: 48 },
  end: { width: 174, height: 48 },
  approval: { width: 210, height: 58 },
  condition: { width: 196, height: 52 },
  cc: { width: 196, height: 52 },
  service: { width: 196, height: 52 },
  parallel: { width: 168, height: 52 },
};

export function nodeSize(type: string): { width: number; height: number } {
  return NODE_SIZES[type] ?? { width: 196, height: 52 };
}

/** 画布渲染状态：选中/错误高亮由设计器状态投影，LogicFlow 不自持语义 */
export interface WorkflowGraphState {
  selectedNodeKey?: string | null;
  selectedEdgeKey?: string | null;
  errorNodeKeys?: ReadonlySet<string>;
  errorEdgeKeys?: ReadonlySet<string>;
}

/**
 * 解析全部节点坐标：以 settings.designer.layout 为准，缺失的节点按
 * 分层自动布局补齐（新增节点、历史文档无坐标等场景），返回完整坐标表。
 */
export function resolveNodePositions(document: WorkflowDocument): Record<string, WorkflowPosition> {
  const saved = document.settings.designer?.layout ?? {};
  const auto = computeAutoLayout(document);
  const positions: Record<string, WorkflowPosition> = {};
  for (const node of document.nodes) {
    const current = saved[node.key];
    positions[node.key] =
      current && Number.isFinite(current.x) && Number.isFinite(current.y)
        ? current
        : (auto[node.key] ?? { x: 0, y: 0 });
  }
  return positions;
}

/**
 * 分层自动布局：从 start 沿边计算节点层级（最长路径），同层节点水平居中
 * 排开；并行/条件分支天然获得独立纵列。环与不可达节点做纵深防御
 * （层级封顶），布局永不抛错。
 */
export function computeAutoLayout(document: WorkflowDocument): Record<string, WorkflowPosition> {
  const outgoing = new Map<string, string[]>();
  for (const edge of document.edges) {
    outgoing.set(edge.source, [...(outgoing.get(edge.source) ?? []), edge.target]);
  }

  // 层级 = 从 start 出发的最长路径（并行分支各占一层深的独立链路）
  const layers = new Map<string, number>();
  const roots = document.nodes.filter((node) => node.type === 'start').map((node) => node.key);
  const queue: Array<{ key: string; layer: number }> = roots.map((key) => ({ key, layer: 0 }));
  const inQueue = new Set(roots);
  let guard = document.nodes.length * document.edges.length + document.nodes.length + 16;
  while (queue.length && guard-- > 0) {
    const { key, layer } = queue.shift()!;
    inQueue.delete(key);
    // 取更长路径（>= 保证已有层不被短路径回退）
    if ((layers.get(key) ?? -1) >= layer) continue;
    layers.set(key, layer);
    for (const target of outgoing.get(key) ?? []) {
      if (inQueue.has(target)) continue;
      inQueue.add(target);
      queue.push({ key: target, layer: layer + 1 });
    }
  }

  // 同层排布：按出现顺序水平居中；未达节点（脏数据）沉底兜底
  const byLayer = new Map<number, string[]>();
  for (const node of document.nodes) {
    const layer = layers.get(node.key) ?? 99;
    byLayer.set(layer, [...(byLayer.get(layer) ?? []), node.key]);
  }
  const positions: Record<string, WorkflowPosition> = {};
  let fallbackRow = 0;
  for (const [layer, keys] of byLayer) {
    if (layer === 99) {
      for (const key of keys) {
        positions[key] = { x: CANVAS_CENTER_X + 320, y: CANVAS_TOP + fallbackRow * 120 };
        fallbackRow += 1;
      }
      continue;
    }
    keys.forEach((key, index) => {
      const offset = (index - (keys.length - 1) / 2) * SIBLING_GAP;
      positions[key] = { x: CANVAS_CENTER_X + offset, y: CANVAS_TOP + layer * LAYER_GAP };
    });
  }
  return positions;
}

/** DSL 文档 → LogicFlow 渲染图数据（节点 id = 节点 key，边 id = 边 key） */
export function toGraphData(
  document: WorkflowDocument,
  state: WorkflowGraphState = {},
): LogicFlow.GraphConfigData {
  const positions = resolveNodePositions(document);
  const nodes: LogicFlow.NodeConfig[] = document.nodes.map((node) => {
    const size = nodeSize(node.type);
    return {
      id: node.key,
      type: `workflow-${node.type}`,
      x: positions[node.key]?.x ?? 0,
      y: positions[node.key]?.y ?? 0,
      text: '',
      // Vue 节点注册器使用 HtmlNodeModel；显式标记可拖拽，避免其在重渲染时
      // 因配置缺省回落为不可拖动。只读状态仍由画布的 adjustNodePosition 统一收口。
      draggable: true,
      properties: {
        width: size.width,
        height: size.height,
        workflowType: node.type,
        label: node.name,
        selected: node.key === state.selectedNodeKey,
        error: state.errorNodeKeys?.has(node.key) ?? false,
      },
    };
  });

  const edges: LogicFlow.EdgeConfig[] = document.edges.map((edge) => {
    const source = document.nodes.find((node) => node.key === edge.source);
    const isConditionSource = source?.type === 'condition';
    const label = !isConditionSource
      ? ''
      : edge.condition
        ? truncate(edge.condition.expression, 24)
        : '默认';
    return {
      id: edge.key,
      type: 'polyline',
      sourceNodeId: edge.source,
      targetNodeId: edge.target,
      text: label,
      properties: {
        isConditionEdge: isConditionSource,
        error: state.errorEdgeKeys?.has(edge.key) ?? false,
        selected: edge.key === state.selectedEdgeKey,
      },
    };
  });

  return { nodes, edges };
}

/** 从 LogicFlow 图数据回收画布坐标 → settings.designer.layout（写回 DSL 文档） */
export function collectLayout(
  document: WorkflowDocument,
  graphNodes: Array<{ id: string; x: number; y: number }>,
): Record<string, WorkflowPosition> {
  const layout: Record<string, WorkflowPosition> = {};
  const validKeys = new Set(document.nodes.map((node) => node.key));
  for (const node of graphNodes) {
    if (validKeys.has(node.id) && Number.isFinite(node.x) && Number.isFinite(node.y)) {
      layout[node.id] = { x: node.x, y: node.y };
    }
  }
  return layout;
}

function truncate(value: string, max: number): string {
  const normalized = value.trim().replace(/\s+/g, ' ');
  return normalized.length <= max ? normalized : `${normalized.slice(0, max - 1)}…`;
}
