import type { FormDraftPayload, FormSubmitPayload, FormSubmitResult, FormValue } from '../types';

/**
 * 运行时与服务端的唯一注入边界（文档 §10.1）。
 * 运行时不导入宿主 API：认证头、租户上下文、接口地址与离线策略全部由 adapter 实现；
 * Web、移动 Web 与后续原生容器以替换 adapter 的方式复用同一运行时。
 * R1 仅消费 submit/saveDraft；成员、部门、附件等能力随 R2 重型字段接入，
 * 因此全部方法可选——宿主按已启用能力提供，运行时对缺失能力降级展示。
 */

/** 分页结果：大集合一律走字段级分页检索，禁止 bootstrap 携带全量数据（文档 §9.1）。 */
export interface Page<T> {
  items: T[];
  total: number;
  nextCursor?: string | null;
}

export interface MemberQuery {
  keyword?: string;
  cursor?: string | null;
  pageSize?: number;
}

export interface MemberValue {
  id: string;
  name: string;
  department?: string | null;
  avatar?: string | null;
}

export interface DepartmentQuery {
  keyword?: string;
  cursor?: string | null;
  pageSize?: number;
}

export interface DepartmentValue {
  id: string;
  name: string;
  parentId?: string | null;
}

export interface UploadInput {
  file: File;
  fieldKey: string;
}

/** 附件值：服务端返回的已上传文件描述，未完成上传的本地条目不得进入提交载荷（文档 §10.2）。 */
export interface FileValue {
  id: string;
  name: string;
  size: number;
  mimeType: string;
  url?: string;
}

export interface RelatedDataQuery {
  source: string;
  keyword?: string;
  cursor?: string | null;
  pageSize?: number;
}

export interface RelatedValue {
  id: string;
  title: string;
  [key: string]: FormValue;
}

export interface FormRuntimeAdapter {
  queryMembers?(input: MemberQuery, signal: AbortSignal): Promise<Page<MemberValue>>;
  queryDepartments?(input: DepartmentQuery, signal: AbortSignal): Promise<Page<DepartmentValue>>;
  uploadFile?(input: UploadInput, signal: AbortSignal): Promise<FileValue>;
  queryRelatedData?(input: RelatedDataQuery, signal: AbortSignal): Promise<Page<RelatedValue>>;
  submit?(payload: FormSubmitPayload, signal: AbortSignal): Promise<FormSubmitResult>;
  saveDraft?(payload: FormDraftPayload, signal: AbortSignal): Promise<void>;
}
