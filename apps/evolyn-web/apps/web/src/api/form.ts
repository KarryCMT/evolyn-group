import type { QueryDocument } from '@evolyn.do/query';
import type {
  FormDetail,
  FormDraftSaveResult,
  FormPage,
  FormPublishResult,
  FormRecordDeleteResult,
  FormRecordMemberCard,
  FormRecordPage,
  FormRecordSubmitResult,
  FormRuntimeBootstrap,
  FormSchemaDocument,
  FormStorageJobDetail,
  FormType,
} from '~/types';
// 表单资产域接口：与后端 /api/v1/forms*、/form-records 一一对应
// （见 evolyn-core internal/platform/form/controller/form.go）
import { http } from '@evolyn.do/utils';

/** 表单权限组的稳定操作键，与后端 PermissionOp 字典一一对应。 */
export type FormPermissionOperation =
  | 'view'
  | 'add'
  | 'copy'
  | 'edit'
  | 'delete'
  | 'batch_print'
  | 'batch_modify'
  | 'import'
  | 'export'
  | 'workflow_owner_transfer'
  | 'workflow_terminate'
  | 'workflow_activate';

export interface FormPermissionSubjectInput {
  type: 'member' | 'department' | 'role';
  id: number;
}

export interface FormPermissionSubject extends FormPermissionSubjectInput {
  name: string;
}

export interface FormPermissionFieldRule {
  field: string;
  visible: boolean;
  editable: boolean;
}

export interface FormPermissionDataCondition {
  field: string;
  operator: string;
  value: unknown[];
}

export interface FormPermissionDataScope {
  match: 'all' | 'any';
  conditions: FormPermissionDataCondition[];
}

export interface FormPermissionGroup {
  code: string;
  name: string;
  description: string;
  enabled: boolean;
  operations: FormPermissionOperation[];
  fieldPermissions: FormPermissionFieldRule[];
  dataScope: FormPermissionDataScope;
  revision: number;
  subjects: FormPermissionSubject[];
}

export interface FormPermissionField {
  field: string;
  label: string;
  type: string;
  required: boolean;
}

export interface SaveFormPermissionGroupPayload {
  name: string;
  description: string;
  enabled: boolean;
  operations: FormPermissionOperation[];
  fieldPermissions: FormPermissionFieldRule[];
  dataScope: FormPermissionDataScope;
  subjectIds: FormPermissionSubjectInput[];
}

/**
 * 创建表单（POST /forms）：后端事务内完成 forms 配额校验，草稿初始化为空协议文档。
 * formType 创建时固化为表单资产事实，后续设计器从详情接口读取。
 * parentEntryCode 可选：传入时菜单节点挂到该分组下（须为同应用分组节点编码，
 * 非法分组抛 APP_MENU_PARENT_INVALID），否则挂应用根级。
 */
export function createForm(payload: {
  applicationId: number;
  name: string;
  formType: FormType;
  parentEntryCode?: string;
}): Promise<FormDetail> {
  return http.post('/forms', payload);
}

/** 应用内表单列表（游标分页，id 倒序） */
export function listForms(query: {
  applicationId: number;
  limit?: number;
  cursor?: string;
}): Promise<FormPage> {
  return http.get('/forms', query);
}

/** 表单详情（含草稿全文与 draftRevision 口令） */
export function getForm(code: string): Promise<FormDetail> {
  return http.get(`/forms/${code}`);
}

/**
 * 修改表单基础展示信息（PATCH /forms/:code，白名单字段）。名称由 forms
 * 保存；图标与颜色由服务端在同一事务内同步到应用菜单节点。
 */
export function updateForm(
  code: string,
  payload: { name?: string; icon?: string; color?: string },
): Promise<FormDetail> {
  return http.patch(`/forms/${code}`, payload);
}

/**
 * 删除表单（DELETE /forms/:code）：后端软删表单，并在同一事务内摘除应用菜单
 * 节点及其收藏关联；已发布版本仍保留以支持历史记录追溯。
 */
export function deleteForm(code: string): Promise<null> {
  return http.delete(`/forms/${code}`);
}

/** 表单改名兼容入口；新页面应优先使用 updateForm 一次提交完整展示信息。 */
export function updateFormName(code: string, name: string): Promise<FormDetail> {
  return updateForm(code, { name });
}

/**
 * 保存草稿（PUT /forms/:code/draft）：全量替换 + 乐观锁。
 * 协议校验失败抛 errCode=FORM_SCHEMA_INVALID（ApiError.data.issues 为路径级问题清单），
 * 口令过期抛 FORM_REVISION_CONFLICT。
 */
export function saveFormDraft(
  code: string,
  draftRevision: number,
  protocolVersion: number,
  content: FormSchemaDocument,
): Promise<FormDraftSaveResult> {
  return http.put(`/forms/${code}/draft`, { draftRevision, protocolVersion, content });
}

/**
 * 发布（POST /forms/:code/publish）：白名单外控件抛 FORM_PUBLISH_UNSUPPORTED_FIELD
 *（ApiError.data.issues）；成功返回双口令。
 */
export function publishForm(code: string, draftRevision: number): Promise<FormPublishResult> {
  return http.post(`/forms/${code}/publish`, { draftRevision });
}

/**
 * 查询表单结构变更任务（GET /forms/:code/storage-jobs/:jobId）：发布返回
 * async=true 后轮询执行状态；SUCCEEDED 后发布版本才真正生效。
 */
export function getFormStorageJob(code: string, jobId: number): Promise<FormStorageJobDetail> {
  return http.get(`/forms/${code}/storage-jobs/${jobId}`);
}

/**
 * 重试终态失败的结构变更任务（POST .../retry）：仅 FAILED 可重试，复位后
 * 由服务端 Worker 重新执行，调用方继续轮询同 jobId。
 */
export function retryFormStorageJob(code: string, jobId: number): Promise<FormStorageJobDetail> {
  return http.post(`/forms/${code}/storage-jobs/${jobId}/retry`);
}

/**
 * 运行时引导（GET /applications/code/:appCode/forms/:formCode/runtime）：
 * 返回已发布快照与双口令；未发布抛 FORM_NOT_PUBLISHED。
 */
export function getFormRuntime(
  appCode: string,
  formCode: string,
  signal?: AbortSignal,
): Promise<FormRuntimeBootstrap> {
  return http.get(`/applications/code/${appCode}/forms/${formCode}/runtime`, undefined, signal);
}

/**
 * 提交记录（POST /form-records）：服务端按发布快照终审。
 * 字段校验失败抛 errCode=FORM_RECORD_INVALID；表单级提交校验失败抛
 * FORM_RECORD_VALIDATION_FAILED（data.validatorErrors 按规则下标返回），
 * 版本口令不符抛 FORM_VERSION_CONFLICT。
 */
export function submitFormRecord(
  payload: {
    appCode: string;
    entryCode?: string;
    formCode: string;
    publishedVersion: number;
    schemaRevision: string;
    // 具体字段包装类型由 @evolyn.do/form/runtime-core 维护；Web API 层只负责透传。
    values: Record<string, unknown>;
    hasResult: true;
    dataOpId: string;
  },
  signal?: AbortSignal,
): Promise<FormRecordSubmitResult> {
  return http.post('/form-records', payload, { signal });
}

/**
 * 查询表单记录（POST /forms/:code/records）。Query DSL 直接作为 JSON 请求体
 * 上送，避免复杂筛选条件超出 URL 长度上限；后端仍会按发布快照白名单重新
 * 校验，客户端不具备 JSONB 路径、物理列名或 SQL 输入能力。
 */
export function listFormRecords(
  code: string,
  query: QueryDocument & { keyword?: string },
  signal?: AbortSignal,
): Promise<FormRecordPage> {
  return http.post(`/forms/${code}/records`, query, { signal });
}

/** 永久删除数据管理中勾选的记录；服务端仍会逐条复核数据范围权限。 */
export function deleteFormRecords(
  code: string,
  recordIds: number[],
): Promise<FormRecordDeleteResult> {
  return http.delete(`/forms/${code}/records`, { recordIds });
}

/** 数据管理成员卡片：受当前表单记录查看权限约束，响应不含完整成员档案。 */
export function getFormRecordMemberCard(
  code: string,
  memberCode: string,
  signal?: AbortSignal,
): Promise<FormRecordMemberCard> {
  return http.get(
    `/forms/${code}/records/member-cards/${encodeURIComponent(memberCode)}`,
    undefined,
    signal,
  );
}

/** 每次用户提交生成独立幂等键；同一次 HTTP 调用及其网络重放复用同一载荷。 */
export function createFormDataOperationId(): string {
  return globalThis.crypto.randomUUID();
}

/** 读取表单全部权限组（含停用项）。 */
export function listFormPermissionGroups(formCode: string): Promise<FormPermissionGroup[]> {
  return http.get(`/forms/${formCode}/permission-groups`);
}

/** 读取当前表单可配置权限的字段清单。 */
export function listFormPermissionFields(formCode: string): Promise<FormPermissionField[]> {
  return http.get(`/forms/${formCode}/permission-fields`);
}

/** 创建表单权限组。 */
export function createFormPermissionGroup(
  formCode: string,
  payload: SaveFormPermissionGroupPayload,
): Promise<FormPermissionGroup> {
  return http.post(`/forms/${formCode}/permission-groups`, payload);
}

/** 全量更新表单权限组；baseRevision 用于处理并发配置。 */
export function updateFormPermissionGroup(
  formCode: string,
  groupCode: string,
  payload: SaveFormPermissionGroupPayload & { baseRevision: number },
): Promise<FormPermissionGroup> {
  return http.put(`/forms/${formCode}/permission-groups/${groupCode}`, payload);
}

/** 删除表单权限组。 */
export function deleteFormPermissionGroup(formCode: string, groupCode: string): Promise<null> {
  return http.delete(`/forms/${formCode}/permission-groups/${groupCode}`);
}
