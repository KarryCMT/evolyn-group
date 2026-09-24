import type {
  CreateIntelligentActionInput,
  CreateIntelligentDocumentInput,
  FormTriggerAction,
  IntelligentDocument,
  IntelligentNode,
  IntelligentPosition,
  IntelligentTrigger,
} from './types';

const CANVAS_CENTER_X = 520;
const CANVAS_TOP = 130;
const NODE_GAP = 154;

function createId(prefix: string): string {
  const suffix = globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2);
  return `${prefix}_${suffix}`;
}

function triggerEventName(type: CreateIntelligentDocumentInput['triggerType']): string {
  if (type === 'schedule') return '按设定时间触发';
  if (type === 'http') return '接收到 HTTP 请求时';
  return '新增或修改数据时';
}

function createDefaultFormActions(): FormTriggerAction[] {
  return [
    { id: createId('trigger_action'), type: 'create' },
    {
      id: createId('trigger_action'),
      type: 'update',
      updateScope: 'specified-fields',
      fieldIds: ['employee_name'],
    },
    { id: createId('trigger_action'), type: 'delete' },
  ];
}

/** 将触发动作投影成画布节点摘要，保证节点与属性面板使用同一事实源。 */
export function describeIntelligentTrigger(trigger: IntelligentTrigger): string {
  if (trigger.type === 'schedule') return '按设定时间触发';
  if (trigger.type === 'http') return '接收到 HTTP 请求时';
  const labels = trigger.actions.map((action) => {
    if (action.type === 'create') return '新增';
    if (action.type === 'update') return '修改';
    return '删除';
  });
  return labels.length > 0 ? `${labels.join('、')}数据时` : '未配置触发动作';
}

/** 新建助手时建立最小可编辑图：触发节点直接连接结束节点。 */
export function createIntelligentDocument(
  input: CreateIntelligentDocumentInput,
): IntelligentDocument {
  const triggerType = input.triggerType ?? 'form';
  const triggerId = createId('trigger');
  const endId = createId('end');
  const trigger: IntelligentTrigger = {
    type: triggerType,
    formCode: input.triggerFormCode,
    formName: input.triggerFormName,
    eventName: triggerEventName(triggerType),
    actions: triggerType === 'form' ? createDefaultFormActions() : [],
    conditionMode: 'all',
    conditions: [],
  };
  return {
    version: '1.0',
    id: input.id,
    name: input.name,
    tags: [...(input.tags ?? [])],
    trigger,
    nodes: [
      {
        id: triggerId,
        type: 'trigger',
        name: input.triggerFormName || (triggerType === 'form' ? '当前表单' : '触发器'),
        description: describeIntelligentTrigger(trigger),
        position: { x: CANVAS_CENTER_X, y: CANVAS_TOP },
      },
      {
        id: endId,
        type: 'end',
        name: '结束',
        description: '',
        position: { x: CANVAS_CENTER_X, y: CANVAS_TOP + NODE_GAP },
      },
    ],
    edges: [
      {
        id: createId('edge'),
        source: triggerId,
        target: endId,
      },
    ],
  };
}

/** 更新触发器配置，并同步画布触发节点的可读摘要。 */
export function updateIntelligentTrigger(
  document: IntelligentDocument,
  patch: Partial<IntelligentTrigger>,
): IntelligentDocument {
  const trigger: IntelligentTrigger = {
    ...document.trigger,
    ...patch,
    actions: patch.actions ? patch.actions.map((action) => ({ ...action })) : document.trigger.actions,
    conditions: patch.conditions
      ? patch.conditions.map((condition) => ({ ...condition, values: [...condition.values] }))
      : document.trigger.conditions,
  };
  return {
    ...document,
    trigger,
    nodes: document.nodes.map((node) =>
      node.type === 'trigger'
        ? {
            ...node,
            name: trigger.formName || node.name,
            description: describeIntelligentTrigger(trigger),
          }
        : node,
    ),
  };
}

/** 在结束节点前插入执行节点，并保持纵向布局稳定。 */
export function addActionNode(
  document: IntelligentDocument,
  input: CreateIntelligentActionInput = {},
): IntelligentDocument {
  const endNode = document.nodes.find((node) => node.type === 'end');
  const predecessorEdge = endNode
    ? document.edges.find((edge) => edge.target === endNode.id)
    : undefined;
  if (!endNode || !predecessorEdge) return document;

  const actionCount = document.nodes.filter((node) => node.type === 'action').length;
  const actionId = createId('action');
  const node: IntelligentNode = {
    id: actionId,
    type: 'action',
    name: input.name ?? `执行节点 ${actionCount + 1}`,
    description: input.description ?? '待配置执行操作',
    position: { x: CANVAS_CENTER_X, y: endNode.position.y },
    actionType: input.actionType ?? 'create-record',
    config: input.config ? { ...input.config } : {},
  };
  const nodes = document.nodes.map((candidate) =>
    candidate.id === endNode.id
      ? {
          ...candidate,
          position: { ...candidate.position, y: candidate.position.y + NODE_GAP },
        }
      : candidate,
  );

  return {
    ...document,
    nodes: [
      ...nodes.filter((candidate) => candidate.type !== 'end'),
      node,
      ...nodes.filter((candidate) => candidate.type === 'end'),
    ],
    edges: [
      ...document.edges.filter((edge) => edge.id !== predecessorEdge.id),
      { id: createId('edge'), source: predecessorEdge.source, target: actionId },
      { id: createId('edge'), source: actionId, target: endNode.id },
    ],
  };
}

/** 以不可变方式更新节点，供属性面板与画布共同写入同一文档事实源。 */
export function updateIntelligentNode(
  document: IntelligentDocument,
  nodeId: string,
  patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>,
): IntelligentDocument {
  return {
    ...document,
    nodes: document.nodes.map((node) =>
      node.id === nodeId
        ? {
            ...node,
            ...patch,
            position: patch.position ? { ...patch.position } : node.position,
            config: patch.config ? { ...patch.config } : node.config,
          }
        : node,
    ),
  };
}

export function moveIntelligentNode(
  document: IntelligentDocument,
  nodeId: string,
  position: IntelligentPosition,
): IntelligentDocument {
  return {
    ...document,
    nodes: document.nodes.map((node) =>
      node.id === nodeId ? { ...node, position: { ...position } } : node,
    ),
  };
}

/** 只允许删除执行节点，把前后边重新接回并收拢其下方节点。 */
export function removeActionNode(
  document: IntelligentDocument,
  nodeId: string,
): IntelligentDocument {
  const node = document.nodes.find((candidate) => candidate.id === nodeId);
  if (node?.type !== 'action') return document;
  const incoming = document.edges.find((edge) => edge.target === nodeId);
  const outgoing = document.edges.find((edge) => edge.source === nodeId);
  if (!incoming || !outgoing) return document;

  return {
    ...document,
    nodes: document.nodes
      .filter((candidate) => candidate.id !== nodeId)
      .map((candidate) =>
        candidate.position.y > node.position.y
          ? {
              ...candidate,
              position: { ...candidate.position, y: candidate.position.y - NODE_GAP },
            }
          : candidate,
      ),
    edges: [
      ...document.edges.filter((edge) => edge.source !== nodeId && edge.target !== nodeId),
      { id: createId('edge'), source: incoming.source, target: outgoing.target },
    ],
  };
}
