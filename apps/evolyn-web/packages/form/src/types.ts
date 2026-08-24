export type PluginDesignViewMode = 'form' | 'code';
export type PluginDesignMenuKey = 'auth' | 'common' | 'code' | 'request' | 'response';
export type PluginDesignFunctionType = 'ai' | 'frontend' | 'backend';
// 前端扩展与后端函数分别维护排序号，AI 函数沿用后端函数分组。
export type PluginDesignFunctionGroupType = 'frontend' | 'backend';

export enum PluginRuntime {
  NodeJS = 'nodejs',
  Python3 = 'python3',
}

export interface PluginDesignFunctionCreatePayload {
  functionType: PluginDesignFunctionType;
  runtime: PluginRuntime;
}

export interface PluginDesignFunctionUpdatePayload {
  id: string;
  name: string;
  /** 函数描述，与后端 functionDescription 字段保持一致。 */
  functionDescription: string;
  runtime: PluginRuntime;
}

export interface PluginDesignOption {
  label: string;
  value: string;
}

export interface PluginDesignMenuItem {
  label: string;
  key: PluginDesignMenuKey;
  icon: unknown;
}

/** 表单设计器素材面板中的一个字段组件。 */
export interface FormDesignPaletteItem {
  label: string;
  /** 新增字段提交给函数接口的组件标识。 */
  widgetName: string;
  /** 新增字段提交给函数接口的数据类型。 */
  dataType: string;
  icon: unknown;
}

/** 点击或拖拽新增字段时透传的组件元数据。 */
export interface FormDesignFieldSource {
  widgetName: string;
  dataType: string;
}

/** 拖拽新增字段时携带插入位置的组件元数据。 */
export interface FormDesignDragField extends FormDesignFieldSource {
  index: number;
  /** Schema 适配层传递的字段类型，仅用于将拖放项还原为持久化字段。 */
  formFieldType?: string;
}

export interface FormDesignSelectorDefaultValue {
  users?: string[];
  departs?: string[];
  [key: string]: unknown;
}

/** 智能助手人员/部门下拉框的通用选项结构。 */
export interface FormDesignSelectorOption {
  label: string;
  value: string | number;
  [key: string]: unknown;
}

/** 根据插件字段类型获取智能助手人员/部门下拉选项。 */
export type FormDesignSelectorOptionsResolver = (
  field: FormDesignField | FormDesignTemplateField,
) => FormDesignSelectorOption[];

export type FormDesignFieldDefaultValue =
  | string
  | number
  | boolean
  | FormDesignSelectorDefaultValue;

/** 表单设计器内部字段模型，是编辑画布与后续持久化适配的唯一字段结构。 */
export interface FormDesignField {
  id: string | null;
  fieldKey: string;
  fieldLabel: string;
  /** 低代码表单组件标识。 */
  widgetName: string;
  /** 低代码表单字段数据类型。 */
  dataType: string;
  placeholder?: string;
  isRequired?: boolean;
  defaultValue?: FormDesignFieldDefaultValue;
  options?: string[];
  fieldConf?: Record<string, unknown>;
}

export interface PluginDesignOAuthVectorRow {
  key: string;
  value: string;
}

export interface PluginDesignOAuthVariable {
  id: string;
  label: string;
  type: string;
}

/** 子表单字段及表单模板共用的字段结构。 */
export interface FormDesignTemplateField {
  [key: string]: unknown;
  id: string | null;
  fieldKey: string;
  fieldLabel: string;
  description?: string;
  widgetName: string;
  dataType: string;
  isHidden?: boolean;
  isEnabled?: boolean;
  isRequired?: boolean;
  fieldConf: Record<string, unknown>;
  defaultValue?: FormDesignFieldDefaultValue | PluginDesignOAuthVectorRow[];
}

export interface FormDesignTemplateSection {
  [key: string]: unknown;
  fields: FormDesignTemplateField[];
  enabled?: boolean;
}

/**
 * 旧插件设计器仍在迁移中，保留别名以维持未迁移模块可编译。
 * 新表单代码必须使用 FormDesign* 名称。
 */
export type PluginDesignPaletteItem = FormDesignPaletteItem;
export type PluginDesignFieldSource = FormDesignFieldSource;
export type PluginDesignDragField = FormDesignDragField;
export type PluginDesignSelectorDefaultValue = FormDesignSelectorDefaultValue;
export type PluginDesignSelectorOption = FormDesignSelectorOption;
export type PluginDesignSelectorOptionsResolver = FormDesignSelectorOptionsResolver;
export type PluginDesignFieldDefaultValue = FormDesignFieldDefaultValue;
export type PluginDesignField = FormDesignField;
export type PluginDesignTemplateField = FormDesignTemplateField;
export type PluginDesignTemplateSection = FormDesignTemplateSection;

export interface PluginDesignAuthentication {
  type: string;
  conf_template: PluginDesignTemplateSection;
  credential_template?: PluginDesignTemplateSection;
  oauth2_request_template?: PluginDesignTemplateSection;
  refresh_token_request_template?: PluginDesignTemplateSection;
  app_auth_template?: PluginDesignTemplateSection;
}

/** 函数返回参数字段配置，结构与后端 responseParameter.fields 保持一致。 */
export interface PluginDesignResponseField {
  /** 已保存字段使用后端 ID，新增字段保存前为 null。 */
  id: string | null;
  fieldKey: string;
  fieldLabel: string;
  widgetName: string;
  dataType: string;
  fieldConf: {
    fields?: PluginDesignResponseField[];
    [key: string]: unknown;
  };
}

export interface PluginCodeDiagnostic {
  line: number;
  column: number;
  endLine?: number;
  endColumn?: number;
  message: string;
  severity?: 'error' | 'warning';
}

export interface PluginDesignFunction {
  id: string;
  /** 后端生成的函数唯一标识，插件调试时不能使用函数 ID 或源码代替。 */
  functionKey?: string;
  name: string;
  /** 函数说明，更新函数时与名称等完整信息一起提交。 */
  functionDescription: string;
  functionType?: PluginDesignFunctionType;
  /** 当前函数在所属前端/后端分组内从 0 开始的排序号。 */
  seq: number;
  runtime: PluginRuntime;
  viewMode: PluginDesignViewMode;
  fields: PluginDesignField[];
  responseParams: PluginDesignResponseField[];
  code: string;
}

/** 侧栏完成分组内拖拽后提交给设计器的完整排序结果。 */
export interface PluginDesignFunctionSortPayload {
  functionGroup: PluginDesignFunctionGroupType;
  functions: PluginDesignFunction[];
  sortedFunctions: PluginDesignFunction[];
}

/** 排序接口完成后通知设计器成功持久化或重新加载列表。 */
export type PluginDesignFunctionSortComplete = (success: boolean) => void;

export interface PluginDesignConfig {
  activeFunctionId: string;
  savedAt?: string;
  functions: PluginDesignFunction[];
}

// activeMenu 仅用于前端选择本次局部保存模块，不写入 pluginDesign 函数数据。
export interface PluginDesignSaveValue {
  activeMenu: PluginDesignMenuKey;
  authentication: PluginDesignAuthentication;
  globalFields: PluginDesignField[];
  pluginDesign: PluginDesignConfig;
}

// 父组件完成异步保存后，通过回调通知设计器更新保存时间或打开调试抽屉。
export type PluginDesignSaveComplete = (success: boolean) => void;

// 函数名称更新完成后返回接口确认的完整函数，确保字段 ID 等后端回显同步到设计器。
export type PluginDesignFunctionUpdateComplete = (
  success: boolean,
  functionData?: PluginDesignFunction,
) => void;
