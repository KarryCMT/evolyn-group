import type { WorkflowDocument } from '../schema';

/** 应用层实现持久化和版本读取，工作流内核不感知路由、表单或 HTTP。 */
export interface WorkflowPersistenceAdapter {
  load: (workflowId: string) => Promise<WorkflowDocument | null>;
  save: (workflowId: string, document: WorkflowDocument) => Promise<WorkflowDocument>;
}
