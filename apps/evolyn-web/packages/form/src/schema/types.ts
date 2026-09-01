/**
 * 目标保存协议 TypeScript 类型（P1，ADR-010 / docs/低代码平台/表单设计器/目标协议字段字典.md）。
 *
 * 根结构 `{ content: { type: "form", layout: "normal", items: [...], layout_fields: [...], field_layout: [...] } }`
 * 是设计器、后端草稿、发布版本与
 * 最终渲染器共用的唯一事实结构；文档内不携带版本号，协议版本由持久层外部承载
 * （forms.protocol_version 列 + FORM_PROTOCOL_VERSION 常量）。
 * 类型只表达「合法文档」的形状；JSON 的未知键/非法值由 validate.ts 按 JSON Path 拒绝。
 */

/** 协议版本常量；递增时必须同步版本迁移器（migrate.ts）与字段字典。 */
export const FORM_PROTOCOL_VERSION = 4 as const;
export type FormProtocolVersion = typeof FORM_PROTOCOL_VERSION;

/** Schema 可以安全持久化的 JSON 值；不允许组件、函数或循环引用进入文档。 */
export type FormJsonValue =
  | string
  | number
  | boolean
  | null
  | FormJsonValue[]
  | { [key: string]: FormJsonValue };

/** 27 种控件判别键（字段字典 §3）；新增类型必须先修订字典再动代码。 */
export type FormWidgetType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'datetime'
  | 'radiogroup'
  | 'checkboxgroup'
  | 'combo'
  | 'combocheck'
  | 'separator'
  | 'user'
  | 'usergroup'
  | 'dept'
  | 'deptgroup'
  | 'image'
  | 'upload'
  | 'address'
  | 'location'
  | 'signature'
  | 'phone'
  | 'subform'
  | 'linkquery'
  | 'linkfield'
  | 'lookup'
  | 'aggregation'
  | 'sn'
  | 'richtext'
  | 'button';

/**
 * 已开放运行时的发布白名单（字段字典 §6）：基础字段及成员单选/多选。
 * 前后端各维护一份并保持一致，新增类型时必须同时补齐运行组件与值校验。
 */
export const PUBLISHABLE_WIDGET_TYPES: readonly FormWidgetType[] = [
  'text',
  'textarea',
  'number',
  'datetime',
  'radiogroup',
  'checkboxgroup',
  'combo',
  'combocheck',
  'separator',
  'user',
  'usergroup',
];

/** 选项结构：label/value 均为 1–100 字符，组内 value 唯一。 */
export interface FormWidgetOption {
  label: string;
  value: string;
}

/** 附件/图片值元素（P3 执行；服务端文件 ID 引用，未完成上传的本地条目不得提交）。 */
export interface FormFileValue {
  id: string;
  name: string;
  size: number;
  mimeType: string;
  url?: string;
}

/** 控件层公共属性（字段字典 §2）：全部必填键，布尔不允许 null。 */
export interface FormWidgetCommon {
  /** 控件判别键。 */
  type: FormWidgetType;
  /** 稳定字段键：记录值、规则与错误回填的取值键，全表单唯一（子表单按作用域）。 */
  widgetName: string;
  /** false 时禁用输入，值仍参与提交。 */
  enable: boolean;
  /** false 时不渲染、不校验、不收集值。 */
  visible: boolean;
  /** false 时必填（allowBlank 反向表达旧 required）。 */
  allowBlank: boolean;
}

export interface TextWidget extends FormWidgetCommon {
  type: 'text';
  placeholder?: string;
  minLength?: number | null;
  maxLength?: number | null;
  format?: '' | 'email';
  defaultValue?: string | null;
}

export interface TextAreaWidget extends FormWidgetCommon {
  type: 'textarea';
  placeholder?: string;
  minLength?: number | null;
  maxLength?: number | null;
  autoHeight?: boolean;
  defaultValue?: string | null;
}

export interface NumberWidget extends FormWidgetCommon {
  type: 'number';
  placeholder?: string;
  min?: number | null;
  max?: number | null;
  precision?: number | null;
  defaultValue?: number | null;
}

export type DateTimeFormat = 'date' | 'datetime' | 'month' | 'time';

export interface DateTimeWidget extends FormWidgetCommon {
  type: 'datetime';
  format?: DateTimeFormat;
  placeholder?: string;
  defaultValue?: string | null;
}

export interface RadioGroupWidget extends FormWidgetCommon {
  type: 'radiogroup';
  options: FormWidgetOption[];
  layout?: 'vertical' | 'horizontal';
  defaultValue?: string | null;
}

export interface CheckboxGroupWidget extends FormWidgetCommon {
  type: 'checkboxgroup';
  options: FormWidgetOption[];
  layout?: 'vertical' | 'horizontal';
  defaultValue?: string[] | null;
}

export interface ComboWidget extends FormWidgetCommon {
  type: 'combo';
  options: FormWidgetOption[];
  placeholder?: string;
  filterable?: boolean;
  defaultValue?: string | null;
}

export interface ComboCheckWidget extends FormWidgetCommon {
  type: 'combocheck';
  options: FormWidgetOption[];
  placeholder?: string;
  defaultValue?: string[] | null;
}

export interface SeparatorWidget extends FormWidgetCommon {
  type: 'separator';
  /** Element Plus Divider 默认插槽文案，独立于字段通用标题。 */
  content?: string;
  /** 对齐 Element Plus direction 属性；垂直分隔线在画布中仍作为独立布局项显示。 */
  direction?: 'horizontal' | 'vertical';
  /** 对齐 Element Plus border-style，接受 CSS 的全部标准边框样式。 */
  borderStyle?:
    | 'none'
    | 'hidden'
    | 'dotted'
    | 'dashed'
    | 'solid'
    | 'double'
    | 'groove'
    | 'ridge'
    | 'inset'
    | 'outset';
  /** 对齐 Element Plus content-position；仅横向分割线展示文案位置。 */
  contentPosition?: 'left' | 'center' | 'right';
  /** @deprecated v4 历史字段，读取时回退为 borderStyle。 */
  style?: 'solid' | 'dashed';
}

export interface UserWidget extends FormWidgetCommon {
  type: 'user';
  scope?: 'tenant' | 'department';
  departments?: string[];
  defaultValue?: string | null;
}

export interface UserGroupWidget extends FormWidgetCommon {
  type: 'usergroup';
  scope?: 'tenant' | 'department';
  departments?: string[];
  defaultValue?: string[] | null;
}

export interface DeptWidget extends FormWidgetCommon {
  type: 'dept';
  includeChildren?: boolean;
  defaultValue?: string | null;
}

export interface DeptGroupWidget extends FormWidgetCommon {
  type: 'deptgroup';
  includeChildren?: boolean;
  defaultValue?: string[] | null;
}

export interface ImageWidget extends FormWidgetCommon {
  type: 'image';
  maxCount?: number;
  maxSizeMB?: number;
  accept?: string[];
}

export interface UploadWidget extends FormWidgetCommon {
  type: 'upload';
  maxCount?: number;
  maxSizeMB?: number;
  accept?: string[];
}

export type AddressLevel = 'province' | 'city' | 'district' | 'detail';

export interface AddressWidget extends FormWidgetCommon {
  type: 'address';
  level?: AddressLevel;
  detailPlaceholder?: string;
}

export interface LocationWidget extends FormWidgetCommon {
  type: 'location';
  radius?: number | null;
}

export interface SignatureWidget extends FormWidgetCommon {
  type: 'signature';
}

export interface PhoneWidget extends FormWidgetCommon {
  type: 'phone';
  areaCode?: string;
  verification?: boolean;
}

/**
 * 子表单子项允许的控件集（字段字典 §3.3）。
 * 标签页不属于字段类型；分割线、富文本与子表单本身没有行值语义，因此禁止放入。
 */
export const SUBFORM_ALLOWED_WIDGET_TYPES: readonly FormWidgetType[] = [
  'text',
  'textarea',
  'number',
  'datetime',
  'radiogroup',
  'checkboxgroup',
  'combo',
  'combocheck',
  'user',
  'usergroup',
  'dept',
  'deptgroup',
  'image',
  'upload',
  'address',
  'location',
  'signature',
  'phone',
  'linkquery',
  'linkfield',
  'lookup',
  'aggregation',
  'sn',
  'button',
];

/** 子表单冻结列配置；关闭时 limit 仍保留，便于再次开启恢复上次设置。 */
export interface SubformStickyColumnConfig {
  enable: boolean;
  limit: number;
}

export interface SubformWidget extends FormWidgetCommon {
  type: 'subform';
  /** 嵌套字段项，结构与顶层 items 相同；子项 widgetName 在本子表单作用域内唯一。 */
  items: FormItem[];
  /** 行操作权限；可编辑关闭时设计器同时禁用四个细分动作。 */
  subformCreate: boolean;
  subformInsert: boolean;
  subformEdit: boolean;
  subformDelete: boolean;
  /** 快速填报仅在新增与编辑已有记录均允许时生效。 */
  quickFill: boolean;
  pcStickyColumn: SubformStickyColumnConfig;
  mobileStickyColumn: SubformStickyColumnConfig;
  /** 移动端纵向卡片或横向表格。 */
  mobileViewStyle: 'vertical' | 'horizontal';
  /** 纵向卡片收起时参与摘要的前 N 个字段。 */
  mobileSummaryFieldCount: number;
  minRowCount?: number | null;
  maxRowCount?: number | null;
}

export type LinkFilterOp = 'eq' | 'ne' | 'gt' | 'lt' | 'ge' | 'le' | 'contains';

export interface LinkFilter {
  field: string;
  op: LinkFilterOp;
  value: FormJsonValue;
}

export interface LinkSort {
  field: string;
  order: 'asc' | 'desc';
}

export interface LinkQueryWidget extends FormWidgetCommon {
  type: 'linkquery';
  targetForm?: string | null;
  multiple?: boolean;
  filters?: LinkFilter[];
  sorts?: LinkSort[];
}

export interface LinkFieldWidget extends FormWidgetCommon {
  type: 'linkfield';
  targetForm?: string | null;
  displayFields?: string[];
}

export interface LookupMapping {
  source: string;
  target: string;
}

export interface LookupWidget extends FormWidgetCommon {
  type: 'lookup';
  targetForm?: string | null;
  mappings?: LookupMapping[];
}

export type AggregationOp = 'sum' | 'avg' | 'count' | 'min' | 'max';

export interface AggregationExpression {
  op: AggregationOp;
  /** 源字段：本表单内 subform / linkquery 的 widgetName。 */
  source: string;
  /** 源内数值字段键；op=count 时省略。 */
  field?: string;
}

export interface AggregationWidget extends FormWidgetCommon {
  type: 'aggregation';
  expression?: AggregationExpression | null;
  precision?: number | null;
  displayMode?: 'plain' | 'percent';
}

export interface SnRule {
  prefix?: string;
  dateFmt?: 'none' | 'yyyyMM' | 'yyyyMMdd';
  seqLength?: number;
  resetCycle?: 'none' | 'daily' | 'monthly' | 'yearly';
}

export interface SnWidget extends FormWidgetCommon {
  type: 'sn';
  rule?: SnRule;
}

export interface RichTextWidget extends FormWidgetCommon {
  type: 'richtext';
  toolbar?: string[];
}

export interface ButtonAction {
  type: 'none' | 'submit';
}

export interface ButtonWidget extends FormWidgetCommon {
  type: 'button';
  text?: string;
  action?: ButtonAction;
}

/** 控件判别联合：以 type 为判别键的 27 种控件。 */
export type FormItemWidget =
  | TextWidget
  | TextAreaWidget
  | NumberWidget
  | DateTimeWidget
  | RadioGroupWidget
  | CheckboxGroupWidget
  | ComboWidget
  | ComboCheckWidget
  | SeparatorWidget
  | UserWidget
  | UserGroupWidget
  | DeptWidget
  | DeptGroupWidget
  | ImageWidget
  | UploadWidget
  | AddressWidget
  | LocationWidget
  | SignatureWidget
  | PhoneWidget
  | SubformWidget
  | LinkQueryWidget
  | LinkFieldWidget
  | LookupWidget
  | AggregationWidget
  | SnWidget
  | RichTextWidget
  | ButtonWidget;

/** 字段项公共展示属性（字段字典 §2）；全部必填键。 */
export interface FormItem<W extends FormItemWidget = FormItemWidget> {
  widget: W;
  /** 字段显示名称；除 separator 允许空串外 trim 后 1–64 字符。 */
  label: string;
  /** 字段说明，0–500 字符，空串即「无」。 */
  description: string;
  /** 隐藏标签（仍渲染控件与说明）。 */
  labelHidden: boolean;
  /** 桌面 12 栅格占列数 1–12；移动端恒单列。 */
  lineWidth: number;
}

export interface FormContent {
  type: 'form';
  /** 表单默认列布局；切换时同步重置所有普通字段的 lineWidth。 */
  layout: FormLayoutMode;
  items: FormItem[];
  /** 表单级布局定义池；v2 首期只开放 multitab。 */
  layout_fields: FormMultitabLayout[];
  /** 顶层字段与布局节点的唯一排列序列。 */
  field_layout: string[];
}

/** 表单级默认列布局；字段仍可通过 lineWidth 单独覆盖实际宽度。 */
export type FormLayoutMode = 'normal' | 'grid-2' | 'grid-3' | 'grid-4';

export type FormTabStyle = 'style1' | 'style2';

/** 标签页容器；field_layout 只引用 content.items 中的顶层 widgetName。 */
export interface FormTabLayout {
  name: string;
  title: string;
  type: 'tab';
  field_layout: string[];
}

/** v2 首期唯一布局类型；布局本身不产生表单值。 */
export interface FormMultitabLayout {
  name: string;
  type: 'multitab';
  tabStyle: FormTabStyle;
  container: FormTabLayout[];
}

/** 目标保存协议根结构：设计器草稿、发布快照与运行时输入的同一形状。 */
export interface FormSchemaDocument {
  content: FormContent;
}
