import type { FormItem, FormJsonValue } from '../schema/types';

/**
 * 运行时值契约（目标保存协议）：桌面、移动与后续原生容器共用同一 JSON 可序列化结构，
 * 值按 `widget.widgetName` 取键；字段类型决定值的形态（字段字典 §4）。
 */
export type FormValue = FormJsonValue;

/** 值写入来源：仅用户输入计入脏状态，初始化与规则回写不污染草稿语义。 */
export type FormValueSource = 'user' | 'init';

/** 表单会话生命周期；请求进行态单独由 activeOperation 表达。 */
export type FormRuntimeLifecycle = 'initializing' | 'ready' | 'submitted';

/** 内置持久化操作互斥执行，业务自定义动作的状态由宿主持有。 */
export type FormRuntimeOperation = 'submit' | 'save-draft';

/** 单字段运行时状态；FieldHost 只按自身 widgetName 订阅，避免整表重渲染。 */
export interface FieldRuntimeState {
  /**
   * 有效可见性（v6 设计方案 §4.2）：静态 visible ∧ 权限可见 ∧ 显隐规则命中。
   * 同时决定渲染与提交信封——v6 起服务端按同一口径复核 visible，权限隐藏
   * 字段不再携带「快照可见 + 空 data」的旧信封，改由不可见字段赋值策略决议。
   */
  visible: boolean;
  disabled: boolean;
  readonly: boolean;
  touched: boolean;
  validating: boolean;
  errors: readonly string[];
}

/**
 * 字段权限矩阵输入（bootstrap permissions 按模式投影的单矩阵）：
 * 未提供（预览/草稿回放）视为全量放行；提供后缺失键按 deny-by-default。
 */
export interface FormRuntimeFieldPermission {
  visible: boolean;
  editable: boolean;
}

/** 表单级问题：字段错误按 widgetName 关联，非字段错误（提交失败等）展示在操作区摘要。 */
export interface FormIssue {
  fieldKey?: string;
  message: string;
  source: 'local' | 'server';
}

/** 运行时唯一状态源；组件经注入的只读视图消费，写入只能走 Store action。 */
export interface FormRuntimeState {
  values: Record<string, FormValue>;
  fieldStates: Record<string, FieldRuntimeState>;
  lifecycle: FormRuntimeLifecycle;
  activeOperation: FormRuntimeOperation | null;
  dirtyKeys: Set<string>;
  issues: FormIssue[];
}

/**
 * 提交载荷（后端契约 §2.2）：值按 widgetName 取键，publishedVersion + schemaRevision
 * 双口令指向服务端按其校验的不可变发布快照。
 */
export interface FormSubmitPayload {
  formId: string;
  publishedVersion: number;
  schemaRevision: string;
  /**
   * 字段提交快照：每个字段同时携带运行时最终可见状态；空值省略 data，
   * 避免 null、缺省和空字符串混用。布局控件不进入该映射。
   */
  values: Record<string, FormSubmittedFieldValue>;
}

/** 单字段提交值：data 为字段类型决定的 JSON 值，visible 为规则执行后的最终状态。 */
export interface FormSubmittedFieldValue {
  data?: FormValue;
  visible: boolean;
}

/** 服务端提交结果；字段级错误按 widgetName 回填，非字段错误经 message 展示在操作区摘要。 */
export interface FormSubmitResult {
  accepted: boolean;
  fieldErrors?: Record<string, string[]>;
  message?: string;
}

/** 草稿载荷；本地草稿的隔离与过期策略由宿主 adapter 负责。 */
export interface FormDraftPayload {
  formId: string;
  publishedVersion: number;
  schemaRevision: string;
  values: Record<string, FormValue>;
}

/** 字段组件的统一值契约；错误展示由 FormFieldHost 承担，组件只消费错误态。 */
export interface RuntimeFieldProps {
  item: FormItem;
  modelValue: FormValue;
  disabled: boolean;
  readonly: boolean;
  errors: readonly string[];
}

/** 字段组件事件契约：值更新走 update:modelValue，失焦触发定向校验。 */
export interface RuntimeFieldEmits {
  'update:modelValue': [value: FormValue];
  blur: [];
}
