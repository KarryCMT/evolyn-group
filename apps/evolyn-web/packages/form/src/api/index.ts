import { request } from '@firefly.do/utils';
import type { AxiosRequestConfig, AxiosResponse } from 'axios';
import type { PluginRuntime } from '../types';

// 平台文件结构由插件包自行维护，避免公共组件反向依赖 admin 工程。
export interface PlatformFile {
  id: string;
  clientId: string | null;
  fileId: string;
  fileName: string;
  fileSuffix: string;
  fileSizeByte: string;
  expiration: string | null;
  expirationTime: string | null;
  status: string | null;
  createTime: string;
  updateTime: string | null;
  url: string;
  downloadUrl: string;
  path: string;
}

export interface PlatformFunctionListType {
  id?: string | number; //函数ID（用于详情查询）
  pluginId: string | number | null; //插件ID（用于列表查询）
}

// 更新当前选中函数时只提交函数数据，不携带设计器 activeMenu 状态。
export interface PluginFunctionUpdateType {
  id: string | number;
  pluginId: string | number;
  functionName: string;
  functionDescription: string;
  functionType: string;
  /** 当前函数在所属前端/后端分组内从 0 开始的排序号。 */
  seq: number;
  requestParameter: PluginFunctionParameter<PluginFunctionRequestField>;
  responseParameter: PluginFunctionParameter<PluginFunctionResponseField>;
  runtime: string;
  sourceCode: string;
}

// 新增后端函数时提交当前插件 ID 和用户选择的运行时。
export interface PluginFunctionCreateType {
  pluginId: string | number;
  runtime: PluginRuntime;
}

// 新增函数接口返回后端生成的稳定标识和默认函数信息。
export interface PluginFunctionCreateResult {
  [key: string]: unknown;
  id: string | number;
  functionKey?: string;
  functionName?: string;
  functionDescription?: string | null;
  functionType?: string;
  runtime?: string;
  sourceCode?: string;
  seq?: number | null;
}

// 复制函数时仅提交待复制函数的稳定 ID。
export interface PluginFunctionCopyType {
  id: string | number;
}

export interface PluginCenterQuery {
  keyword?: string;
  pluginName?: string;
  pageIndex?: number;
  pageSize?: number;
}

// 插件详情查询仅允许使用稳定的 pluginKey，避免列表 id 与插件标识混用。
export interface PluginCenterDetailQuery {
  pluginKey: string;
}

export interface UpdatePluginStatusType {
  pluginKey: string | number;
  isEnabled: boolean;
}

export interface CheckPluginReferenceType {
  functionKey?: string | number;
  pluginKey: string | number;
  sourceType?: string | number;
  version?: number;
}

export interface PluginCopyType {
  pluginKey: string | number;
}

/** 插件复制接口的完整响应，复制结果结构由后端按插件数据返回。 */
export type PluginCopyResponse = AxiosResponse<{
  state: number;
  message: string;
  data: Record<string, unknown> | null;
  extras: unknown;
  traceId: string;
}>;

/** 插件引用检查接口返回的业务数据。 */
export interface CheckPluginReferenceResult {
  pluginKey: string;
  referenceCount: number;
}

/** 插件引用检查接口的完整响应。 */
export type CheckPluginReferenceResponse = AxiosResponse<{
  state: number;
  message: string;
  data: CheckPluginReferenceResult | null;
  extras: unknown;
  traceId: string;
}>;

// 插件导出通过稳定标识定位插件；未传版本号时由后端导出最新版本。
export interface PluginExportQuery {
  pluginKey: string;
  version?: number;
}

// 导出接口直接返回插件包二进制流，调用方需要保留完整响应以读取文件名响应头。
export type PluginExportResponse = AxiosResponse<Blob>;

export interface PluginCenterForm {
  id?: string | number | null;
  pluginIcon: string;
  pluginKey: string;
  /** 插件定义的数据版本，调试接口通过该版本定位当前插件定义。 */
  version?: number | null;
  pluginThemeColor?: string;
  pluginIconColor?: string;
  pluginName: string;
  pluginOverview: string;
  pluginDetail: string;
  pluginDetailFile: PlatformFile[];
  helpDoc: string;
  publishStatus?: boolean;
  pluginDesign?: unknown;
  authentication?: PluginAuthenticationConfig;
  globalTemplate?: PluginAuthTemplateSection;
  pluginConfig?: Partial<PluginAgentConfig>;
}

/** 插件管理详情中的后端函数摘要，用于只读展示函数基本信息。 */
export interface PluginBackendFunctionSummary {
  [key: string]: unknown;
  id: string | number;
  functionKey: string;
  functionName: string;
  functionDescription?: string | null;
  runtime: string;
  version: number;
  seq: number | null;
  status: boolean;
  createTime?: string;
  createBy?: string | number;
  createByName?: string;
}

export interface PluginAuthTemplateField {
  [key: string]: unknown;
  id?: string | null;
  fieldKey: string;
  fieldLabel: string;
  description?: string;
  widgetName: string;
  dataType: string;
  isHidden?: boolean;
  isEnabled?: boolean;
  isRequired?: boolean;
  fieldConf?: Record<string, unknown>;
  defaultValue?: unknown;
  sort?: number;
}

export interface PluginAuthTemplateSection {
  [key: string]: unknown;
  fields: PluginAuthTemplateField[];
}

export interface PluginAuthTemplate {
  [key: string]: unknown;
  auth_type: string;
  conf_template: PluginAuthTemplateSection;
  credential_template: PluginAuthTemplateSection;
  oauth2_request_template?: PluginAuthTemplateSection;
  refresh_token_request_template?: PluginAuthTemplateSection;
  app_auth_template?: PluginAuthTemplateSection;
}

export interface PluginAuthTemplateListResponse {
  plugin_auth_template_list: PluginAuthTemplate[];
}

// 插件级身份验证配置，与 globalTemplate、pluginDesign 在详情和保存参数中保持平级。
export interface PluginAuthenticationConfig {
  type: string;
  conf_template: PluginAuthTemplateSection;
  credential_template?: PluginAuthTemplateSection;
  oauth2_request_template?: PluginAuthTemplateSection;
  refresh_token_request_template?: PluginAuthTemplateSection;
  app_auth_template?: PluginAuthTemplateSection;
}

export interface OwnerPluginUpdateRequest {
  pluginKey: string;
  pluginDetail: string;
  pluginDetailFile: PlatformFile[];
  pluginIcon: string;
  pluginIconColor: string;
  pluginName: string;
  pluginOverview: string;
  pluginThemeColor: string;
  demo?: string;
  docs?: string;
  authentication?: PluginAuthenticationConfig;
  globalTemplate?: PluginAuthTemplateSection;
  pluginDesign?: unknown;
}

export interface PluginTriggerTemplateField {
  [key: string]: unknown;
  id: string | number | null;
  fieldKey?: string;
  fieldLabel?: string;
  // label 仅用于兼容旧函数模板，新的函数接口统一使用 fieldLabel。
  label?: string;
  description?: string;
  widgetName: string;
  dataType: string;
  isHidden?: boolean;
  isEnabled?: boolean;
  isRequired?: boolean;
  fieldConf?: Record<string, unknown>;
  defaultValue?: unknown;
  sort?: number;
}

// 函数请求参数和返回参数由设计器分别维护为 JSON 字段列表。
export interface PluginFunctionParameter<T> {
  fields: T[];
}

// 函数请求字段直接使用 widgetName 识别控件，不再提交 fieldType。
export type PluginFunctionRequestField = PluginTriggerTemplateField;

// 函数返回参数直接使用后端字段结构，vector 子字段递归存放在 fieldConf.fields。
export interface PluginFunctionResponseField {
  [key: string]: unknown;
  id: string | number | null;
  fieldKey: string;
  fieldLabel: string;
  description?: string;
  widgetName: string;
  dataType: string;
  isHidden?: boolean;
  isEnabled?: boolean;
  isRequired?: boolean;
  defaultValue?: unknown;
  sort?: number;
  fieldConf: {
    fields?: PluginFunctionResponseField[];
    [key: string]: unknown;
  };
}

export interface PluginTriggerTemplate {
  [key: string]: unknown;
  fields?: PluginTriggerTemplateField[];
}

export interface PluginTrigger {
  [key: string]: unknown;
  id: string | number;
  pluginId: string | number;
  functionKey: string;
  version: number;
  requestParameter: PluginTriggerTemplate;
  createTime: string;
  functionName: string;
  functionDescription?: string | null;
  responseParameter: PluginFunctionParameter<PluginFunctionResponseField>;
  runtime: string;
  seq: number | null;
  sourceCode: string;
  updateTime: string;
  runtimeList: string;
}
export interface PluginApplet {
  [key: string]: unknown;
  id: string | number;
  pluginId: string | number;
  functionKey: string;
  version: number;
  requestParameter: PluginTriggerTemplate;
  createTime: string;
  functionName: string;
  functionDescription?: string | null;
  responseParameter: PluginFunctionParameter<PluginFunctionResponseField>;
  seq: number | null;
  sourceCode: string;
  updateTime: string;
}

// 函数更新接口返回完整函数数据，用于将后端生成的字段 ID 回填到设计器。
export interface PluginFunctionDetail {
  [key: string]: unknown;
  id: string | number;
  functionKey?: string;
  functionName: string;
  functionDescription?: string | null;
  functionType?: string;
  requestParameter: PluginTriggerTemplate;
  responseParameter: PluginFunctionParameter<PluginFunctionResponseField>;
  runtime?: string | null;
  sourceCode: string;
  seq?: number | null;
}

export interface PluginTriggerListResponse {
  plugin_trigger_list: PluginTrigger[];
}
export interface PluginAppletListResponse {
  plugin_applet_list: PluginApplet[];
}

// 插件函数列表按后端函数和前端扩展分组返回。
export interface PluginFunctionListResponse {
  backendList: PluginTrigger[];
  frontendList: PluginApplet[];
}

export interface PluginFieldMetadata {
  [key: string]: unknown;
  field_type: string;
  resolver_types: string[];
  field_schemas: Record<string, unknown>;
}

export interface PluginFieldMetadataResponse {
  fields: PluginFieldMetadata[];
}

export interface PluginRuntimeRunRequest {
  pluginKey: string;
  version: number;
  functionKey: string;
  authConf?: Record<string, unknown>;
  agentConf?: Record<string, unknown>;
  triggerConf?: Record<string, unknown>;
}

// 导入确认中的字段配置，与预览接口返回的原始插件包字段保持一致。
export interface ImportPluginCenterConfirmField {
  dataType: string;
  defaultValue: Record<string, unknown>;
  description: string;
  fieldConf: Record<string, unknown>;
  fieldKey: string;
  fieldLabel: string;
  id: number;
  isEnabled: boolean;
  isHidden: boolean;
  isRequired: boolean;
  sort: number;
  widgetName: string;
}

// 导入确认中的字段集合，同时用于身份认证配置和全局参数模板。
export interface ImportPluginCenterConfirmFieldSection {
  fields: ImportPluginCenterConfirmField[];
}

// 刷新令牌请求配置由后端按键名解析具体配置项。
export interface ImportPluginCenterRefreshTokenRequestConfig {
  key: string;
}

// 插件包中的身份认证配置。
export interface ImportPluginCenterConfirmAuthentication {
  authConfig: ImportPluginCenterConfirmFieldSection;
  authType: string;
  oauth2Url: string;
  refreshTokenRequestConfigDtoList: ImportPluginCenterRefreshTokenRequestConfig[];
}

// 插件包中的函数定义，参数结构保留为后端返回的动态 JSON 对象。
export interface ImportPluginCenterConfirmFunction {
  functionKey: string;
  functionName: string;
  functionDescription?: string;
  functionType: string;
  requestParameter: Record<string, unknown>;
  responseParameter: Record<string, unknown>;
  runtime: string;
  seq: number;
  sourceCode: string;
}

// 插件详情附件信息，与导入确认接口请求字段保持一致。
export interface ImportPluginCenterConfirmFile {
  bucket: string;
  clientId: number;
  createTime: string;
  downloadUrl: string;
  expiration: number;
  expirationTime: string;
  fileId: string;
  fileMd5: string;
  fileName: string;
  filePath: string;
  fileSizeByte: number;
  fileSuffix: string;
  id: number;
  status: boolean;
  statusStr: string;
  updateTime: string;
  url: string;
}

// 预览接口返回并由确认接口提交的完整插件数据。
export interface ImportPluginCenterConfirmData {
  authentication: ImportPluginCenterConfirmAuthentication;
  demo: string;
  docs: string;
  functions: ImportPluginCenterConfirmFunction[];
  globalTemplate: ImportPluginCenterConfirmFieldSection;
  pluginDetail: string;
  pluginDetailFile: ImportPluginCenterConfirmFile[];
  pluginIcon: string;
  pluginIconColor: string;
  pluginKey: string;
  pluginName: string;
  pluginOverview: string;
  pluginThemeColor: string;
  pluginVersion: string;
  version: number;
}

// 插件导入确认请求，importData 来自预览接口的 data.data。
export interface ImportPluginCenterConfirmType {
  importData: ImportPluginCenterConfirmData;
  overwrite: boolean;
}

export interface PluginRuntimeRunResponse {
  data: unknown;
  extras: unknown;
  message: string;
  state: number;
  traceId: string;
}

// 插件配置支持基础字段值，也兼容成员、部门等对象或数组型扩展字段。
export type PluginAgentResolverArg =
  | string
  | number
  | boolean
  | Record<string, unknown>
  | unknown[]
  | null;

// 单个配置字段使用后端持久化 id 与用户填写值建立关联。
export interface PluginAgentAssignment {
  id: string | number;
  resolver_arg: PluginAgentResolverArg;
}

// 插件实例配置由字段赋值列表组成，详情接口可能返回空对象。
export interface PluginAgentConfig {
  assignments: PluginAgentAssignment[];
}

// 更新租户插件实例配置的请求结构。
export interface PluginAgentUpdateRequest {
  pluginKey: string;
  pluginConfig: PluginAgentConfig;
}

// 导入预览中的详情附件字段与 /plugin/center/import/preview 返回结构保持一致。
export interface PluginCenterImportPreviewFile {
  [key: string]: unknown;
  id: string;
  clientId: string | null;
  fileId: string;
  fileName: string;
  fileSuffix: string;
  fileSizeByte: string;
  fileMd5: string | null;
  filePath: string | null;
  bucket: string | null;
  expiration: string | null;
  expirationTime: string | null;
  status: string | null;
  statusStr: string | null;
  createTime: string;
  updateTime: string | null;
  url: string;
  downloadUrl: string;
}

// 导入预览中的函数数据直接保留接口字段，供函数数量与源码预览使用。
export interface PluginCenterImportPreviewFunction {
  [key: string]: unknown;
  functionKey: string;
  functionName: string;
  functionDescription?: string;
  functionType: string;
  runtime: string;
  sourceCode: string;
  requestParameter: PluginFunctionParameter<PluginFunctionRequestField>;
  responseParameter: PluginFunctionParameter<PluginFunctionResponseField>;
  seq: number;
}

// 插件导入预览对象与接口 data 字段一一对应，不再转换为旧版设计器字段。
export interface PluginCenterImportPreview {
  [key: string]: unknown;
  pluginKey: string;
  pluginIcon: string;
  pluginIconColor: string;
  pluginThemeColor: string;
  pluginName: string;
  version: number;
  pluginVersion: string;
  pluginOverview: string;
  pluginDetail: string;
  pluginDetailFile: PluginCenterImportPreviewFile[];
  globalTemplate: PluginAuthTemplateSection;
  functions: PluginCenterImportPreviewFunction[];
}

// 查询插件详情，编辑时用于回显完整字段
export function getPluginCenterDetail(params: PluginCenterDetailQuery) {
  return request({
    url: `/plugin/center/get`,
    method: 'get',
    params,
  });
}

// 更新插件
export function updatePluginCenter(data: PluginCenterForm) {
  return request({
    url: `/plugin/center/update`,
    method: 'put',
    data,
  });
}

// 上传插件图标或详情附件，保持与原 admin 上传接口一致的响应结构。
export function uploadFile(
  data: FormData,
  options: Omit<AxiosRequestConfig, 'url' | 'data' | 'method'> = {},
) {
  return request({
    url: `/file/upload`,
    headers: {
      'Content-Type': 'application/form-data',
    },
    method: 'post',
    data,
    ...options,
  });
}


// 更新插件函数
export function pluginFunctionUpdate(data: PluginFunctionUpdateType) {
  return request({
    url: `/plugin/function/update`,
    method: 'put',
    data,
  });
}

// 根据插件 ID 查询函数列表，接口按后端函数和前端扩展分组返回。
export function getPluginTriggerList(params: PlatformFunctionListType) {
  return request({
    url: `/plugin/function/list`,
    method: 'get',
    params,
  });
}

export type PluginAgentInstallRequestOptions = Omit<
  AxiosRequestConfig,
  'url' | 'data' | 'method'
>;

