/** 当前包可以读取和写回的表单 Schema 版本。 */
export const FORM_SCHEMA_VERSION = 1 as const;
export type FormSchemaVersion = typeof FORM_SCHEMA_VERSION;

/** 普通表单与流程表单共用字段 Schema；流程图本身由后续工作流包承接。 */
export type FormKind = 'standard' | 'workflow';

/** 首批设计器暴露的稳定字段类型，字段类型值不直接使用界面文案。 */
export type FormFieldType =
  | 'single-line-text'
  | 'multi-line-text'
  | 'number'
  | 'date-time'
  | 'radio-group'
  | 'checkbox-group'
  | 'select'
  | 'multi-select'
  | 'member'
  | 'members'
  | 'department'
  | 'departments'
  | 'divider'
  | 'tabs'
  | 'image'
  | 'attachment'
  | 'address'
  | 'location'
  | 'sub-form'
  | 'query'
  | 'data-selector'
  | 'signature'
  | 'serial-number'
  | 'mobile'
  | 'text-recognition'
  | 'button'
  | 'formula'
  | 'rich-text'
  | 'related-data'
  | 'related-query'
  | 'related-form';

/** Schema 可以安全持久化的 JSON 值；不允许 Vue 组件、函数或循环引用进入文档。 */
export type FormJsonValue =
  | string
  | number
  | boolean
  | null
  | FormJsonValue[]
  | { [key: string]: FormJsonValue };

export interface FormField {
  /** 字段实例 ID，供设计器选择、拖拽和属性面板稳定引用。 */
  id: string;
  /** 字段业务键，供记录数据与公式等后续模块引用。 */
  key: string;
  type: FormFieldType;
  label: string;
  required: boolean;
  config: Record<string, FormJsonValue>;
}

/** 表单级展示、提交等设置；具体设置项随 Schema 版本演进。 */
export interface FormSettings {
  submitText?: string;
  [key: string]: FormJsonValue | undefined;
}

/** 后端持久化、设计器编辑和运行态渲染共用的 JSON 根结构。 */
export interface FormDocument {
  version: FormSchemaVersion;
  kind: FormKind;
  title: string;
  fields: FormField[];
  settings: FormSettings;
}

export type FormSchema = FormDocument;

/** 字段面板按业务认知分组，界面文案与稳定 type 分开维护。 */
export type FormFieldGroupKey = 'common' | 'advanced' | 'relation';

export interface FormFieldPreset {
  type: FormFieldType;
  title: string;
  group: FormFieldGroupKey;
  /** 默认字段标题，创建实例时允许应用层或编辑器覆盖。 */
  defaultLabel: string;
  defaultConfig?: Record<string, FormJsonValue>;
}

export interface FormFieldGroup {
  key: FormFieldGroupKey;
  title: string;
  fields: readonly FormFieldPreset[];
}
