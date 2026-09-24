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

export interface IntelligentActionConfig {
  targetFormCode?: string;
  targetFormName?: string;
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
}
