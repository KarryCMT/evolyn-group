/**
 * Workflow 文档生命周期操作：纯函数集合，全部返回新文档（不可变更新），
 * 供设计器组件编排；画布坐标统一落在 settings.designer.layout，
 * 不进入节点/边结构（LogicFlow Graph != Workflow Runtime Model）。
 */
import {
  WORKFLOW_DSL_SCHEMA_VERSION,
  type WorkflowDocument,
  type WorkflowEdge,
  type WorkflowFieldPermission,
  type WorkflowNode,
  type WorkflowNodeType,
  type WorkflowPosition,
} from './types';

/** 节点类型目录（含 parallel：协议层完整支持，设计器素材面板按需暴露） */
export const WORKFLOW_NODE_TYPES: readonly WorkflowNodeType[] = [
  'start',
  'approval',
  'condition',
  'cc',
  'service',
  'parallel',
  'end',
];

/** 节点类型默认名称（新增节点时取「名称 + 序号」保证可辨识） */
const DEFAULT_NODE_NAMES: Record<WorkflowNodeType, string> = {
  start: '发起',
  approval: '审批',
  condition: '条件分支',
  cc: '抄送',
  service: '服务调用',
  parallel: '并行网关',
  end: '结束',
};

/**
 * 新建文档：与后端 minimalDraft 完全一致（start → end 最小合法链），
 * 保证设计器懒建定义与后端初始草稿零漂移。
 */
export function createWorkflowDocument(): WorkflowDocument {
  return {
    schemaVersion: WORKFLOW_DSL_SCHEMA_VERSION,
    nodes: [
      { key: 'start', type: 'start', name: '发起', config: {} },
      { key: 'end', type: 'end', name: '结束', config: {} },
    ],
    edges: [{ key: 'e_start_end', source: 'start', target: 'end' }],
    settings: {},
  };
}

/** 深拷贝：阻止 Vue 响应式代理进入持久化文档（JSON 往返即纯净结构） */
export function cloneWorkflowDocument(document: WorkflowDocument): WorkflowDocument {
  return JSON.parse(JSON.stringify(document)) as WorkflowDocument;
}

/**
 * 结构化归一：把后端草稿/发布快照还原为可信的编辑态文档。
 * 只做结构校验（不拦截编辑期非法配置——那是发布校验的职责），
 * 任何结构性损坏返回 null，由调用方回退到空文档。
 */
export function normalizeWorkflowDocument(input: unknown): WorkflowDocument | null {
  if (!isPlainObject(input) || input.schemaVersion !== WORKFLOW_DSL_SCHEMA_VERSION) return null;
  if (!Array.isArray(input.nodes) || !Array.isArray(input.edges) || !isPlainObject(input.settings))
    return null;

  const nodes: WorkflowNode[] = [];
  const seenKeys = new Set<string>();
  for (const value of input.nodes) {
    if (!isPlainObject(value)) return null;
    const key = typeof value.key === 'string' ? value.key.trim() : '';
    const type = value.type;
    if (!key || seenKeys.has(key)) return null;
    if (typeof type !== 'string' || !WORKFLOW_NODE_TYPES.includes(type as WorkflowNodeType))
      return null;
    if (typeof value.name !== 'string') return null;
    if (!isPlainObject(value.config)) return null;
    seenKeys.add(key);
    nodes.push({
      key,
      type: type as WorkflowNodeType,
      name: value.name,
      config: value.config as WorkflowNode['config'],
    });
  }
  const nodeKeys = new Set(nodes.map((node) => node.key));

  const edges: WorkflowEdge[] = [];
  const seenEdgeKeys = new Set<string>();
  for (const value of input.edges) {
    if (!isPlainObject(value)) return null;
    const key = typeof value.key === 'string' ? value.key.trim() : '';
    if (!key || seenEdgeKeys.has(key)) return null;
    if (typeof value.source !== 'string' || typeof value.target !== 'string') return null;
    if (!nodeKeys.has(value.source) || !nodeKeys.has(value.target)) return null;
    let condition: WorkflowEdge['condition'];
    if (value.condition != null) {
      if (!isPlainObject(value.condition) || typeof value.condition.expression !== 'string')
        return null;
      condition = { expression: value.condition.expression };
    }
    seenEdgeKeys.add(key);
    edges.push({ key, source: value.source, target: value.target, condition });
  }

  // 设计器坐标只接受有限数字，脏数据静默丢弃（缺失部分由自动布局兜底）
  const layout: Record<string, WorkflowPosition> = {};
  const rawLayout = (input.settings as Record<string, unknown>).designer;
  if (isPlainObject(rawLayout) && isPlainObject(rawLayout.layout)) {
    for (const [key, value] of Object.entries(rawLayout.layout)) {
      if (!nodeKeys.has(key) || !isPlainObject(value)) continue;
      if (Number.isFinite(value.x) && Number.isFinite(value.y)) {
        layout[key] = { x: value.x as number, y: value.y as number };
      }
    }
  }

  return {
    schemaVersion: WORKFLOW_DSL_SCHEMA_VERSION,
    nodes,
    edges,
    settings: { designer: { layout } },
  };
}

/** 生成文档内唯一的节点 key：`类型_序号`（与后端发布语义无耦合，仅文档内唯一） */
export function nextNodeKey(document: WorkflowDocument, type: WorkflowNodeType): string {
  const used = new Set(document.nodes.map((node) => node.key));
  let index = 1;
  let key = `${type}_${index}`;
  while (used.has(key)) {
    index += 1;
    key = `${type}_${index}`;
  }
  return key;
}

/** 生成文档内唯一的边 key */
export function nextEdgeKey(document: WorkflowDocument): string {
  const used = new Set(document.edges.map((edge) => edge.key));
  let index = 1;
  let key = `e_${index}`;
  while (used.has(key)) {
    index += 1;
    key = `e_${index}`;
  }
  return key;
}

/** 同类型节点的下一个展示名（「审批 2」……） */
export function nextNodeName(document: WorkflowDocument, type: WorkflowNodeType): string {
  const base = DEFAULT_NODE_NAMES[type];
  const count = document.nodes.filter((node) => node.type === type).length;
  return count === 0 ? base : `${base} ${count + 1}`;
}

/** 新增节点（position 由调用方按画布交互决定，写入 designer.layout） */
export function addNode(
  document: WorkflowDocument,
  type: WorkflowNodeType,
  position: WorkflowPosition,
): { document: WorkflowDocument; node: WorkflowNode } {
  const next = cloneWorkflowDocument(document);
  const node: WorkflowNode = {
    key: nextNodeKey(next, type),
    type,
    name: nextNodeName(next, type),
    config: createDefaultConfig(type),
  };
  next.nodes.push(node);
  setLayout(next, node.key, position);
  return { document: next, node };
}

/**
 * 在线性分支的既有边中插入节点：保留原边 key 与条件配置，新增节点到原目标的
 * 后继边。新流程的 start → end 默认边因此可直接演进为 start → 审批 → end。
 */
export function insertNodeOnEdge(
  document: WorkflowDocument,
  type: WorkflowNodeType,
  position: WorkflowPosition,
  edgeKey: string,
): { document: WorkflowDocument; node: WorkflowNode } {
  const originalEdge = document.edges.find((edge) => edge.key === edgeKey);
  const added = addNode(document, type, position);
  if (!originalEdge) return added;

  const before = updateEdge(added.document, edgeKey, { target: added.node.key });
  return {
    document: addEdge(before, added.node.key, originalEdge.target),
    node: added.node,
  };
}

/** 删除节点并级联清理关联边与画布坐标 */
export function removeNode(document: WorkflowDocument, nodeKey: string): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  next.nodes = next.nodes.filter((node) => node.key !== nodeKey);
  next.edges = next.edges.filter((edge) => edge.source !== nodeKey && edge.target !== nodeKey);
  removeLayout(next, nodeKey);
  return next;
}

/** 新增连线（自环/重复边由校验器把关，这里只负责结构写入） */
export function addEdge(
  document: WorkflowDocument,
  source: string,
  target: string,
): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  next.edges.push({ key: nextEdgeKey(next), source, target });
  return next;
}

export function removeEdge(document: WorkflowDocument, edgeKey: string): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  next.edges = next.edges.filter((edge) => edge.key !== edgeKey);
  return next;
}

/** 更新节点（浅合并 patch；config 变更请传完整 config 对象） */
export function updateNode(
  document: WorkflowDocument,
  nodeKey: string,
  patch: Partial<Pick<WorkflowNode, 'name'>> & { config?: WorkflowNode['config'] },
): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  const target = next.nodes.find((node) => node.key === nodeKey);
  if (!target) return document;
  if (patch.name !== undefined) target.name = patch.name;
  if (patch.config !== undefined) target.config = patch.config;
  return next;
}

/** 更新边（条件分支编辑的主通道） */
export function updateEdge(
  document: WorkflowDocument,
  edgeKey: string,
  patch: Partial<Pick<WorkflowEdge, 'source' | 'target'>> & {
    condition?: WorkflowEdge['condition'];
  },
): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  const target = next.edges.find((edge) => edge.key === edgeKey);
  if (!target) return document;
  if (patch.source !== undefined) target.source = patch.source;
  if (patch.target !== undefined) target.target = patch.target;
  if (patch.condition !== undefined) target.condition = patch.condition;
  return next;
}

/** 写入画布坐标（settings.designer.layout） */
export function setNodePosition(
  document: WorkflowDocument,
  nodeKey: string,
  position: WorkflowPosition,
): WorkflowDocument {
  const next = cloneWorkflowDocument(document);
  setLayout(next, nodeKey, position);
  return next;
}

function setLayout(document: WorkflowDocument, nodeKey: string, position: WorkflowPosition) {
  document.settings.designer ??= {};
  document.settings.designer.layout ??= {};
  document.settings.designer.layout[nodeKey] = { ...position };
}

function removeLayout(document: WorkflowDocument, nodeKey: string) {
  document.settings.designer?.layout && delete document.settings.designer.layout[nodeKey];
}

/** 按类型生成缺省 config（字段语义见 types；具体必填项由校验器提示） */
function createDefaultConfig(type: WorkflowNodeType): WorkflowNode['config'] {
  switch (type) {
    case 'approval':
      return { approvalMode: 'single' };
    case 'cc':
      return {};
    case 'service':
      return { service: { action: 'http', method: 'POST', url: '' } };
    case 'parallel':
      return { parallel: { role: 'split' } };
    default:
      return {};
  }
}

/** 字段权限缺省值：新字段在审批节点上默认可编辑（与运行时未配置语义一致） */
export function defaultFieldPermission(): WorkflowFieldPermission {
  return 'editable';
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
