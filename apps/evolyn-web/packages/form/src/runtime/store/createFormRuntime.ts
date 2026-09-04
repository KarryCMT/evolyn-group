import { reactive } from 'vue';
import { resolveFieldPermission } from '@evolyn.do/permission';
import { createRuntimeSessionState } from '@evolyn.do/runtime';
import { cloneFormSchema } from '../../schema/clone';
import {
  emptyWidgetValue,
  isLayoutWidgetType,
  normalizeWidgetValue,
  validateWidgetValue,
} from '../../schema/codec';
import {
  compileFieldShowRules,
  downstreamTargets,
  evaluateFieldShowRules,
  matchFieldShowRule,
  type CompiledFieldShowRules,
} from '../../schema/rules';
import {
  readInvisibleValuePolicy,
  resolveSubmitStrategy,
  type InvisibleValuePolicyView,
} from '../../schema/invisible-value-policy';
import type { FormItem, FormSchemaDocument, SubmitRule } from '../../schema/types';
import type { FormRuntimeAdapter } from '../adapters/types';
import type {
  FieldRuntimeState,
  FormDraftPayload,
  FormIssue,
  FormRuntimeFieldPermission,
  FormRuntimeLifecycle,
  FormRuntimeOperation,
  FormRuntimeState,
  FormSubmittedFieldValue,
  FormSubmitPayload,
  FormValue,
  FormValueSource,
} from '../types';

/**
 * 运行时会话工厂：每个 FormRenderer 创建独立 session，不使用全局 store 承载填写值。
 * 所有修改经 setValue/markTouched 等 action 进入，组件只消费只读状态；值表与字段状态
 * 按 widgetName 细粒度响应式追踪，输入一个字段仅重渲染该字段及其规则依赖项。
 * 静态属性按保存值执行（方案 §3.5）：visible=false 不校验且不进入提交载荷，
 * enable=false 禁用但已有值仍提交。
 * v5 显隐规则 + v6 信封语义：初始化与值变化时按编译产物计算有效可见性
 * effectiveVisible = 静态 visible ∧ 权限可见 ∧ 规则命中；该口径同时驱动渲染与
 * 提交信封——隐藏字段保留会话值、清除错误、不携带 data，落库值由服务端按
 * 不可见字段赋值策略（submitRule / widget_submit_rules）终审决议。
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
  /**
   * 当前登录成员 ID：显隐规则 includeCurrentMember 的比较集合注入源；
   * 未提供（匿名填写）时不加入任何值（设计方案 §3.2）。
   */
  currentMemberId?: string;
  /**
   * 字段权限矩阵（bootstrap permissions 按模式投影）：参与
   * effectiveVisible = 静态 ∧ 权限 ∧ 规则 的合成；未提供时全量放行。
   */
  fieldPermissions?: Record<string, FormRuntimeFieldPermission>;
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

export type FormDraftOutcome =
  | { ok: true; payload: FormDraftPayload }
  | {
      ok: false;
      reason: 'busy' | 'unavailable' | 'error' | 'cancelled';
      payload?: FormDraftPayload;
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
  /**
   * 字段的不可见赋值策略（v6 §5.2 客户端预演）：供预览调试与宿主展示
   * 「将保留原值 / 将清空」提示；权威决议在服务端。
   */
  submitStrategyOf(key: string): SubmitRule;
  /** 服务端字段错误按 widgetName 回填（提交失败处理）。 */
  applyServerFieldErrors(fieldErrors: Record<string, string[]>): void;
  addServerIssue(message: string): void;
  clearServerIssues(): void;
  buildSubmitPayload(): FormSubmitPayload;
  buildDraftPayload(): FormDraftPayload;
  submit(signal?: AbortSignal): Promise<FormSubmitOutcome>;
  saveDraft(signal?: AbortSignal): Promise<FormDraftOutcome>;
  reset(): void;
  isDirty(): boolean;
}

export function createFormRuntime(options: FormRuntimeOptions): FormRuntime {
  // 深拷贝锁定会话内 Schema：已发布版本不可覆盖，外部对象再变更不影响本会话。
  const schema = cloneFormSchema(options.schema);
  const itemMap = new Map(schema.content.items.map((item) => [item.widget.widgetName, item]));
  // 布局项（分割线/按钮）无值无状态，不进入值表与校验序列。
  const dataItems = schema.content.items.filter((item) => !isLayoutWidgetType(item.widget.type));
  // 显隐规则编译产物：无规则时整体短路，不产生任何求值开销（v5 设计方案 §6.1）。
  const compiledRules: CompiledFieldShowRules = compileFieldShowRules(schema.content);
  const ruleById = new Map(compiledRules.rules.map((rule) => [rule.id, rule]));
  // v6 不可见字段赋值策略解析（防御式：旧快照缺键回退默认「空值」）。
  const submitPolicy: InvisibleValuePolicyView = readInvisibleValuePolicy(schema.content);
  // 字段权限矩阵：未提供（预览/草稿回放）视为全量放行；提供后缺失键
  // deny-by-default，与后端 FieldsForNew 投影同口径（设计方案 §4.2）。
  const permissions = options.fieldPermissions;
  const permissionVisible = (key: string): boolean =>
    resolveFieldPermission(permissions, key, 'allow').visible;
  const permissionEditable = (key: string): boolean =>
    resolveFieldPermission(permissions, key, 'allow').editable;

  /** 字段静态状态按保存值执行：visible 决定收集，enable 取反映射禁用；
   * v6 有效可见性 = 静态 ∧ 权限 ∧ 规则（渲染与信封同口径）。 */
  function createFieldState(item: FormItem): FieldRuntimeState {
    const key = item.widget.widgetName;
    return {
      visible: item.widget.visible && permissionVisible(key),
      disabled: !item.widget.enable || !permissionEditable(key),
      readonly: false,
      touched: false,
      validating: false,
      errors: [],
    };
  }

  // Engine 仅创建框架无关会话；Form Runtime 作为 Vue 适配层决定响应式实现。
  const state = reactive(
    createRuntimeSessionState<
      FormValue,
      FieldRuntimeState,
      FormIssue,
      FormRuntimeLifecycle,
      FormRuntimeOperation
    >('initializing'),
  ) as FormRuntimeState;

  initializeValues();
  state.lifecycle = 'ready';

  function initializeValues(): void {
    state.values = {};
    state.fieldStates = {};
    state.dirtyKeys = new Set<string>();
    state.issues = [];
    for (const item of dataItems) {
      state.values[item.widget.widgetName] = pickInitialValue(item);
      state.fieldStates[item.widget.widgetName] = createFieldState(item);
    }
    applyFieldShowRules();
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
    // 显隐规则：仅重算变更字段的下游闭包（拓扑序），不做全量规则扫描。
    recomputeDownstreamVisibility(key);
  }

  // ---- 显隐规则引擎（v5 设计方案 §4.2/§6.1） ----

  /**
   * 全量应用规则可见性（初始化/重置）。条件源基线 = 静态 ∧ 权限可见：
   * 权限隐藏的条件源条件视为不成立，下游不得反推其值（设计方案 §4.2）。
   */
  function applyFieldShowRules(): void {
    if (compiledRules.ownerRuleId.size === 0) return;
    const visibility = evaluateFieldShowRules(compiledRules, {
      valueOf: (field) => state.values[field],
      isBaseVisible: baseReadable,
      currentMemberId: options.currentMemberId,
    });
    for (const [target, matched] of visibility) {
      setFieldVisible(target, matched);
    }
  }

  /** 字段基础可达性（不含规则）：静态 visible ∧ 权限可见。 */
  function baseReadable(field: string): boolean {
    const widget = itemMap.get(field)?.widget;
    return Boolean(widget?.visible) && permissionVisible(field);
  }

  /**
   * 定向重算：沿依赖图收集变更字段的下游目标闭包并按拓扑序重算。
   * 上游变化先落地再计算下一级，保证多级 A→B→C 的传播与全量求值一致；
   * 条件源取当前有效可见性（隐藏源条件不成立，不读其值）。
   */
  function recomputeDownstreamVisibility(changedKey: string): void {
    const affected = downstreamTargets(compiledRules, changedKey);
    for (const target of affected) {
      const ruleId = compiledRules.ownerRuleId.get(target);
      const rule = ruleId ? ruleById.get(ruleId) : undefined;
      const matched = rule
        ? matchFieldShowRule(rule, {
            valueOf: (field) => state.values[field],
            isFieldVisible: (field) => effectiveVisible(field),
            currentMemberId: options.currentMemberId,
          })
        : true;
      setFieldVisible(target, matched);
    }
  }

  /** 字段当前有效可见性：无状态字段回退静态值（防御）。 */
  function effectiveVisible(field: string): boolean {
    return state.fieldStates[field]?.visible ?? itemMap.get(field)?.widget.visible ?? false;
  }

  /**
   * 可见性落地（v6 单口径）：渲染与信封同为「静态 ∧ 权限 ∧ 规则命中」。
   * 隐藏保留会话值（再次显示恢复原值的体验缓存，正式记录值由服务端策略
   * 决议）并清除该字段错误；重新显示时对已触碰字段立即重校验
   * （v5 设计方案 §4.2 数据保留约定）。
   */
  function setFieldVisible(key: string, ruleMatched: boolean): void {
    const fieldState = state.fieldStates[key];
    const staticVisible = itemMap.get(key)?.widget.visible ?? false;
    const nextVisible = staticVisible && permissionVisible(key) && ruleMatched;
    if (!fieldState) return;
    if (fieldState.visible === nextVisible) return;
    fieldState.visible = nextVisible;
    if (!nextVisible) {
      fieldState.errors = [];
    } else if (fieldState.touched) {
      validateField(key);
    }
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
    // 隐藏字段（静态或规则）不校验：显隐引擎已保证隐藏字段的可见性语义。
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
    // 仅替换本地字段问题；服务端非字段错误（版本冲突提示等）保留在操作区摘要。
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

  /**
   * 组装稳定提交载荷：数据字段均携带有效可见性（静态 ∧ 权限 ∧ 规则）；
   * 有效不可见字段一律不带 data——落库值由服务端按不可见字段赋值策略决议，
   * 前端不预测更不伪造。
   */
  function buildSubmitPayload(): FormSubmitPayload {
    const values: Record<string, FormSubmittedFieldValue> = {};
    for (const item of dataItems) {
      const key = item.widget.widgetName;
      const fieldState = state.fieldStates[key];
      const visible = fieldState?.visible ?? item.widget.visible;
      const submitted: FormSubmittedFieldValue = { visible };
      values[key] = submitted;
      if (!visible || !permissionEditable(key)) continue;
      const value = state.values[key];
      if (value === undefined || value === null || value === '') continue;
      submitted.data = JSON.parse(JSON.stringify(value)) as FormValue;
    }
    return {
      formId: options.formId ?? '',
      publishedVersion: options.publishedVersion ?? 0,
      schemaRevision: options.schemaRevision ?? '',
      values,
    };
  }

  /** 填写草稿保留全部数据字段，隐藏字段也必须可在恢复填写后继续使用。 */
  function buildDraftPayload(): FormDraftPayload {
    const values: Record<string, FormValue> = {};
    for (const item of dataItems) {
      const key = item.widget.widgetName;
      values[key] = cloneFormValue(state.values[key]);
    }
    return {
      formId: options.formId ?? '',
      publishedVersion: options.publishedVersion ?? 0,
      schemaRevision: options.schemaRevision ?? '',
      values,
    };
  }

  async function submit(signal?: AbortSignal): Promise<FormSubmitOutcome> {
    if (state.activeOperation || state.lifecycle === 'submitted') {
      return { ok: false, reason: 'busy' };
    }
    if (!validateVisibleFields()) return { ok: false, reason: 'invalid' };

    const payload = buildSubmitPayload();
    const submitter = options.adapter?.submit;
    if (!submitter) return { ok: true, payload, submitted: false };

    state.activeOperation = 'submit';
    try {
      const result = await submitter(payload, signal ?? new AbortController().signal);
      if (!result.accepted) {
        applyServerFieldErrors(result.fieldErrors ?? {});
        if (result.message) addServerIssue(result.message);
        state.lifecycle = 'ready';
        return { ok: false, reason: 'server', payload };
      }
      clearServerIssues();
      state.dirtyKeys.clear();
      state.lifecycle = 'submitted';
      return { ok: true, payload, submitted: true };
    } catch (error) {
      if (isAbortError(error)) {
        // 请求取消不是业务错误：恢复可操作状态，不展示错误。
        return { ok: false, reason: 'cancelled', payload };
      }
      addServerIssue('提交失败，请稍后重试');
      state.lifecycle = 'ready';
      return { ok: false, reason: 'error', payload, error };
    } finally {
      state.activeOperation = null;
    }
  }

  /** 保存填写草稿不执行必填校验；无 adapter 时明确返回 unavailable。 */
  async function saveDraft(signal?: AbortSignal): Promise<FormDraftOutcome> {
    if (state.activeOperation || state.lifecycle === 'submitted') {
      return { ok: false, reason: 'busy' };
    }
    const saver = options.adapter?.saveDraft;
    if (!saver) return { ok: false, reason: 'unavailable' };

    const payload = buildDraftPayload();
    state.activeOperation = 'save-draft';
    try {
      await saver(payload, signal ?? new AbortController().signal);
      clearServerIssues();
      state.dirtyKeys.clear();
      return { ok: true, payload };
    } catch (error) {
      if (isAbortError(error)) return { ok: false, reason: 'cancelled', payload };
      addServerIssue('保存草稿失败，请稍后重试');
      return { ok: false, reason: 'error', payload, error };
    } finally {
      state.activeOperation = null;
    }
  }

  function reset(): void {
    initializeValues();
    state.lifecycle = 'ready';
    state.activeOperation = null;
  }

  function isDirty(): boolean {
    return state.dirtyKeys.size > 0;
  }

  /** 字段的不可见赋值策略（v6 客户端预演入口，§5.2）。 */
  function submitStrategyOf(key: string): SubmitRule {
    return resolveSubmitStrategy(submitPolicy, key);
  }

  return {
    schema,
    state,
    itemMap,
    setValue,
    markTouched,
    validateField,
    validateVisibleFields,
    submitStrategyOf,
    applyServerFieldErrors,
    addServerIssue,
    clearServerIssues,
    buildSubmitPayload,
    buildDraftPayload,
    submit,
    saveDraft,
    reset,
    isDirty,
  };
}

/** 载荷必须是会话当前值的 JSON 快照，不能把响应式对象泄漏给 Adapter。 */
function cloneFormValue(value: FormValue | undefined): FormValue {
  if (value === undefined || value === null) return null;
  return JSON.parse(JSON.stringify(value)) as FormValue;
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
  if (typeof error !== 'object' || error === null) return false;
  const cancelled = error as { code?: unknown; name?: unknown };
  // 同时兼容浏览器 AbortError 与 Axios 对 AbortSignal 的 ERR_CANCELED 包装。
  return cancelled.name === 'AbortError' || cancelled.code === 'ERR_CANCELED';
}

function readOwn(
  source: Record<string, FormValue> | undefined,
  key: string,
): FormValue | undefined {
  if (!source || !Object.prototype.hasOwnProperty.call(source, key)) return undefined;
  return source[key];
}
