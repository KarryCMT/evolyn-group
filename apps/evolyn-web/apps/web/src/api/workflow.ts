import type {
  WorkflowDetailDto,
  WorkflowDraftSaveResult,
  WorkflowPageDto,
  WorkflowPublishResult,
  WorkflowVersionDetailDto,
  WorkflowVersionDto,
} from '~/types';
// 流程定义域接口：与后端 /api/v1/workflows 一一对应
// （见 evolyn-core internal/platform/workflow/controller/workflow.go）
import { http } from '@evolyn.do/utils';
import type { WorkflowDocument } from '@evolyn.do/workflow';

/**
 * 创建流程定义（POST /workflows）：草稿初始化为最小合法 DSL（start → end）。
 * formCode 可选：流程型表单的流程设计页懒建定义时传入，
 * 一条表单租户内至多绑定一条未删除定义（部分唯一索引兜底）。
 */
export function createWorkflow(payload: {
  name: string;
  description?: string;
  formCode?: string;
}): Promise<WorkflowDetailDto> {
  return http.post('/workflows', payload);
}

/**
 * 流程定义列表（游标分页，id 倒序）；formCode 精确过滤用于
 * 流程设计页按绑定表单定位定义（limit=1 即查唯一绑定）。
 */
export function listWorkflows(query: {
  limit?: number;
  cursor?: string;
  formCode?: string;
}): Promise<WorkflowPageDto> {
  return http.get('/workflows', query);
}

/** 流程定义详情（含 DSL 草稿全文与 draftRevision 口令） */
export function getWorkflow(code: string): Promise<WorkflowDetailDto> {
  return http.get(`/workflows/${code}`);
}

/** 修改流程定义基础信息（PATCH 白名单字段） */
export function updateWorkflow(
  code: string,
  payload: { name?: string; description?: string },
): Promise<WorkflowDetailDto> {
  return http.patch(`/workflows/${code}`, payload);
}

/**
 * 保存流程草稿（PUT /workflows/:code/draft）：draftRevision 乐观锁 +
 * DSL 全量替换；口令过期返回 WORKFLOW_REVISION_CONFLICT。
 * draft 为 Workflow DSL v1 全文档（含 settings.designer 画布坐标）。
 */
export function saveWorkflowDraft(
  code: string,
  payload: { draftRevision: number; draft: WorkflowDocument },
): Promise<WorkflowDraftSaveResult> {
  return http.put(`/workflows/${code}/draft`, payload);
}

/**
 * 发布流程定义（POST /workflows/:code/publish）：按草稿当前口令发布，
 * 经 DSL 严格校验与 Expr 预编译；失败返回 WORKFLOW_DEFINITION_INVALID
 * 且 data.issues 携带逐条定位问题（path/code/message）。
 */
export function publishWorkflow(
  code: string,
  payload: { draftRevision: number },
): Promise<WorkflowPublishResult> {
  return http.post(`/workflows/${code}/publish`, payload);
}

/** 发布版本列表（versionNo 降序） */
export function listWorkflowVersions(code: string): Promise<WorkflowVersionDto[]> {
  return http.get(`/workflows/${code}/versions`);
}

/** 版本详情：不可变发布快照全文（只读预览的数据源） */
export function getWorkflowVersion(
  code: string,
  versionNo: number,
): Promise<WorkflowVersionDetailDto> {
  return http.get(`/workflows/${code}/versions/${versionNo}`);
}
