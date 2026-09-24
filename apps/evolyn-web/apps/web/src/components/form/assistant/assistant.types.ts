export type AssistantTriggerType = 'form' | 'schedule' | 'http';

export type AssistantFormAction = 'create' | 'update' | 'delete' | 'workflow-end' | 'node';

export interface AssistantFormOption {
  code: string;
  name: string;
  formType: 'standard' | 'workflow';
}

export interface AssistantDraft {
  name: string;
  triggerType: AssistantTriggerType;
  triggerFormCode?: string;
  triggerFormName?: string;
  tags: string[];
}

export interface AssistantListItem extends AssistantDraft {
  id: string;
  enabled: boolean;
  action: AssistantFormAction;
  targetFormCode?: string;
  targetFormName?: string;
  updatedAt: string;
}
