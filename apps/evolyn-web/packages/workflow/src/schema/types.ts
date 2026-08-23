/** 当前工作流包可以读取与写回的文档版本。 */
export const WORKFLOW_SCHEMA_VERSION = 1 as const;
export type WorkflowSchemaVersion = typeof WORKFLOW_SCHEMA_VERSION;

/** 初始能力覆盖线性审批流；条件、抄送和子流程在后续版本扩展。 */
export type WorkflowNodeType = 'start' | 'approval' | 'end';

/** 可持久化的 JSON 值，隔离运行时图编辑器对象。 */
export type WorkflowJsonValue =
  | string
  | number
  | boolean
  | null
  | WorkflowJsonValue[]
  | { [key: string]: WorkflowJsonValue };

export interface WorkflowPosition {
  x: number;
  y: number;
}

export interface WorkflowFieldPermission {
  visible: boolean;
  editable: boolean;
  confidential: boolean;
}

/** 节点配置只存产品语义，画布框架的临时状态不得写入。 */
export interface WorkflowNode {
  id: string;
  type: WorkflowNodeType;
  name: string;
  position: WorkflowPosition;
  fieldPermissions: Record<string, WorkflowFieldPermission>;
  config: Record<string, WorkflowJsonValue>;
}

export interface WorkflowEdge {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
}

export interface WorkflowSettings {
  versionName?: string;
  [key: string]: WorkflowJsonValue | undefined;
}

/** 后端持久化与设计器编辑共用的工作流根文档。 */
export interface WorkflowDocument {
  version: WorkflowSchemaVersion;
  title: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  settings: WorkflowSettings;
}

/** 工作流不依赖表单包，应用层将字段适配为这个轻量结构注入。 */
export interface WorkflowField {
  id: string;
  label: string;
  required?: boolean;
}
