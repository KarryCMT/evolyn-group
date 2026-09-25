export type IntelligentNodeType = 'trigger' | 'action' | 'end';

export type IntelligentActionType =
  | 'create-record'
  | 'update-record'
  | 'delete-record'
  | 'send-notification'
  | 'http-request'
  | 'data-transform'
  | 'condition';

export type IntelligentValueSourceType = 'constant' | 'trigger-field' | 'node-output';

/** 智能助手资源层使用的 JSON 值；配置文档不得保存组件实例或函数。 */
export type IntelligentJsonValue =
  | string
  | number
  | boolean
  | null
  | IntelligentJsonValue[]
  | { [key: string]: IntelligentJsonValue };

/** 表单字段的运行时值形态，用于来源过滤和自定义编辑器分派。 */
export type IntelligentFieldValueKind =
  | 'text'
  | 'number'
  | 'date'
  | 'choice'
  | 'multi-choice'
  | 'member'
  | 'members'
  | 'department'
  | 'departments'
  | 'address';

export interface IntelligentFormOption {
  code: string;
  name: string;
  formType: 'standard' | 'workflow';
  publishedVersion: number;
  disabled?: boolean;
}

export interface IntelligentFieldOption {
  /** 表单协议 v8 后稳定字段标识；历史协议回退为 widgetName。 */
  fieldId: string;
  widgetName: string;
  label: string;
  widgetType: string;
  valueKind: IntelligentFieldValueKind;
  required: boolean;
  choices?: readonly { label: string; value: string }[];
}

export interface IntelligentSourceFieldGroup {
  nodeId: string;
  nodeName: string;
  fields: readonly IntelligentFieldOption[];
}

export interface IntelligentActorOption {
  value: string;
  label: string;
}

/** 宿主应用向共享设计器注入业务目录，包内不直接依赖宿主 API。 */
export interface IntelligentDesignerResources {
  forms: readonly IntelligentFormOption[];
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
  loadFormFields?: (
    formCode: string,
    signal: AbortSignal,
  ) => Promise<readonly IntelligentFieldOption[]>;
}

export type FormTriggerActionType = 'create' | 'update' | 'delete';
export type FormTriggerUpdateScope = 'any-field' | 'specified-fields';
export type FormTriggerConditionMode = 'all' | 'any';
export type FormTriggerConditionOperator =
  | 'equals'
  | 'not-equals'
  | 'equals-any'
  | 'not-equals-any'
  | 'is-empty'
  | 'is-not-empty';

export interface FormTriggerAction {
  id: string;
  type: FormTriggerActionType;
  updateScope?: FormTriggerUpdateScope;
  fieldIds?: string[];
}

export interface FormTriggerCondition {
  id: string;
  fieldId: string;
  operator: FormTriggerConditionOperator;
  values: string[];
}

export interface IntelligentValueSource {
  type: IntelligentValueSourceType;
  value?: string;
  field?: string;
  nodeId?: string;
}

export interface IntelligentNodeFieldValueSource {
  type: 'node-field';
  nodeId: string;
  field: string;
}

export interface IntelligentCustomValueSource {
  type: 'custom';
  value: IntelligentJsonValue;
}

export interface IntelligentEmptyValueSource {
  type: 'empty';
}

export type IntelligentFieldValueSource =
  | IntelligentNodeFieldValueSource
  | IntelligentCustomValueSource
  | IntelligentEmptyValueSource;

export interface IntelligentFieldAssignment {
  targetFieldId: string;
  targetWidgetName: string;
  source: IntelligentFieldValueSource;
}

export type IntelligentUpdateTargetMode = 'form' | 'node';
export type IntelligentUpdateMatchMode = 'all' | 'any';
export type IntelligentUpdateFilterOperator =
  | 'equals'
  | 'not-equals'
  | 'equals-any'
  | 'not-equals-any'
  | 'is-empty'
  | 'is-not-empty';

/** 修改数据节点的一条筛选条件；一元运算符不保存 source。 */
export interface IntelligentUpdateFilter {
  id: string;
  targetFieldId: string;
  targetWidgetName: string;
  operator: IntelligentUpdateFilterOperator;
  source?: Exclude<IntelligentFieldValueSource, IntelligentEmptyValueSource>;
}

export interface IntelligentActionConfig {
  targetFormCode?: string;
  targetFormName?: string;
  targetFormPublishedVersion?: number;
  updateTargetMode?: IntelligentUpdateTargetMode;
  targetNodeId?: string;
  updateMatchMode?: IntelligentUpdateMatchMode;
  updateFilters?: IntelligentUpdateFilter[];
  createWhenNoMatch?: boolean;
  fieldAssignments?: IntelligentFieldAssignment[];
  requestMethod?: 'GET' | 'POST';
  requestUrl?: string;
  message?: string;
  expression?: string;
  valueSource?: IntelligentValueSource;
}

export interface IntelligentPosition {
  x: number;
  y: number;
}

export interface IntelligentTrigger {
  type: 'form' | 'schedule' | 'http';
  formCode?: string;
  formName?: string;
  eventName: string;
  actions: FormTriggerAction[];
  conditionMode: FormTriggerConditionMode;
  conditions: FormTriggerCondition[];
}

export interface IntelligentNode {
  id: string;
  type: IntelligentNodeType;
  name: string;
  description: string;
  position: IntelligentPosition;
  actionType?: IntelligentActionType;
  config?: IntelligentActionConfig;
}

export interface IntelligentEdge {
  id: string;
  source: string;
  target: string;
}

/** 智能助手设计器的前端文档协议，画布只消费该事实源。 */
export interface IntelligentDocument {
  version: '1.0';
  id: string;
  name: string;
  tags: string[];
  trigger: IntelligentTrigger;
  nodes: IntelligentNode[];
  edges: IntelligentEdge[];
}

export interface CreateIntelligentDocumentInput {
  id: string;
  name: string;
  tags?: string[];
  triggerType?: IntelligentTrigger['type'];
  triggerFormCode?: string;
  triggerFormName?: string;
}

export interface CreateIntelligentActionInput {
  actionType?: IntelligentActionType;
  name?: string;
  description?: string;
  config?: IntelligentActionConfig;
  /** 指定需要拆分的连线；省略时兼容为在结束节点前追加。 */
  sourceEdgeId?: string;
}
