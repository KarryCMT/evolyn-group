import { reactive } from 'vue';
import { cloneFormSchema } from '../../schema/clone';
import {
  emptyWidgetValue,
  isLayoutWidgetType,
  normalizeWidgetValue,
  validateWidgetValue,
} from '../../schema/codec';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import type { FormRuntimeAdapter } from '../adapters/types';
import type {
  FieldRuntimeState,
  FormIssue,
  FormRuntimeState,
  FormSubmitPayload,
  FormValue,
  FormValueSource,
} from '../types';

/**
 * 运行时会话工厂：每个 FormRenderer 创建独立 session，不使用全局 store 承载填写值。
 * 所有修改经 setValue/markTouched 等 action 进入，组件只消费只读状态；值表与字段状态
 * 按 widgetName 细粒度响应式追踪，输入一个字段仅重渲染该字段（及 R2 起的规则依赖项）。
 * 静态属性按保存值执行（方案 §3.5）：visible=false 不校验不收集，enable=false 禁用但值仍提交。
 */
export interface FormRuntimeOptions {
  /** 已发布 Schema（目标协议文档），为运行时唯一输入；工厂内部深拷贝锁定会话。 */
  schema: FormSchemaDocument;
  formId?: string;
  publishedVersion?: number;
  schemaRevision?: string;
  /** 已保存记录值，初始化优先级最高。 */
  initialValues?: Record<string, FormValue>;
  /** 记录上下文默认值，例如「当前成员作为申请人」（P5 前不执行 Schema 静态 defaultValue）。 */
  contextDefaults?: Record<string, FormValue>;
  /** 业务能力注入边界；缺省时提交仅生成载荷交由页面处理。 */
  adapter?: FormRuntimeAdapter;
}

export type FormSubmitOutcome =
  | { ok: true; payload: FormSubmitPayload; submitted: boolean }
  | {
      ok: false;
      reason: 'busy' | 'invalid' | 'server' | 'error' | 'cancelled';
      payload?: FormSubmitPayload;
      error?: unknown;
    };

export interface FormRuntime {
  readonly schema: FormSchemaDocument;
  readonly state: FormRuntimeState;
  /** 字段项索引（键=widgetName）；重复键已被 Schema 校验拦截。 */
  readonly itemMap: ReadonlyMap<string, FormItem>;
  setValue(key: string, value: FormValue, source?: FormValueSource): void;
  markTouched(key: string): void;
  validateField(key: string): readonly string[];
  validateVisibleFields(): boolean;
  /** 服务端字段错误按 widgetName 回填（提交失败处理）。 */
  applyServerFieldErrors(fieldErrors: Record<string, string[]>): void;
  addServerIssue(message: string): void;
  clearServerIssues(): void;
  buildSubmitPayload(): FormSubmitPayload;
  submit(signal?: AbortSignal): Promise<FormSubmitOutcome>;
  reset(): void;
  isDirty(): boolean;
}

export function createFormRuntime(options: FormRuntimeOptions): FormRuntime {
  // 深拷贝锁定会话内 Schema：已发布版本不可覆盖，外部对象再变更不影响本会话。
  const schema = cloneFormSchema(options.schema);
  const itemMap = new Map(schema.content.items.map((item) => [item.widget.widgetName, item]));
  // 布局项（分割线/按钮）无值无状态，不进入值表与校验序列。
  const dataItems = schema.content.items.filter((item) => !isLayoutWidgetType(item.widget.type));

  const state = reactive({
    values: {},
    fieldStates: {},
    formState: 'initializing',
    dirtyKeys: new Set<string>(),
    issues: [],
  }) as FormRuntimeState;

  initializeValues();
  state.formState = 'ready';

  function initializeValues(): void {
    state.values = {};
    state.fieldStates = {};
    state.dirtyKeys = new Set<string>();
    state.issues = [];
    for (const item of dataItems) {
      state.values[item.widget.widgetName] = pickInitialValue(item);
      state.fieldStates[item.widget.widgetName] = createFieldState(item);
    }
  }

  /** 初始化优先级：已保存值 → 记录上下文默认值 → 类型化空值（Schema 静态 defaultValue 随 P5 规则执行）。 */
  function pickInitialValue(item: FormItem): FormValue {
    const key = item.widget.widgetName;
    const saved = readOwn(options.initialValues, key);
    if (saved !== undefined) return normalizeWidgetValue(item.widget, saved);
    const contextual = readOwn(options.contextDefaults, key);
    if (contextual !== undefined) return normalizeWidgetValue(item.widget, contextual);
    return emptyWidgetValue(item.widget.type);
  }

  function setValue(key: string, value: FormValue, source: FormValueSource = 'user'): void {
    const item = itemMap.get(key);
    const fieldState = state.fieldStates[key];
    if (!item || !fieldState) return;

    const next = normalizeWidgetValue(item.widget, value);
    if (isSameFieldValue(state.values[key], next)) return;
    state.values[key] = next;

    if (source === 'user') {
      state.dirtyKeys.add(key);
      // 已出错的字段在输入时即时重校验，便于立刻清除错误；未触碰字段留待失焦校验。
      if (fieldState.errors.length > 0) validateField(key);
    }
    // R2 扩展点：规则回写在此沿依赖图触发下游显隐/必填/计算。
  }

  function markTouched(key: string): void {
    const fieldState = state.fieldStates[key];
    if (!fieldState || fieldState.touched) return;
    fieldState.touched = true;
    validateField(key);
  }

  function validateField(key: string): readonly string[] {
    const item = itemMap.get(key);
    const fieldState = state.fieldStates[key];
    if (!item || !fieldState) return [];
    // 隐藏字段不校验（R2 显隐联动后由规则保证可见性）。
    if (!fieldState.visible) {
      fieldState.errors = [];
      return [];
    }
    fieldState.errors = validateWidgetValue(item, state.values[key]);
    return fieldState.errors;
  }

  /** 提交前校验：校验全部可见且可提交字段，错误聚合进 issues 摘要。 */
  function validateVisibleFields(): boolean {
    const fieldIssues: FormIssue[] = [];
    let valid = true;
    for (const item of dataItems) {
      const errors = validateField(item.widget.widgetName);
      if (errors.length > 0) {
        valid = false;
        fieldIssues.push(
          ...errors.map<FormIssue>((message) => ({
            fieldKey: item.widget.widgetName,
            message,
            source: 'local',
          })),
        );
      }
    }
    // 仅替换本地字段问题；服务端非字段错误（版本冲突提示等）保留在提交栏。
    const serverIssues = state.issues.filter(
      (issue) => issue.source === 'server' && !issue.fieldKey,
    );
    state.issues = [...fieldIssues, ...serverIssues];
    return valid;
  }

  function applyServerFieldErrors(fieldErrors: Record<string, string[]>): void {
    const serverIssues: FormIssue[] = [];
    for (const [key, messages] of Object.entries(fieldErrors)) {
      const fieldState = state.fieldStates[key];
      if (!fieldState || !Array.isArray(messages)) continue;
      const errors = messages.filter((message): message is string => typeof message === 'string');
      fieldState.errors = errors;
      for (const message of errors) serverIssues.push({ fieldKey: key, message, source: 'server' });
    }
    const kept = state.issues.filter((issue) => issue.source === 'server' && !issue.fieldKey);
    state.issues = [...serverIssues, ...kept];
  }

  function addServerIssue(message: string): void {
    state.issues.push({ message, source: 'server' });
  }

  function clearServerIssues(): void {
    state.issues = state.issues.filter((issue) => issue.source !== 'server');
  }

  /** 组装稳定提交载荷：仅收集可见字段（隐藏字段不渲染不收集），深拷贝值快照。 */
  function buildSubmitPayload(): FormSubmitPayload {
    const values: Record<string, FormValue> = {};
    for (const item of dataItems) {
      const key = item.widget.widgetName;
      const visible = state.fieldStates[key]?.visible ?? item.widget.visible;
      if (!visible) continue;
      const value = state.values[key];
      values[key] =
        value === undefined || value === null
          ? null
          : (JSON.parse(JSON.stringify(value)) as FormValue);
    }
    return {
      formId: options.formId ?? '',
      publishedVersion: options.publishedVersion ?? 0,
      schemaRevision: options.schemaRevision ?? '',
      values,
    };
  }

  async function submit(signal?: AbortSignal): Promise<FormSubmitOutcome> {
    if (state.formState === 'submitting' || state.formState === 'submitted') {
      return { ok: false, reason: 'busy' };
    }
    if (!validateVisibleFields()) return { ok: false, reason: 'invalid' };

    const payload = buildSubmitPayload();
    const submitter = options.adapter?.submit;
    if (!submitter) return { ok: true, payload, submitted: false };

    state.formState = 'submitting';
    try {
      const result = await submitter(payload, signal ?? new AbortController().signal);
      if (!result.accepted) {
        applyServerFieldErrors(result.fieldErrors ?? {});
        if (result.message) addServerIssue(result.message);
        state.formState = 'failed';
        return { ok: false, reason: 'server', payload };
      }
      clearServerIssues();
      state.formState = 'submitted';
      return { ok: true, payload, submitted: true };
    } catch (error) {
      if (isAbortError(error)) {
        // 请求取消不是业务错误：恢复可提交状态，不展示错误。
        state.formState = 'ready';
        return { ok: false, reason: 'cancelled', payload };
      }
      addServerIssue('提交失败，请稍后重试');
      state.formState = 'failed';
      return { ok: false, reason: 'error', payload, error };
    }
  }

  function reset(): void {
    initializeValues();
    state.formState = 'ready';
  }

  function isDirty(): boolean {
    return state.dirtyKeys.size > 0;
  }

  return {
    schema,
    state,
    itemMap,
    setValue,
    markTouched,
    validateField,
    validateVisibleFields,
    applyServerFieldErrors,
    addServerIssue,
    clearServerIssues,
    buildSubmitPayload,
    submit,
    reset,
    isDirty,
  };
}

/** 字段静态状态按保存值执行：visible 决定收集，enable 取反映射禁用。 */
function createFieldState(item: FormItem): FieldRuntimeState {
  return {
    visible: item.widget.visible,
    disabled: !item.widget.enable,
    readonly: false,
    touched: false,
    validating: false,
    errors: [],
  };
}

/** 值相等判断：标量用同值比较，数组做浅比较；对象仅比较引用，避免热路径深比较。 */
function isSameFieldValue(current: FormValue | undefined, next: FormValue): boolean {
  if (current === next) return true;
  if (Array.isArray(current) && Array.isArray(next)) {
    return current.length === next.length && current.every((item, index) => item === next[index]);
  }
  return false;
}

function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    (error as { name?: unknown }).name === 'AbortError'
  );
}

function readOwn(
  source: Record<string, FormValue> | undefined,
  key: string,
): FormValue | undefined {
  if (!source || !Object.prototype.hasOwnProperty.call(source, key)) return undefined;
  return source[key];
}
