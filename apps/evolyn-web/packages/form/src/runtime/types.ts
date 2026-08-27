import type { FormItem, FormJsonValue } from '../schema/types';

/**
 * 运行时值契约（目标保存协议）：桌面、移动与后续原生容器共用同一 JSON 可序列化结构，
 * 值按 `widget.widgetName` 取键；字段类型决定值的形态（字段字典 §4）。
 */
export type FormValue = FormJsonValue;

/** 值写入来源：仅用户输入计入脏状态，初始化与规则回写不污染草稿语义。 */
export type FormValueSource = 'user' | 'init';

/** 表单会话阶段；submitted 后由宿主决定跳转、重置或继续填写。 */
export type FormRuntimePhase = 'initializing' | 'ready' | 'submitting' | 'submitted' | 'failed';

/** 单字段运行时状态；FieldHost 只按自身 widgetName 订阅，避免整表重渲染。 */
export interface FieldRuntimeState {
  visible: boolean;
  disabled: boolean;
  readonly: boolean;
  touched: boolean;
  validating: boolean;
  errors: readonly string[];
}

/** 表单级问题：字段错误按 widgetName 关联，非字段错误（提交失败等）展示在提交栏。 */
export interface FormIssue {
  fieldKey?: string;
  message: string;
  source: 'local' | 'server';
}

/** 运行时唯一状态源；组件经注入的只读视图消费，写入只能走 Store action。 */
export interface FormRuntimeState {
  values: Record<string, FormValue>;
  fieldStates: Record<string, FieldRuntimeState>;
  formState: FormRuntimePhase;
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
  values: Record<string, FormValue>;
}

/** 服务端提交结果；字段级错误按 widgetName 回填，非字段错误经 message 展示在提交栏。 */
export interface FormSubmitResult {
  accepted: boolean;
  fieldErrors?: Record<string, string[]>;
  message?: string;
}

/** 草稿载荷；本地草稿的隔离与过期策略由宿主 adapter 负责。 */
export interface FormDraftPayload {
  formId: string;
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
