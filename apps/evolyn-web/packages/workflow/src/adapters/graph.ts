/**
 * DSL ↔ LogicFlow 适配层（Phase 9「LogicFlow Graph != Workflow Runtime Model」
 * 的落点）：DSL 文档是唯一事实源；LogicFlow 只消费投影出的图数据。
 * 画布坐标不属于 DSL 节点结构，持久化在 settings.designer.layout，
 * 缺失坐标的节点由分层自动布局兜底。
 */
import type LogicFlow from '@logicflow/core';
import { type WorkflowDocument, type WorkflowPosition, setNodePositions } from '../schema';

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
  subflow: { width: 196, height: 52 },
  plugin: { width: 196, height: 52 },
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
  viewMode?: 'compact' | 'detailed';
  /** 只读快照不投影编辑入口，避免历史版本出现可新增节点的误导状态。 */
  readonly?: boolean;
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
 * 将历史草稿中缺失的坐标一次性固化到设计器私有布局。此函数只在文档发生
 * 实际编辑时写回，避免普通选择或画布重绘再次运行自动布局而造成节点跳动。
 */
export function ensureDesignerLayout(document: WorkflowDocument): WorkflowDocument {
  const saved = document.settings.designer?.layout ?? {};
  const incomplete = document.nodes.some((node) => {
    const position = saved[node.key];
    return !position || !Number.isFinite(position.x) || !Number.isFinite(position.y);
  });
  return incomplete ? setNodePositions(document, resolveNodePositions(document)) : document;
}

/**
 * 分层自动布局：合法 DAG 使用拓扑顺序计算从 start 出发的最长层级，同层节点
 * 水平居中排开。环与不可达节点不会参与层级传播，而是进入固定兜底区域，
 * 从根本上避免编辑期非法环路把节点推到画布极远位置。
 */
export function computeAutoLayout(document: WorkflowDocument): Record<string, WorkflowPosition> {
  const outgoing = new Map<string, string[]>();
  const indegree = new Map(document.nodes.map((node) => [node.key, 0]));
  for (const edge of document.edges) {
    if (!indegree.has(edge.source) || !indegree.has(edge.target)) continue;
    outgoing.set(edge.source, [...(outgoing.get(edge.source) ?? []), edge.target]);
    indegree.set(edge.target, (indegree.get(edge.target) ?? 0) + 1);
  }

  // 合法 DAG 中按拓扑顺序传播最长层级；未被处理的节点属于环路。
  const layers = new Map<string, number>();
  for (const node of document.nodes) {
    if (node.type === 'start') layers.set(node.key, 0);
  }
  const queue = document.nodes
    .filter((node) => (indegree.get(node.key) ?? 0) === 0)
    .map((node) => node.key);
  while (queue.length > 0) {
    const key = queue.shift()!;
    const layer = layers.get(key);
    for (const target of outgoing.get(key) ?? []) {
      if (layer !== undefined) {
        layers.set(target, Math.max(layers.get(target) ?? -1, layer + 1));
      }
      const nextIndegree = (indegree.get(target) ?? 1) - 1;
      indegree.set(target, nextIndegree);
      if (nextIndegree === 0) queue.push(target);
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
    const detailed = state.viewMode === 'detailed' && !['start', 'end'].includes(node.type);
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
        height: detailed ? 76 : size.height,
        workflowType: node.type,
        nodeKey: node.key,
        label: node.name,
        subtitle: detailed ? nodeSubtitle(node) : '',
        detailed,
        readonly: state.readonly === true,
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
      type: 'workflow-edge',
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

/** 详细视图只投影人类可读摘要，不泄露运行协议内部字段。 */
function nodeSubtitle(node: WorkflowDocument['nodes'][number]): string {
  if (node.type === 'approval') return `负责人：${assigneeLabel(node.config.assignee)}`;
  if (node.type === 'cc') return `抄送人：${assigneeLabel(node.config.recipients)}`;
  if (node.type === 'subflow') {
    return `目标流程：${node.config.subflow?.definitionCode || '未选择'}`;
  }
  if (node.type === 'plugin') {
    const plugin = node.config.plugin;
    return plugin?.pluginCode
      ? `${plugin.pluginCode} · ${plugin.actionCode || '未选择动作'}`
      : '未选择插件';
  }
  if (node.type === 'service') {
    const method = node.config.service?.method || 'POST';
    return `${method} · ${node.config.service?.url || '未配置调用地址'}`;
  }
  if (node.type === 'condition') return '按条件选择后续分支';
  if (node.type === 'parallel') return '并行分流 / 汇聚';
  return '';
}

function assigneeLabel(assignee: WorkflowDocument['nodes'][number]['config']['assignee']): string {
  if (!assignee) return '未设置';
  if (assignee.type === 'user') return assignee.userIds?.length ? `${assignee.userIds.length} 位成员` : '未设置';
  const labels: Record<string, string> = {
    role: assignee.roleCode || '角色',
    form_field: assignee.formField || '表单成员字段',
    department: '指定部门',
    department_manager: '部门负责人',
    starter_manager: '发起人直属主管',
  };
  return labels[assignee.type] || '未设置';
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
