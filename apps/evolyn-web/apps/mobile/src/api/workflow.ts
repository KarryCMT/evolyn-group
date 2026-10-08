import type {
  WorkflowInstancePage,
  WorkflowTaskPage,
  WorkflowTaskScope,
} from '~/types/app';
import { http } from '@evolyn.do/utils';

export function listWorkflowTasks(scope: WorkflowTaskScope): Promise<WorkflowTaskPage> {
  return http.get('/workflow-tasks', { scope, limit: 30 });
}

export function listStartedWorkflowInstances(): Promise<WorkflowInstancePage> {
  return http.get('/workflow-instances', { scope: 'started-by-me', limit: 30 });
}
