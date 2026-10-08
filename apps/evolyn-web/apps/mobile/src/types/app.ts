import type { FormSchemaDocument } from '@evolyn.do/form/schema';

export interface MobileAppItem {
  id: number;
  code: string;
  name: string;
  color: string | null;
  status: 'active' | 'archived';
}

export interface MobileAppPage {
  items: MobileAppItem[];
  nextCursor: string;
  hasMore: boolean;
}

export type MobileMenuTarget =
  | { type: 'form'; code: string; formType: 'standard' | 'workflow' }
  | { type: 'dashboard'; code: string }
  | { type: 'page'; code: string };

export interface MobileMenuNode {
  menuId: string;
  parentMenuId: string | null;
  type: 'group' | 'form' | 'dashboard' | 'page';
  name: string;
  icon: string | null;
  color: string | null;
  sortOrder: number;
  target: MobileMenuTarget | null;
  capabilities: { view: boolean; favorite: boolean };
  favorited?: boolean;
}

export interface MobileAppMenu {
  appCode: string;
  rootMenuIds: string[];
  nodeMap: Record<string, MobileMenuNode>;
  features: { workflow: boolean };
}

export interface MobileFormRuntimeBootstrap {
  formCode: string;
  name: string;
  publishedVersion: number;
  schemaRevision: string;
  protocolVersion: number;
  content: FormSchemaDocument;
  permissions?: {
    operations?: string[];
    addFields?: Record<string, { visible: boolean; editable: boolean }>;
  };
}

export type WorkflowTaskScope = 'pending' | 'completed' | 'cc-to-me';

export interface WorkflowTaskSummary {
  id: number;
  instanceNo: string;
  title: string;
  nodeName: string;
  starterName: string;
  status: string;
  createdAt: string;
}

export interface WorkflowTaskPage {
  items: WorkflowTaskSummary[];
  nextCursor: string;
}

export interface WorkflowInstanceSummary {
  id: number;
  instanceNo: string;
  status: string;
  createdAt: string;
}

export interface WorkflowInstancePage {
  items: WorkflowInstanceSummary[];
  nextCursor: string;
}
