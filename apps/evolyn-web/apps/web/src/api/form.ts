import type {
  FormDetail,
  FormDraftSaveResult,
  FormPage,
  FormPublishResult,
  FormRecordSubmitResult,
  FormRuntimeBootstrap,
  FormSchemaDocument,
  FormType,
} from '~/types';
// 表单资产域接口：与后端 /api/v1/forms*、/form-records 一一对应
// （见 evolyn-core internal/platform/form/controller/form.go）
import { http } from '@evolyn.do/utils';

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

/** 表单改名（PATCH /forms/:code，白名单字段） */
export function updateFormName(code: string, name: string): Promise<FormDetail> {
  return http.patch(`/forms/${code}`, { name });
}

/**
 * 保存草稿（PUT /forms/:code/draft）：全量替换 + 乐观锁。
 * 协议校验失败抛 errCode=FORM_SCHEMA_INVALID（ApiError.data.issues 为路径级问题清单），
 * 口令过期抛 FORM_REVISION_CONFLICT。
 */
export function saveFormDraft(
  code: string,
  draftRevision: number,
  content: FormSchemaDocument,
): Promise<FormDraftSaveResult> {
  return http.put(`/forms/${code}/draft`, { draftRevision, content });
}

/**
 * 发布（POST /forms/:code/publish）：白名单外控件抛 FORM_PUBLISH_UNSUPPORTED_FIELD
 *（ApiError.data.issues）；成功返回双口令。
 */
export function publishForm(code: string, draftRevision: number): Promise<FormPublishResult> {
  return http.post(`/forms/${code}/publish`, { draftRevision });
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
 * 校验失败抛 errCode=FORM_RECORD_INVALID（ApiError.data.fieldErrors 按字段键回填），
 * 版本口令不符抛 FORM_VERSION_CONFLICT。
 */
export function submitFormRecord(
  payload: {
    formCode: string;
    publishedVersion: number;
    schemaRevision: string;
    values: Record<string, unknown>;
  },
  signal?: AbortSignal,
): Promise<FormRecordSubmitResult> {
  return http.post('/form-records', payload, { signal });
}
