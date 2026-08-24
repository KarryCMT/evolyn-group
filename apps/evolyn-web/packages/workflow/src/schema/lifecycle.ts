import {
  WORKFLOW_SCHEMA_VERSION,
  type WorkflowDocument,
  type WorkflowEdge,
  type WorkflowField,
  type WorkflowFieldPermission,
  type WorkflowJsonValue,
  type WorkflowNode,
  type WorkflowNodeType,
} from './types';

/** 新建工作流默认拥有完整、可运行的起点—审批—结束主链。 */
export function createDefaultWorkflowDocument(
  fields: readonly WorkflowField[] = [],
): WorkflowDocument {
  const fieldPermissions = createFieldPermissions(fields);
  return {
    version: WORKFLOW_SCHEMA_VERSION,
    title: '未命名流程',
    nodes: [
      createNode('start', '流程发起节点', 480, 108, fieldPermissions),
      createNode('approval', '审批节点', 480, 274, fieldPermissions),
      createNode('end', '流程结束', 480, 440, fieldPermissions),
    ],
    edges: [
      { id: 'edge_start_approval', sourceNodeId: 'start', targetNodeId: 'approval' },
      { id: 'edge_approval_end', sourceNodeId: 'approval', targetNodeId: 'end' },
    ],
    settings: { versionName: 'V1' },
  };
}

/** 对外暴露克隆方法，阻止 Vue 响应式代理和 LogicFlow 实例进入持久化文档。 */
export function cloneWorkflowDocument(document: WorkflowDocument): WorkflowDocument {
  return JSON.parse(JSON.stringify(document)) as WorkflowDocument;
}

/** 初始阶段只接受安全的工作流主链文档；版本升级在这里集中处理。 */
export function normalizeWorkflowDocument(input: unknown): WorkflowDocument | null {
  if (!isRecord(input) || input.version !== WORKFLOW_SCHEMA_VERSION) return null;
  if (typeof input.title !== 'string' || !input.title.trim()) return null;
  if (!Array.isArray(input.nodes) || !Array.isArray(input.edges) || !isRecord(input.settings))
    return null;

  const nodes = input.nodes.flatMap((node) => normalizeNode(node));
  if (
    nodes.length !== input.nodes.length ||
    new Set(nodes.map((node) => node.id)).size !== nodes.length
  ) {
    return null;
  }

  const nodeIds = new Set(nodes.map((node) => node.id));
  const edges = input.edges.flatMap((edge) => normalizeEdge(edge, nodeIds));
  if (
    edges.length !== input.edges.length ||
    new Set(edges.map((edge) => edge.id)).size !== edges.length
  ) {
    return null;
  }

  return {
    version: WORKFLOW_SCHEMA_VERSION,
    title: input.title.trim(),
    nodes,
    edges,
    settings: { ...input.settings },
  };
}

function createNode(
  type: WorkflowNodeType,
  name: string,
  x: number,
  y: number,
  fieldPermissions: Record<string, WorkflowFieldPermission>,
): WorkflowNode {
  return {
    id: type,
    type,
    name,
    position: { x, y },
    fieldPermissions: cloneFieldPermissions(fieldPermissions),
    config: {},
  };
}

function createFieldPermissions(
  fields: readonly WorkflowField[],
): Record<string, WorkflowFieldPermission> {
  return Object.fromEntries(
    fields.map((field) => [
      field.id,
      { visible: true, editable: field.required !== true, confidential: false },
    ]),
  );
}

function cloneFieldPermissions(
  permissions: Record<string, WorkflowFieldPermission>,
): Record<string, WorkflowFieldPermission> {
  return Object.fromEntries(
    Object.entries(permissions).map(([fieldId, permission]) => [fieldId, { ...permission }]),
  );
}

function normalizeNode(value: unknown): WorkflowNode[] {
  if (!isRecord(value)) return [];
  if (!isWorkflowNodeType(value.type) || typeof value.id !== 'string' || !value.id.trim())
    return [];
  if (typeof value.name !== 'string' || !value.name.trim() || !isRecord(value.position)) return [];
  if (!isFiniteNumber(value.position.x) || !isFiniteNumber(value.position.y)) return [];
  if (!isRecord(value.fieldPermissions) || !isRecord(value.config)) return [];

  const fieldPermissions = Object.fromEntries(
    Object.entries(value.fieldPermissions).flatMap(([fieldId, permission]) => {
      if (!isRecord(permission)) return [];
      if (
        typeof permission.visible !== 'boolean' ||
        typeof permission.editable !== 'boolean' ||
        typeof permission.confidential !== 'boolean'
      ) {
        return [];
      }
      return [[fieldId, { ...permission }]];
    }),
  ) as Record<string, WorkflowFieldPermission>;
  if (Object.keys(fieldPermissions).length !== Object.keys(value.fieldPermissions).length)
    return [];

  if (!isJsonValue(value.config)) return [];
  return [
    {
      id: value.id.trim(),
      type: value.type,
      name: value.name.trim(),
      position: { x: value.position.x, y: value.position.y },
      fieldPermissions,
      config: { ...value.config },
    },
  ];
}

function normalizeEdge(value: unknown, nodeIds: Set<string>): WorkflowEdge[] {
  if (!isRecord(value)) return [];
  if (
    typeof value.id !== 'string' ||
    typeof value.sourceNodeId !== 'string' ||
    typeof value.targetNodeId !== 'string' ||
    !nodeIds.has(value.sourceNodeId) ||
    !nodeIds.has(value.targetNodeId)
  ) {
    return [];
  }
  return [{ id: value.id, sourceNodeId: value.sourceNodeId, targetNodeId: value.targetNodeId }];
}

function isWorkflowNodeType(value: unknown): value is WorkflowNodeType {
  return value === 'start' || value === 'approval' || value === 'end';
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

function isRecord(value: unknown): value is Record<string, WorkflowJsonValue> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isJsonValue(value: unknown, seen = new WeakSet<object>()): value is WorkflowJsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return true;
  if (typeof value === 'number') return Number.isFinite(value);
  if (typeof value !== 'object') return false;
  if (seen.has(value)) return false;

  seen.add(value);
  const valid = Array.isArray(value)
    ? value.every((item) => isJsonValue(item, seen))
    : (Object.getPrototypeOf(value) === Object.prototype ||
        Object.getPrototypeOf(value) === null) &&
      Object.values(value).every((item) => isJsonValue(item, seen));
  seen.delete(value);
  return valid;
}
