/**
 * 目标保存协议严格校验器（P1，字段字典 §1/§2/§3）。
 *
 * 校验目标：任何非法输入都产出 JSON Path 级错误（与后端 Go 校验器逐字节一致），
 * 合法输入产出深拷贝后的规范文档（保证「未编辑属性不丢失」由原样落库承接）。
 * 未知键在任何层级都不被静默容忍——拒绝保存并返回精确路径（方案 §1.1）。
 */

import {
  FIELD_SHOW_CONDITION_METHODS,
  FIELD_SHOW_CURRENT_MEMBER_TYPES,
  FIELD_SHOW_EMPTY_METHODS,
  FIELD_SHOW_RULE_LIMITS,
  FORM_PROTOCOL_LIMITS,
  SUBMIT_VALIDATION_LIMITS,
  SUBMIT_VALIDATOR_FUNCTIONS,
  SUBMIT_VALIDATOR_SOURCE_TYPES,
  SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES,
  SUBMIT_RULE_LIMITS,
  SUBMIT_RULE_RECOMPUTE_SUPPORTED,
  WIDGET_OPTION_LIMITS,
  WIDGET_SPECS,
  type WidgetPropSpec,
} from './dictionary';
import { isCanonicalDateTime } from './codec';
import {
  compareDecimalText,
  DECIMAL_TEXT_PATTERN,
  decimalDigitIssue,
  effectiveNumericPrecision,
  effectiveNumericScale,
  isNumericWidgetType,
  type NumericWidgetType,
} from './numeric';
import {
  type FormSchemaDocument,
  type FormItem,
  type FormWidgetType,
  PUBLISHABLE_WIDGET_TYPES,
  SUBFORM_ALLOWED_WIDGET_TYPES,
  SUBFORM_PUBLISHABLE_WIDGET_TYPES,
} from './types';
import { cloneFormSchema } from './clone';
import { createValidationResult, type ValidationDiagnostic } from '@evolyn.do/validator';
import { FORMULA_FUNCTION_BY_NAME, parseFormula, type FormulaNode } from '@evolyn.do/formula';

/** 单条校验问题：path 为 JSON Path（如 content.items[2].widget.options[0].value）。 */
export type FormSchemaIssue = ValidationDiagnostic;

export interface FormSchemaValidationResult {
  valid: boolean;
  /** 校验通过时返回深拷贝文档；失败时为 null。 */
  document: FormSchemaDocument | null;
  issues: FormSchemaIssue[];
}

/** widgetName 形状约束（字典 1.4）：标识符形，1–64 字符。 */
const WIDGET_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;
/** 字段不可变标识（v8 契约冻结，与后端 storage.ValidateFieldID 镜像）。 */
const FIELD_ID_PATTERN = /^[0-9a-z]{10}$/;

const OPTION_WIDGET_TYPES: ReadonlySet<string> = new Set([
  'radiogroup',
  'checkboxgroup',
  'combo',
  'combocheck',
]);

export function validateFormSchema(input: unknown): FormSchemaValidationResult {
  const issues: FormSchemaIssue[] = [];
  validateRoot(input, issues);
  if (issues.length > 0) return toFormSchemaValidationResult(issues, null);
  // 形状校验通过后再深拷贝：合法文档只含 JSON 安全值，克隆不会抛错。
  return toFormSchemaValidationResult([], cloneFormSchema(input as FormSchemaDocument));
}

/**
 * 发布校验：在结构校验之上叠加能力白名单（字典 §6）。
 * 白名单外控件返回精确路径错误，交由前端提示与后端 FORM_PUBLISH_UNSUPPORTED_FIELD。
 */
export function validatePublishableFormSchema(input: unknown): FormSchemaValidationResult {
  const base = validateFormSchema(input);
  if (!base.valid || !base.document) return base;
  const issues: FormSchemaIssue[] = [];
  collectUnsupportedWidgets(base.document.content.items, 'content.items', issues);
  collectUnsupportedConditionSources(base.document.content, issues);
  if (issues.length > 0) return toFormSchemaValidationResult(issues, null);
  return base;
}

/** 将 Validator Engine 的通用结果适配回 Form 的稳定公开返回结构。 */
function toFormSchemaValidationResult(
  issues: readonly FormSchemaIssue[],
  document: FormSchemaDocument | null,
): FormSchemaValidationResult {
  const result = createValidationResult(issues, document);
  return { valid: result.valid, document: result.value, issues: [...result.issues] };
}

/** 发布期条件源白名单：未开放运行能力的字段不能作为条件源（设计方案 §3.3）。 */
function collectUnsupportedConditionSources(
  content: FormSchemaDocument['content'],
  issues: FormSchemaIssue[],
): void {
  const typesByName = new Map(
    content.items.map((item) => [item.widget.widgetName, item.widget.type]),
  );
  (Array.isArray(content.fieldShowRules) ? content.fieldShowRules : []).forEach(
    (rule, ruleIndex) => {
      const conditions = Array.isArray(rule?.filter?.cond) ? rule.filter.cond : [];
      conditions.forEach((condition, condIndex) => {
        const type = typesByName.get(condition?.field);
        // 子表单的行集合没有字段显隐条件的标量求值语义，不能作为条件源。
        if (
          type !== undefined &&
          (!PUBLISHABLE_WIDGET_TYPES.includes(type) || type === 'subform')
        ) {
          issues.push({
            path: `content.fieldShowRules[${ruleIndex}].filter.cond[${condIndex}].field`,
            message: `条件字段「${condition.field}」的运行能力尚未开放，暂不能发布`,
          });
        }
      });
    },
  );
}

function collectUnsupportedWidgets(
  items: FormItem[],
  itemsPath: string,
  issues: FormSchemaIssue[],
): void {
  items.forEach((item, index) => {
    const path = `${itemsPath}[${index}].widget.type`;
    if (!PUBLISHABLE_WIDGET_TYPES.includes(item.widget.type)) {
      issues.push({
        path,
        message: `控件「${item.widget.type}」的运行能力尚未开放，暂不能发布`,
      });
    }
    if (item.widget.type === 'subform') {
      // 子表单的字段数组位于 widget.items；其运行时支持范围比顶层字段更窄。
      collectUnsupportedSubformChildren(
        item.widget.items,
        `${itemsPath}[${index}].widget.items`,
        issues,
      );
    }
  });
}

function collectUnsupportedSubformChildren(
  items: FormItem[],
  itemsPath: string,
  issues: FormSchemaIssue[],
): void {
  items.forEach((item, index) => {
    if (SUBFORM_PUBLISHABLE_WIDGET_TYPES.includes(item.widget.type)) return;
    issues.push({
      path: `${itemsPath}[${index}].widget.type`,
      message: `子表单内控件「${item.widget.type}」的运行能力尚未开放，暂不能发布`,
    });
  });
}

// ---- 逐层校验 ----

function validateRoot(input: unknown, issues: FormSchemaIssue[]): void {
  if (!isPlainObject(input)) {
    issues.push({ path: 'content', message: '表单文档必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(input, ['content'], 'content', issues);
  const content = input.content;
  if (!isPlainObject(content)) {
    issues.push({ path: 'content', message: 'content 必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(
    content,
    [
      'type',
      'layout',
      'items',
      'layout_fields',
      'field_layout',
      'fieldShowRules',
      'submitRule',
      'widget_submit_rules',
      'validators',
      'preSubmitConfirm',
      'formEvents',
    ],
    'content',
    issues,
  );
  if (content.type !== 'form') {
    issues.push({ path: 'content.type', message: 'content.type 必须固定为 "form"' });
  }
  if (!['normal', 'grid-2', 'grid-3', 'grid-4'].includes(String(content.layout))) {
    issues.push({
      path: 'content.layout',
      message: 'layout 必须是 normal / grid-2 / grid-3 / grid-4',
    });
  }
  if (!Array.isArray(content.items)) {
    issues.push({ path: 'content.items', message: 'content.items 必须是数组' });
    return;
  }
  if (content.items.length > FORM_PROTOCOL_LIMITS.maxItems) {
    issues.push({
      path: 'content.items',
      message: `字段项数量不能超过 ${FORM_PROTOCOL_LIMITS.maxItems}`,
    });
    return;
  }
  const seenNames = new Set<string>();
  // fieldId 表单内全局唯一（跨子表单作用域共享，与 widgetName 作用域规则不同）
  const fieldIDScope = new Map<string, string>();
  content.items.forEach((item, index) => {
    validateItem(item, `content.items[${index}]`, issues, seenNames, fieldIDScope);
  });
  validateSerialNumberRules(content.items, issues);
  validateLayouts(content, seenNames, issues);
  validateFieldShowRules(content, issues);
  validateSubmitRules(content, issues);
  validateSubmitValidation(content, issues);
  validateFrontendEvents(content, issues);
}

// ---- 前端事件（v10）----

const FRONTEND_EVENT_ID_PATTERN = /^evt_[A-Za-z0-9_-]{4,60}$/;
const FRONTEND_EVENT_TOKEN_PATTERN = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;
const FRONTEND_EVENT_HEADER_PATTERN = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;
const FRONTEND_EVENT_JSON_PATH_PATTERN = /^\$response(?:\.[A-Za-z_][A-Za-z0-9_]*|\[[0-9]+\])*$/;
const FRONTEND_EVENT_XML_PATH_PATTERN =
  /^\/(?:[A-Za-z_][A-Za-z0-9_.-]*)(?:\/[A-Za-z_][A-Za-z0-9_.-]*)*$/;
// Authorization 是外部业务 API 的常规鉴权方式，允许表单事件按需透传。
const FRONTEND_EVENT_RESTRICTED_HEADERS = new Set([
  'cookie',
  'proxy-authorization',
  'set-cookie',
]);

/** v10 前端事件保存期校验；依赖索引仅校验形状，服务端执行时会重新提取。 */
function validateFrontendEvents(content: Record<string, unknown>, issues: FormSchemaIssue[]): void {
  const rawEvents = content.formEvents;
  if (!Array.isArray(rawEvents)) {
    issues.push({ path: 'content.formEvents', message: 'formEvents 必须是数组（v10 起必填）' });
    return;
  }
  if (rawEvents.length > 50) {
    issues.push({ path: 'content.formEvents', message: '前端事件数量不能超过 50' });
  }
  const fields = new Map<string, FormWidgetType>();
  for (const item of content.items as FormItem[]) {
    if (isPlainObject(item) && isPlainObject(item.widget)) {
      const name = item.widget.widgetName;
      const type = item.widget.type;
      if (typeof name === 'string' && typeof type === 'string') {
        fields.set(name, type as FormWidgetType);
      }
    }
  }
  const ids = new Set<string>();
  const names = new Set<string>();
  rawEvents.forEach((rawEvent, index) => {
    const path = `content.formEvents[${index}]`;
    if (!isPlainObject(rawEvent)) {
      issues.push({ path, message: '前端事件必须是 JSON 对象' });
      return;
    }
    rejectUnknownKeys(
      rawEvent,
      [
        'id',
        'enabled',
        'name',
        'description',
        'trigger',
        'trigger_type',
        'request_type',
        'request',
        'request_rely',
        'action',
        'action_rely',
        'subform_fill_rule',
      ],
      path,
      issues,
    );
    validateFrontendEventIdentity(rawEvent, path, ids, names, issues);
    const trigger = typeof rawEvent.trigger === 'string' ? rawEvent.trigger : '';
    if (!isFrontendEventValueField(fields.get(trigger))) {
      issues.push({ path: `${path}.trigger`, message: '触发字段必须是表单中存在的可填写字段' });
    }
    if (rawEvent.trigger_type !== 'widget') {
      issues.push({ path: `${path}.trigger_type`, message: 'trigger_type 必须固定为 "widget"' });
    }
    if (rawEvent.request_type !== 0) {
      issues.push({ path: `${path}.request_type`, message: 'request_type 必须固定为 0' });
    }
    const request = isPlainObject(rawEvent.request) ? rawEvent.request : null;
    validateFrontendEventRequest(request, `${path}.request`, fields, issues);
    validateFrontendEventDependencyList(
      rawEvent.request_rely,
      `${path}.request_rely`,
      fields,
      issues,
    );
    validateFrontendEventActions(
      rawEvent.action,
      `${path}.action`,
      trigger,
      request,
      fields,
      issues,
    );
    validateFrontendEventDependencyList(
      rawEvent.action_rely,
      `${path}.action_rely`,
      fields,
      issues,
    );
    if (rawEvent.subform_fill_rule !== 'merge' && rawEvent.subform_fill_rule !== 'replace') {
      issues.push({
        path: `${path}.subform_fill_rule`,
        message: 'subform_fill_rule 必须是 merge / replace',
      });
    }
  });
}

function validateFrontendEventIdentity(
  event: Record<string, unknown>,
  path: string,
  ids: Set<string>,
  names: Set<string>,
  issues: FormSchemaIssue[],
): void {
  const id = typeof event.id === 'string' ? event.id : '';
  if (!FRONTEND_EVENT_ID_PATTERN.test(id) || id.length > 64) {
    issues.push({ path: `${path}.id`, message: '事件 id 必须以 evt_ 开头且长度为 8–64' });
  } else if (ids.has(id)) {
    issues.push({ path: `${path}.id`, message: '事件 id 不能重复' });
  } else ids.add(id);
  const name = typeof event.name === 'string' ? event.name.trim() : '';
  if (!name || [...name].length > 64) {
    issues.push({ path: `${path}.name`, message: '事件名称必须为 1–64 个字符' });
  } else if (names.has(name)) {
    issues.push({ path: `${path}.name`, message: '事件名称不能重复' });
  } else names.add(name);
  if (typeof event.description !== 'string' || [...event.description].length > 500) {
    issues.push({
      path: `${path}.description`,
      message: '事件说明必须是不超过 500 个字符的字符串',
    });
  }
  if (typeof event.enabled !== 'boolean') {
    issues.push({ path: `${path}.enabled`, message: 'enabled 必须是布尔值' });
  }
}

function validateFrontendEventRequest(
  request: Record<string, unknown> | null,
  path: string,
  fields: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  if (!request) {
    issues.push({ path, message: 'request 必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(request, ['method', 'url', 'header', 'body', 'format'], path, issues);
  if (request.method !== 'get' && request.method !== 'post') {
    issues.push({ path: `${path}.method`, message: 'method 必须是 get / post' });
  }
  const url = typeof request.url === 'string' ? request.url : '';
  let validURL = false;
  try {
    const parsed = new URL(url);
    validURL =
      parsed.protocol === 'https:' &&
      Boolean(parsed.hostname) &&
      !parsed.username &&
      !parsed.password;
  } catch {
    validURL = false;
  }
  if (!validURL || url.length > 4_000) {
    issues.push({
      path: `${path}.url`,
      message: 'url 必须是不超过 4000 字符且不含用户信息的 HTTPS 地址',
    });
  }
  validateFrontendEventTemplate(url, `${path}.url`, fields, issues);
  validateFrontendEventEntries(request.header, `${path}.header`, true, fields, issues);
  validateFrontendEventEntries(request.body, `${path}.body`, false, fields, issues);
  if (request.format !== 'json' && request.format !== 'xml') {
    issues.push({ path: `${path}.format`, message: 'format 必须是 json / xml' });
  }
}

function validateFrontendEventEntries(
  raw: unknown,
  path: string,
  header: boolean,
  fields: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  if (!Array.isArray(raw)) {
    issues.push({ path, message: '请求参数必须是数组' });
    return;
  }
  if (raw.length > 20) issues.push({ path, message: '请求参数不能超过 20 条' });
  raw.forEach((rawEntry, index) => {
    const entryPath = `${path}[${index}]`;
    if (!isPlainObject(rawEntry)) {
      issues.push({ path: entryPath, message: '请求参数必须是 JSON 对象' });
      return;
    }
    rejectUnknownKeys(rawEntry, ['key', 'value'], entryPath, issues);
    const key = typeof rawEntry.key === 'string' ? rawEntry.key : '';
    const value = typeof rawEntry.value === 'string' ? rawEntry.value : '';
    if (!key.trim() || key.length > 256 || !value || value.length > 4_000) {
      issues.push({ path: entryPath, message: '请求参数 key/value 不能为空且不能超过长度限制' });
      return;
    }
    if (
      header &&
      (!FRONTEND_EVENT_HEADER_PATTERN.test(key.replace(FRONTEND_EVENT_TOKEN_PATTERN, 'X')) ||
        FRONTEND_EVENT_RESTRICTED_HEADERS.has(key.trim().toLowerCase()))
    ) {
      issues.push({ path: `${entryPath}.key`, message: 'Header 名称无效或属于禁止的敏感 Header' });
    }
    validateFrontendEventTemplate(key, `${entryPath}.key`, fields, issues);
    validateFrontendEventTemplate(value, `${entryPath}.value`, fields, issues);
  });
}

function validateFrontendEventActions(
  raw: unknown,
  path: string,
  trigger: string,
  request: Record<string, unknown> | null,
  fields: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  if (!Array.isArray(raw)) {
    issues.push({ path, message: 'action 必须是数组' });
    return;
  }
  if (raw.length > 50) issues.push({ path, message: '返回值映射不能超过 50 条' });
  const targets = new Set<string>();
  raw.forEach((rawAction, index) => {
    const actionPath = `${path}[${index}]`;
    if (!isPlainObject(rawAction)) {
      issues.push({ path: actionPath, message: '返回值映射必须是 JSON 对象' });
      return;
    }
    rejectUnknownKeys(rawAction, ['field', 'value'], actionPath, issues);
    const field = typeof rawAction.field === 'string' ? rawAction.field : '';
    if (!isFrontendEventValueField(fields.get(field)) || field === trigger || targets.has(field)) {
      issues.push({
        path: `${actionPath}.field`,
        message: '目标字段不存在、重复、不可写或与触发字段相同',
      });
    } else targets.add(field);
    const value = typeof rawAction.value === 'string' ? rawAction.value : '';
    const isXML = request?.format === 'xml';
    const validPath = isXML
      ? FRONTEND_EVENT_XML_PATH_PATTERN.test(value)
      : FRONTEND_EVENT_JSON_PATH_PATTERN.test(value);
    if (!value || (!validPath && !value.includes('${'))) {
      issues.push({ path: `${actionPath}.value`, message: '返回值路径或字段模板格式无效' });
    }
    validateFrontendEventTemplate(value, `${actionPath}.value`, fields, issues);
  });
}

function validateFrontendEventTemplate(
  template: string,
  path: string,
  fields: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  FRONTEND_EVENT_TOKEN_PATTERN.lastIndex = 0;
  for (const match of template.matchAll(FRONTEND_EVENT_TOKEN_PATTERN)) {
    if (!isFrontendEventValueField(fields.get(match[1] ?? ''))) {
      issues.push({ path, message: `模板引用的字段「${match[1]}」不存在或不可读取` });
    }
  }
  const stripped = template.replace(FRONTEND_EVENT_TOKEN_PATTERN, '');
  if (stripped.includes('${')) issues.push({ path, message: '字段模板包含不完整或非法令牌' });
}

function validateFrontendEventDependencyList(
  raw: unknown,
  path: string,
  fields: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  if (!Array.isArray(raw)) {
    issues.push({ path, message: '依赖索引必须是字符串数组' });
    return;
  }
  raw.forEach((value, index) => {
    if (typeof value !== 'string' || !fields.has(value)) {
      issues.push({ path: `${path}[${index}]`, message: '依赖字段不存在' });
    }
  });
}

function isFrontendEventValueField(type: FormWidgetType | undefined): boolean {
  return Boolean(type && !['separator', 'button', 'richtext'].includes(type));
}

/**
 * v7 提交校验与二次确认的保存期校验。这里仅做静态结构、字段引用、函数目录与
 * 根结果类型检查；提交时的权威求值由后端编译产物完成，浏览器不会执行源码。
 */
function validateSubmitValidation(
  content: Record<string, unknown>,
  issues: FormSchemaIssue[],
): void {
  const topItems = new Map<string, FormWidgetType>();
  for (const item of content.items as FormItem[]) {
    if (isPlainObject(item) && isPlainObject(item.widget)) {
      const name = item.widget.widgetName;
      const type = item.widget.type;
      if (typeof name === 'string' && typeof type === 'string') {
        topItems.set(name, type as FormWidgetType);
      }
    }
  }

  const rawValidators = content.validators;
  if (!Array.isArray(rawValidators)) {
    issues.push({ path: 'content.validators', message: 'validators 必须是数组（v7 起必填）' });
  } else {
    if (rawValidators.length > SUBMIT_VALIDATION_LIMITS.maxValidators) {
      issues.push({
        path: 'content.validators',
        message: `提交校验规则数量不能超过 ${SUBMIT_VALIDATION_LIMITS.maxValidators}`,
      });
    }
    rawValidators.forEach((rawValidator, index) => {
      validateSubmitValidator(rawValidator, `content.validators[${index}]`, topItems, issues);
    });
  }

  validatePreSubmitConfirm(content.preSubmitConfirm, topItems, issues);
}

function validateSubmitValidator(
  rawValidator: unknown,
  path: string,
  topItems: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  if (!isPlainObject(rawValidator)) {
    issues.push({ path, message: '提交校验规则必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(
    rawValidator,
    ['formula', 'remind', 'remark', 'realtime', 'failAction'],
    path,
    issues,
  );
  const formula = rawValidator.formula;
  if (typeof formula !== 'string' || formula.trim() === '') {
    issues.push({ path: `${path}.formula`, message: 'formula 必须是非空字符串' });
  } else if (formula.length > SUBMIT_VALIDATION_LIMITS.formulaMaxLength) {
    issues.push({
      path: `${path}.formula`,
      message: `formula 不能超过 ${SUBMIT_VALIDATION_LIMITS.formulaMaxLength} 个字符`,
    });
  } else {
    validateValidatorFormula(formula, `${path}.formula`, topItems, issues);
  }

  validateTemplate(
    rawValidator.remind,
    `${path}.remind`,
    'remind',
    SUBMIT_VALIDATION_LIMITS.remindMaxLength,
    topItems,
    true,
    issues,
  );
  if (
    typeof rawValidator.remark !== 'string' ||
    rawValidator.remark.length > SUBMIT_VALIDATION_LIMITS.remarkMaxLength
  ) {
    issues.push({
      path: `${path}.remark`,
      message: `remark 必须是不超过 ${SUBMIT_VALIDATION_LIMITS.remarkMaxLength} 个字符的字符串`,
    });
  }
  if (typeof rawValidator.realtime !== 'boolean') {
    issues.push({ path: `${path}.realtime`, message: 'realtime 必须是布尔值' });
  }
  if (rawValidator.failAction !== 0 && rawValidator.failAction !== 1) {
    issues.push({ path: `${path}.failAction`, message: 'failAction 必须是整数 0 / 1' });
  }
}

function validatePreSubmitConfirm(
  rawConfirm: unknown,
  topItems: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  const path = 'content.preSubmitConfirm';
  if (!isPlainObject(rawConfirm)) {
    issues.push({ path, message: 'preSubmitConfirm 必须是对象（v7 起必填）' });
    return;
  }
  rejectUnknownKeys(rawConfirm, ['enable', 'title', 'content'], path, issues);
  if (typeof rawConfirm.enable !== 'boolean') {
    issues.push({ path: `${path}.enable`, message: 'enable 必须是布尔值' });
  }
  validateTemplate(
    rawConfirm.title,
    `${path}.title`,
    'title',
    SUBMIT_VALIDATION_LIMITS.confirmTitleMaxLength,
    topItems,
    true,
    issues,
  );
  validateTemplate(
    rawConfirm.content,
    `${path}.content`,
    'content',
    SUBMIT_VALIDATION_LIMITS.confirmContentMaxLength,
    topItems,
    true,
    issues,
  );
}

function validateTemplate(
  rawTemplate: unknown,
  path: string,
  label: string,
  maxLength: number,
  topItems: ReadonlyMap<string, FormWidgetType>,
  required: boolean,
  issues: FormSchemaIssue[],
): void {
  if (typeof rawTemplate !== 'string' || (required && rawTemplate.trim() === '')) {
    issues.push({ path, message: `${label} 必须是非空字符串` });
    return;
  }
  if (rawTemplate.length > maxLength) {
    issues.push({ path, message: `${label} 不能超过 ${maxLength} 个字符` });
    return;
  }
  const tokenPattern = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;
  const consumed = rawTemplate.replace(tokenPattern, '');
  if (consumed.includes('${')) {
    issues.push({ path, message: `${label} 包含非法字段变量` });
  }
  for (const match of rawTemplate.matchAll(tokenPattern)) {
    const name = match[1]!;
    const type = topItems.get(name);
    if (!type) {
      issues.push({ path, message: `${label} 引用了不存在的字段「${name}」` });
    } else if (!SUBMIT_VALIDATOR_SOURCE_TYPES.includes(type)) {
      issues.push({ path, message: `${label} 字段「${name}」的类型暂不支持插值` });
    }
  }
}

function validateValidatorFormula(
  formula: string,
  path: string,
  topItems: ReadonlyMap<string, FormWidgetType>,
  issues: FormSchemaIssue[],
): void {
  const parsed = parseFormula(formula);
  for (const diagnostic of parsed.diagnostics) {
    if (diagnostic.severity === 'error') issues.push({ path, message: diagnostic.message });
  }
  if (!parsed.ast || parsed.diagnostics.some((diagnostic) => diagnostic.severity === 'error'))
    return;

  const fields = collectFormulaFieldNames(parsed.ast);
  for (const name of fields) {
    const type = topItems.get(name);
    if (!type) {
      issues.push({ path, message: `formula 引用了不存在的字段「${name}」` });
    } else if (!SUBMIT_VALIDATOR_SOURCE_TYPES.includes(type)) {
      issues.push({ path, message: `formula 字段「${name}」的类型暂不支持参与提交校验` });
    }
  }
  const calls = collectFormulaCalls(parsed.ast);
  for (const call of calls) {
    if (!SUBMIT_VALIDATOR_FUNCTIONS.has(call) || !FORMULA_FUNCTION_BY_NAME.has(call)) {
      issues.push({ path, message: `formula 使用了未开放函数「${call}」` });
    }
  }
  visitFormulaNode(parsed.ast, (node) => {
    if (node.kind !== 'call') return;
    const definition = FORMULA_FUNCTION_BY_NAME.get(node.name);
    if (!definition || !SUBMIT_VALIDATOR_FUNCTIONS.has(node.name)) return;
    if (!isFormulaArityValid(definition, node.args.length)) {
      issues.push({ path, message: `函数「${node.name}」的参数数量不正确` });
    }
  });
  validateFormulaNodeTypes(parsed.ast, topItems, path, issues);
  if (inferFormulaNodeType(parsed.ast, topItems) !== 'boolean') {
    issues.push({ path, message: 'formula 的根结果必须是 boolean' });
  }
}

/** v7 白名单函数的输入类型矩阵；unknown 仅用于已由字段/函数校验覆盖的过渡分支。 */
function validateFormulaNodeTypes(
  node: FormulaNode,
  topItems: ReadonlyMap<string, FormWidgetType>,
  path: string,
  issues: FormSchemaIssue[],
): void {
  const typeOf = (entry: FormulaNode): string => inferFormulaNodeType(entry, topItems);
  const expectType = (entry: FormulaNode | undefined, expected: string, label: string): void => {
    if (!entry) return;
    const actual = typeOf(entry);
    if (actual !== 'unknown' && actual !== expected) {
      issues.push({ path, message: `${label} 必须是 ${expected} 类型` });
    }
  };
  const expectAll = (entries: readonly FormulaNode[], expected: string, label: string): void => {
    entries.forEach((entry) => expectType(entry, expected, label));
  };
  visitFormulaNode(node, (entry) => {
    if (entry.kind === 'unary') {
      expectType(entry.argument, 'number', '一元运算参数');
      return;
    }
    if (entry.kind === 'binary') {
      const left = typeOf(entry.left);
      const right = typeOf(entry.right);
      if (['+', '-', '*', '/', '%', '^'].includes(entry.operator)) {
        expectType(entry.left, 'number', '算术运算左参数');
        expectType(entry.right, 'number', '算术运算右参数');
      } else if (
        left !== 'unknown' &&
        right !== 'unknown' &&
        (left !== right || left === 'array')
      ) {
        issues.push({ path, message: '比较运算两侧必须是相同的标量类型' });
      }
      return;
    }
    if (entry.kind !== 'call') return;
    switch (entry.name) {
      case 'AND':
      case 'OR':
        expectAll(entry.args, 'boolean', `${entry.name} 参数`);
        break;
      case 'NOT':
        expectType(entry.args[0], 'boolean', 'NOT 参数');
        break;
      case 'IF': {
        expectType(entry.args[0], 'boolean', 'IF 条件参数');
        const trueType = entry.args[1] ? typeOf(entry.args[1]) : 'unknown';
        const falseType = entry.args[2] ? typeOf(entry.args[2]) : 'unknown';
        if (trueType !== 'unknown' && falseType !== 'unknown' && trueType !== falseType) {
          issues.push({ path, message: 'IF 的两个结果参数必须类型一致' });
        }
        break;
      }
      case 'LEN':
      case 'LOWER':
      case 'UPPER':
      case 'TRIM':
        expectType(entry.args[0], 'text', `${entry.name} 参数`);
        break;
      case 'CONCATENATE':
        expectAll(entry.args, 'text', 'CONCATENATE 参数');
        break;
      case 'ABS':
      case 'ROUND':
        expectAll(entry.args, 'number', `${entry.name} 参数`);
        break;
      case 'DATE':
        expectAll(entry.args, 'number', 'DATE 参数');
        break;
      case 'DATEDIF':
        expectType(entry.args[0], 'date', 'DATEDIF 开始日期');
        expectType(entry.args[1], 'date', 'DATEDIF 结束日期');
        expectType(entry.args[2], 'text', 'DATEDIF 单位');
        break;
      default:
        break;
    }
  });
}

function isFormulaArityValid(
  definition: { minArgs?: number; maxArgs?: number; arity?: readonly number[] },
  count: number,
): boolean {
  if (definition.arity) return definition.arity.includes(count);
  return (
    (definition.minArgs === undefined || count >= definition.minArgs) &&
    (definition.maxArgs === undefined || count <= definition.maxArgs)
  );
}

function collectFormulaFieldNames(node: FormulaNode): Set<string> {
  const names = new Set<string>();
  visitFormulaNode(node, (current) => {
    if (current.kind === 'field') names.add(current.widgetName);
  });
  return names;
}

function collectFormulaCalls(node: FormulaNode): Set<string> {
  const names = new Set<string>();
  visitFormulaNode(node, (current) => {
    if (current.kind === 'call') names.add(current.name);
  });
  return names;
}

function visitFormulaNode(node: FormulaNode, visit: (node: FormulaNode) => void): void {
  visit(node);
  if (node.kind === 'array') node.elements.forEach((entry) => visitFormulaNode(entry, visit));
  if (node.kind === 'unary') visitFormulaNode(node.argument, visit);
  if (node.kind === 'binary') {
    visitFormulaNode(node.left, visit);
    visitFormulaNode(node.right, visit);
  }
  if (node.kind === 'call') node.args.forEach((entry) => visitFormulaNode(entry, visit));
}

function inferFormulaNodeType(
  node: FormulaNode,
  topItems: ReadonlyMap<string, FormWidgetType>,
): string {
  if (node.kind === 'literal') return node.valueType;
  if (node.kind === 'field') return formulaValueTypeOf(topItems.get(node.widgetName));
  if (node.kind === 'array') return 'array';
  if (node.kind === 'unary') return 'number';
  if (node.kind === 'binary') {
    return ['==', '!=', '>', '>=', '<', '<='].includes(node.operator) ? 'boolean' : 'number';
  }
  if (node.name === 'IF' && node.args.length === 3) {
    const whenTrue = inferFormulaNodeType(node.args[1]!, topItems);
    const whenFalse = inferFormulaNodeType(node.args[2]!, topItems);
    return whenTrue === whenFalse ? whenTrue : 'unknown';
  }
  return FORMULA_FUNCTION_BY_NAME.get(node.name)?.returnType ?? 'unknown';
}

function formulaValueTypeOf(type: FormWidgetType | undefined): string {
  switch (type) {
    case 'number':
      return 'number';
    case 'datetime':
      return 'date';
    case 'checkboxgroup':
    case 'combocheck':
      return 'array';
    default:
      return type ? 'text' : 'unknown';
  }
}

/**
 * v2 布局引用校验：items 仍是字段定义唯一事实源；布局数组只保存稳定键引用。
 * 标签页允许引用任意顶层字段（含未来开放的 subform），但不能引用子表单子字段。
 */
function validateLayouts(
  content: Record<string, unknown>,
  topLevelNames: Set<string>,
  issues: FormSchemaIssue[],
): void {
  const rawLayouts = content.layout_fields;
  const rawTopLayout = content.field_layout;
  if (!Array.isArray(rawLayouts)) {
    issues.push({ path: 'content.layout_fields', message: 'layout_fields 必须是数组' });
    return;
  }
  if (!Array.isArray(rawTopLayout)) {
    issues.push({ path: 'content.field_layout', message: 'field_layout 必须是数组' });
    return;
  }
  if (rawLayouts.length > FORM_PROTOCOL_LIMITS.maxLayouts) {
    issues.push({
      path: 'content.layout_fields',
      message: `布局数量不能超过 ${FORM_PROTOCOL_LIMITS.maxLayouts}`,
    });
  }

  const layoutNames = new Set<string>();
  const nodeNames = new Set(topLevelNames);
  const placedReferences = new Map<string, string>();

  rawLayouts.forEach((layout, layoutIndex) => {
    const path = `content.layout_fields[${layoutIndex}]`;
    if (!isPlainObject(layout)) {
      issues.push({ path, message: '布局项必须是 JSON 对象' });
      return;
    }
    rejectUnknownKeys(layout, ['name', 'type', 'tabStyle', 'container'], path, issues);
    const name = validateStableLayoutName(layout.name, '_layout_', `${path}.name`, issues);
    if (name) {
      if (nodeNames.has(name)) {
        issues.push({ path: `${path}.name`, message: `布局键「${name}」重复` });
      } else {
        nodeNames.add(name);
        layoutNames.add(name);
      }
    }
    if (layout.type !== 'multitab') {
      issues.push({ path: `${path}.type`, message: '当前协议仅支持 multitab 布局' });
    }
    if (layout.tabStyle !== 'style1' && layout.tabStyle !== 'style2') {
      issues.push({ path: `${path}.tabStyle`, message: 'tabStyle 必须是 style1 / style2' });
    }
    if (!Array.isArray(layout.container)) {
      issues.push({ path: `${path}.container`, message: 'container 必须是标签页数组' });
      return;
    }
    if (
      layout.container.length < 1 ||
      layout.container.length > FORM_PROTOCOL_LIMITS.maxTabsPerLayout
    ) {
      issues.push({
        path: `${path}.container`,
        message: `标签页数量必须在 1–${FORM_PROTOCOL_LIMITS.maxTabsPerLayout} 之间`,
      });
    }
    layout.container.forEach((tab, tabIndex) => {
      validateTab(
        tab,
        `${path}.container[${tabIndex}]`,
        topLevelNames,
        nodeNames,
        placedReferences,
        issues,
      );
    });
  });

  rawTopLayout.forEach((rawReference, index) => {
    const path = `content.field_layout[${index}]`;
    if (typeof rawReference !== 'string' || rawReference === '') {
      issues.push({ path, message: '顶层布局引用必须是非空字符串' });
      return;
    }
    if (!topLevelNames.has(rawReference) && !layoutNames.has(rawReference)) {
      issues.push({ path, message: `顶层引用「${rawReference}」不存在` });
      return;
    }
    registerPlacement(rawReference, path, placedReferences, issues);
  });

  for (const name of topLevelNames) {
    if (!placedReferences.has(name)) {
      issues.push({ path: 'content.field_layout', message: `顶层字段「${name}」未加入布局` });
    }
  }
  for (const name of layoutNames) {
    if (!placedReferences.has(name)) {
      issues.push({ path: 'content.field_layout', message: `布局「${name}」未加入顶层布局` });
    }
  }
}

/** 顶层字段索引：widgetName → {widget, label}，供规则引用交叉校验。 */
type TopLevelIndex = Map<string, { widget: Record<string, unknown>; label: string }>;

/**
 * 字段显隐规则校验（v5 设计方案 §4.1）：结构、字段引用、类型指纹、方法×值
 * 形状、目标唯一性、自引用与依赖图成环。与后端 schema.go 逐字镜像。
 */
function validateFieldShowRules(content: Record<string, unknown>, issues: FormSchemaIssue[]): void {
  const rawRules = content.fieldShowRules;
  if (!Array.isArray(rawRules)) {
    issues.push({
      path: 'content.fieldShowRules',
      message: 'fieldShowRules 必须是数组（v5 起必填）',
    });
    return;
  }
  if (rawRules.length > FIELD_SHOW_RULE_LIMITS.maxRules) {
    issues.push({
      path: 'content.fieldShowRules',
      message: `显隐规则数量不能超过 ${FIELD_SHOW_RULE_LIMITS.maxRules}`,
    });
  }
  const topItems: TopLevelIndex = new Map();
  for (const item of content.items as FormItem[]) {
    if (isPlainObject(item) && isPlainObject(item.widget)) {
      const name = String(item.widget.widgetName ?? '');
      if (name) topItems.set(name, { widget: item.widget, label: String(item.label ?? '') });
    }
  }

  const seenRuleIds = new Set<string>();
  const targetOwner = new Map<string, { ruleId: string; ruleIndex: number }>();

  rawRules.forEach((rawRule, ruleIndex) => {
    const rulePath = `content.fieldShowRules[${ruleIndex}]`;
    if (!isPlainObject(rawRule)) {
      issues.push({ path: rulePath, message: '规则必须是 JSON 对象' });
      return;
    }
    rejectUnknownKeys(rawRule, ['id', 'filter', 'fields'], rulePath, issues);

    const ruleId = rawRule.id;
    if (typeof ruleId !== 'string' || ruleId === '') {
      issues.push({ path: `${rulePath}.id`, message: 'id 必须是非空字符串' });
    } else if (ruleId.length > FIELD_SHOW_RULE_LIMITS.idMaxLength) {
      issues.push({
        path: `${rulePath}.id`,
        message: `id 不能超过 ${FIELD_SHOW_RULE_LIMITS.idMaxLength} 个字符`,
      });
    } else if (seenRuleIds.has(ruleId)) {
      issues.push({ path: `${rulePath}.id`, message: `规则 id「${ruleId}」重复` });
    } else {
      seenRuleIds.add(ruleId);
    }

    const filter = rawRule.filter;
    if (!isPlainObject(filter)) {
      issues.push({ path: `${rulePath}.filter`, message: 'filter 必须是 {rel, cond} 对象' });
    } else {
      rejectUnknownKeys(filter, ['rel', 'cond'], `${rulePath}.filter`, issues);
      if (filter.rel !== 'and' && filter.rel !== 'or') {
        issues.push({ path: `${rulePath}.filter.rel`, message: 'rel 必须是 and / or' });
      }
      const conditions = filter.cond;
      if (!Array.isArray(conditions)) {
        issues.push({ path: `${rulePath}.filter.cond`, message: 'cond 必须是数组' });
      } else if (
        conditions.length < 1 ||
        conditions.length > FIELD_SHOW_RULE_LIMITS.maxConditions
      ) {
        issues.push({
          path: `${rulePath}.filter.cond`,
          message: `条件数量必须在 1–${FIELD_SHOW_RULE_LIMITS.maxConditions} 之间`,
        });
      }
    }

    const fields = rawRule.fields;
    if (!Array.isArray(fields)) {
      issues.push({ path: `${rulePath}.fields`, message: 'fields 必须是数组' });
      return;
    }
    if (fields.length < 1 || fields.length > FIELD_SHOW_RULE_LIMITS.maxTargets) {
      issues.push({
        path: `${rulePath}.fields`,
        message: `目标字段数量必须在 1–${FIELD_SHOW_RULE_LIMITS.maxTargets} 之间`,
      });
    }
    const ownTargets = new Set<string>();
    fields.forEach((rawTarget, fieldIndex) => {
      const fieldPath = `${rulePath}.fields[${fieldIndex}]`;
      if (typeof rawTarget !== 'string' || rawTarget === '') {
        issues.push({ path: fieldPath, message: '目标字段必须是非空字符串' });
        return;
      }
      const entry = topItems.get(rawTarget);
      if (!entry) {
        issues.push({ path: fieldPath, message: `目标字段「${rawTarget}」不存在` });
        return;
      }
      if (entry.widget.type === 'separator' || entry.widget.type === 'button') {
        issues.push({ path: fieldPath, message: `布局控件「${rawTarget}」不能作为显隐目标` });
        return;
      }
      if (entry.widget.visible === false) {
        issues.push({
          path: fieldPath,
          message: `目标字段「${rawTarget}」是静态隐藏字段，不能作为显隐目标`,
        });
        return;
      }
      if (ownTargets.has(rawTarget)) {
        issues.push({ path: fieldPath, message: `目标字段「${rawTarget}」重复` });
        return;
      }
      ownTargets.add(rawTarget);
      const previous = targetOwner.get(rawTarget);
      if (previous) {
        issues.push({
          path: fieldPath,
          message: `目标字段「${rawTarget}」已被规则「${previous.ruleId}」使用`,
        });
        return;
      }
      targetOwner.set(rawTarget, { ruleId: String(ruleId), ruleIndex });
    });

    // 条件行校验（目标不完整时仍尽力校验，错误定位到具体条件行）。
    if (!isPlainObject(filter) || !Array.isArray(filter.cond)) return;
    filter.cond.forEach((rawCondition, condIndex) => {
      validateFieldShowCondition(
        rawCondition,
        `${rulePath}.filter.cond[${condIndex}]`,
        topItems,
        issues,
      );
    });
  });

  detectFieldShowRuleCycle(rawRules, targetOwner, issues);
}

/**
 * 不可见字段赋值校验（v6 设计方案 §3.2）：submitRule 枚举、widget_submit_rules
 * 键可处理性与值形状、冗余配置拒绝与 recompute 能力门控。与后端 schema.go 的
 * validateSubmitRules 逐字镜像。
 */
function validateSubmitRules(content: Record<string, unknown>, issues: FormSchemaIssue[]): void {
  const submitRule = content.submitRule;
  const ruleOK = isInteger(submitRule) && submitRule >= 1 && submitRule <= 3;
  if (!ruleOK) {
    issues.push({
      path: 'content.submitRule',
      message: 'submitRule 必须是 1 / 2 / 3 之一（1=保持原值，2=空值，3=始终重新计算）',
    });
  }

  const rawRules = content.widget_submit_rules;
  if (!isPlainObject(rawRules)) {
    issues.push({
      path: 'content.widget_submit_rules',
      message: 'widget_submit_rules 必须是对象（v6 起必填，空对象合法）',
    });
    return;
  }
  if (Object.keys(rawRules).length > SUBMIT_RULE_LIMITS.maxSpecialRules) {
    issues.push({
      path: 'content.widget_submit_rules',
      message: `特殊字段赋值规则数量不能超过 ${SUBMIT_RULE_LIMITS.maxSpecialRules}`,
    });
  }

  const topItems = new Map<string, Record<string, unknown>>();
  for (const item of content.items as FormItem[]) {
    if (isPlainObject(item) && isPlainObject(item.widget)) {
      const name = String(item.widget.widgetName ?? '');
      if (name) topItems.set(name, item.widget);
    }
  }

  for (const [key, rawValue] of Object.entries(rawRules)) {
    const entryPath = `content.widget_submit_rules.${key}`;
    const widget = topItems.get(key);
    if (!widget) {
      issues.push({ path: entryPath, message: `特殊规则字段「${key}」不存在` });
      continue;
    }
    if (!SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES.includes(widget.type as FormWidgetType)) {
      issues.push({
        path: entryPath,
        message: `字段「${key}」的类型不支持配置特殊赋值规则`,
      });
      continue;
    }
    if (!isInteger(rawValue) || rawValue < 1 || rawValue > 3) {
      issues.push({
        path: entryPath,
        message: `「${key}」的特殊规则必须是 1 / 2 / 3 之一`,
      });
      continue;
    }
    if (ruleOK && rawValue === submitRule) {
      issues.push({
        path: entryPath,
        message: `字段「${key}」的特殊规则与默认策略相同，无需单独配置`,
      });
      continue;
    }
    if (rawValue === 3 && !SUBMIT_RULE_RECOMPUTE_SUPPORTED) {
      issues.push({
        path: entryPath,
        message: `「始终重新计算」需要派生计算执行器，当前尚未开放，暂不能配置字段「${key}」`,
      });
    }
  }

  // submitRule=3 时，未被覆盖为 1/2 的可处理字段必须全部可重算（§3.2）：
  // 当前尚无可重算字段，存在任一未被覆盖字段即拒绝保存。
  if (ruleOK && submitRule === 3 && !SUBMIT_RULE_RECOMPUTE_SUPPORTED) {
    const uncovered = [...topItems.entries()].some(
      ([name, widget]) =>
        SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES.includes(widget.type as FormWidgetType) &&
        !(isInteger(rawRules[name]) && rawRules[name] !== 3),
    );
    if (uncovered) {
      issues.push({
        path: 'content.submitRule',
        message: '默认策略「始终重新计算」要求全部可处理字段支持重算，当前尚未开放',
      });
    }
  }
}

/** 单条件校验：字段存在性/类型指纹/方法×值形状/成员开关。 */ function validateFieldShowCondition(
  rawCondition: unknown,
  condPath: string,
  topItems: TopLevelIndex,
  issues: FormSchemaIssue[],
): void {
  if (!isPlainObject(rawCondition)) {
    issues.push({ path: condPath, message: '条件必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(
    rawCondition,
    ['field', 'type', 'method', 'value', 'includeCurrentMember'],
    condPath,
    issues,
  );
  const field = rawCondition.field;
  if (typeof field !== 'string' || field === '') {
    issues.push({ path: `${condPath}.field`, message: '条件字段必须是非空字符串' });
    return;
  }
  const entry = topItems.get(field);
  if (!entry) {
    issues.push({ path: `${condPath}.field`, message: `条件字段「${field}」不存在` });
    return;
  }
  const actualType = String(entry.widget.type ?? '');
  if (rawCondition.type !== actualType) {
    issues.push({
      path: `${condPath}.type`,
      message: `条件类型指纹与字段「${field}」的实际类型不一致`,
    });
  }
  const methods = FIELD_SHOW_CONDITION_METHODS[actualType];
  if (!methods) {
    issues.push({
      path: `${condPath}.field`,
      message: `控件「${actualType}」不能作为显隐规则条件字段`,
    });
    return;
  }
  if (entry.widget.visible === false) {
    issues.push({
      path: `${condPath}.field`,
      message: `条件字段「${field}」是静态隐藏字段，不能作为条件源`,
    });
  }

  const method = rawCondition.method;
  if (typeof method !== 'string' || !(methods as readonly string[]).includes(method)) {
    issues.push({
      path: `${condPath}.method`,
      message: `method 必须是以下枚举值之一：${methods.join(' / ')}`,
    });
    return;
  }

  const includeCurrentMember = rawCondition.includeCurrentMember;
  if (includeCurrentMember !== undefined) {
    if (!FIELD_SHOW_CURRENT_MEMBER_TYPES.has(actualType)) {
      issues.push({
        path: `${condPath}.includeCurrentMember`,
        message: 'includeCurrentMember 仅成员字段可用',
      });
    } else if (typeof includeCurrentMember !== 'boolean') {
      issues.push({
        path: `${condPath}.includeCurrentMember`,
        message: 'includeCurrentMember 必须是布尔值',
      });
    }
  }

  validateFieldShowConditionValue(rawCondition, method, entry, condPath, issues);
}

/** 条件值形状校验（设计方案 §3.3 方法×值矩阵）。 */
function validateFieldShowConditionValue(
  condition: Record<string, unknown>,
  method: string,
  entry: { widget: Record<string, unknown>; label: string },
  condPath: string,
  issues: FormSchemaIssue[],
): void {
  const widgetType = String(entry.widget.type ?? '');
  const hasValue = 'value' in condition;
  if (FIELD_SHOW_EMPTY_METHODS.has(method)) {
    if (hasValue) {
      issues.push({ path: `${condPath}.value`, message: '空值方法不允许携带 value' });
    }
    return;
  }
  if (!hasValue) {
    issues.push({ path: `${condPath}.value`, message: '缺少比较值 value' });
    return;
  }
  const value = condition.value;
  if (!Array.isArray(value)) {
    issues.push({ path: `${condPath}.value`, message: 'value 必须是数组' });
    return;
  }

  // 数量约束：单值方法恰 1 项；between 恰 2 项且有序；集合方法 1–200 项。
  const multiSelect =
    widgetType === 'checkboxgroup' ||
    widgetType === 'combocheck' ||
    widgetType === 'usergroup' ||
    widgetType === 'deptgroup';
  if (method === 'between') {
    if (value.length !== 2 || !orderedPairOk(widgetType, value[0], value[1], entry.widget)) {
      issues.push({
        path: `${condPath}.value`,
        message: 'between 的 value 必须恰好 2 项且下界不大于上界',
      });
    }
  } else if (multiSelect || method === 'in' || method === 'notIn') {
    if (value.length < 1 || value.length > FIELD_SHOW_RULE_LIMITS.maxValues) {
      issues.push({
        path: `${condPath}.value`,
        message: `该方法的 value 必须是 1–${FIELD_SHOW_RULE_LIMITS.maxValues} 项`,
      });
    }
  } else if (value.length !== 1) {
    issues.push({ path: `${condPath}.value`, message: '该方法的 value 必须恰好 1 项' });
  }

  // 逐项形状：文本/数值/日期/选项命中/成员部门标识。
  const optionValues = collectOptionValueSet(entry.widget);
  const textCap = widgetType === 'textarea' ? 2000 : widgetType === 'text' ? 1000 : 0;
  const seen = new Set<string>();
  value.forEach((rawItem, index) => {
    const itemPath = `${condPath}.value[${index}]`;
    if (widgetType === 'number') {
      if (typeof rawItem !== 'number' || !Number.isFinite(rawItem)) {
        issues.push({ path: itemPath, message: 'value 条目必须是有限数值' });
      }
      return;
    }
    if (isNumericWidgetType(widgetType)) {
      // 数值字段族条件值是 decimal string（值协议 §15/§27）。
      if (typeof rawItem !== 'string' || !DECIMAL_TEXT_PATTERN.test(rawItem)) {
        issues.push({ path: itemPath, message: 'value 条目必须是十进制数字字符串' });
      }
      return;
    }
    if (widgetType === 'datetime') {
      const format = (entry.widget.format as 'date' | 'datetime' | 'month' | 'time') ?? 'datetime';
      if (typeof rawItem !== 'string' || !isCanonicalDateTime(rawItem, format)) {
        issues.push({ path: itemPath, message: 'value 条目的日期格式不正确' });
      }
      return;
    }
    if (typeof rawItem !== 'string' || rawItem === '') {
      issues.push({ path: itemPath, message: 'value 条目必须是非空字符串' });
      return;
    }
    if (textCap > 0 && rawItem.length > textCap) {
      issues.push({ path: itemPath, message: `value 条目不能超过 ${textCap} 个字符` });
      return;
    }
    if (optionValues && !optionValues.has(rawItem)) {
      issues.push({ path: itemPath, message: 'value 条目不在字段选项范围内' });
      return;
    }
    if (
      optionValues === null &&
      !textCap &&
      rawItem.length > FORM_PROTOCOL_LIMITS.widgetNameMaxLength
    ) {
      // 成员/部门标识只校验形状，不查目录（设计方案 §4.1）。
      issues.push({
        path: itemPath,
        message: `value 条目不能超过 ${FORM_PROTOCOL_LIMITS.widgetNameMaxLength} 个字符`,
      });
      return;
    }
    if (seen.has(rawItem)) {
      issues.push({ path: itemPath, message: 'value 存在重复项' });
    }
    seen.add(rawItem);
  });
}

/** between 下界 ≤ 上界（number 数值序、datetime 规范字符串字典序）。 */
function orderedPairOk(
  widgetType: string,
  lower: unknown,
  upper: unknown,
  widget: Record<string, unknown>,
): boolean {
  if (widgetType === 'number') {
    return (
      typeof lower === 'number' &&
      typeof upper === 'number' &&
      Number.isFinite(lower) &&
      Number.isFinite(upper) &&
      lower <= upper
    );
  }
  if (isNumericWidgetType(widgetType)) {
    // 数值字段族按 decimal 值序比较（精确比较，禁 parseFloat）。
    return (
      typeof lower === 'string' &&
      typeof upper === 'string' &&
      DECIMAL_TEXT_PATTERN.test(lower) &&
      DECIMAL_TEXT_PATTERN.test(upper) &&
      compareDecimalText(lower, upper) !== 1
    );
  }
  if (widgetType === 'datetime') {
    if (typeof lower !== 'string' || typeof upper !== 'string') return false;
    const format = (widget.format as 'date' | 'datetime' | 'month' | 'time') ?? 'datetime';
    return (
      isCanonicalDateTime(lower, format) && isCanonicalDateTime(upper, format) && lower <= upper
    );
  }
  return false;
}

/** 选项类控件返回选项 value 集合；其余返回 null（不做选项命中校验）。 */
function collectOptionValueSet(widget: Record<string, unknown>): Set<string> | null {
  if (!OPTION_WIDGET_TYPES.has(String(widget.type ?? ''))) return null;
  const values = new Set<string>();
  if (Array.isArray(widget.options)) {
    for (const option of widget.options) {
      if (isPlainOption(option) && option.value !== '') values.add(option.value);
    }
  }
  return values;
}

/**
 * 依赖图环检测（设计方案 §4.1）：规则图边方向为「条件源 → 目标字段」，
 * 按规则数组序构图（与 Go 侧同构遍历，保证两侧对同一文档产出同一错误）；
 * 发现环即报错并给出参与环的规则 id 与字段路径。
 */
function detectFieldShowRuleCycle(
  rawRules: unknown[],
  targetOwner: Map<string, { ruleId: string; ruleIndex: number }>,
  issues: FormSchemaIssue[],
): void {
  // 邻接表与节点序（保持插入序）；anchor 记录边归属用于错误定位。
  const adjacency = new Map<string, string[]>();
  const edgeAnchor = new Map<string, { ruleIndex: number; fieldIndex: number }>();
  const nodes: string[] = [];
  const nodeSeen = new Set<string>();
  const addNode = (node: string) => {
    if (node !== '' && !nodeSeen.has(node)) {
      nodeSeen.add(node);
      nodes.push(node);
    }
  };
  const pushEdge = (
    source: string,
    target: string,
    anchor: { ruleIndex: number; fieldIndex: number },
  ) => {
    const key = `${source}→${target}`;
    if (!edgeAnchor.has(key)) edgeAnchor.set(key, anchor);
    const list = adjacency.get(source) ?? [];
    if (!list.includes(target)) {
      list.push(target);
      adjacency.set(source, list);
    }
  };
  rawRules.forEach((rawRule, ruleIndex) => {
    if (!isPlainObject(rawRule)) return;
    const filter = rawRule.filter;
    const conditions = isPlainObject(filter) && Array.isArray(filter.cond) ? filter.cond : [];
    const fields = Array.isArray(rawRule.fields) ? rawRule.fields : [];
    fields.forEach((target, fieldIndex) => {
      if (typeof target !== 'string') return;
      for (const rawCondition of conditions) {
        const source = isPlainObject(rawCondition) ? String(rawCondition.field ?? '') : '';
        if (!source) continue;
        pushEdge(source, target, { ruleIndex, fieldIndex });
        addNode(source);
        addNode(target);
      }
    });
  });
  if (nodes.length === 0) return;

  const WHITE = 0;
  const GRAY = 1;
  const BLACK = 2;
  const color = new Map<string, number>(nodes.map((node) => [node, WHITE]));

  /** 迭代 DFS：发现回边时提取环（首个环即返回）。 */
  const cycle = ((): string[] | null => {
    for (const start of nodes) {
      if (color.get(start) !== WHITE) continue;
      color.set(start, GRAY);
      const stack: Array<{ node: string; edgeIndex: number }> = [{ node: start, edgeIndex: 0 }];
      const path: string[] = [start];
      while (stack.length > 0) {
        const frame = stack[stack.length - 1]!;
        const neighbors = adjacency.get(frame.node) ?? [];
        if (frame.edgeIndex >= neighbors.length) {
          color.set(frame.node, BLACK);
          stack.pop();
          path.pop();
          continue;
        }
        const neighbor = neighbors[frame.edgeIndex]!;
        frame.edgeIndex += 1;
        const neighborColor = color.get(neighbor) ?? WHITE;
        if (neighborColor === GRAY) {
          // 回边：从 path 中 neighbor 的位置截取环。
          const startAt = path.lastIndexOf(neighbor);
          return [...path.slice(startAt), neighbor];
        }
        if (neighborColor === WHITE) {
          color.set(neighbor, GRAY);
          path.push(neighbor);
          stack.push({ node: neighbor, edgeIndex: 0 });
        }
      }
    }
    return null;
  })();

  if (!cycle) return;
  // 参与环的规则 id（按环上边归属去重，保持出现顺序）。
  const ruleIds: string[] = [];
  for (let i = 0; i + 1 < cycle.length; i += 1) {
    const source = cycle[i];
    const target = cycle[i + 1];
    if (!source || !target) continue;
    const owner = edgeAnchor.has(`${source}→${target}`) ? targetOwner.get(target) : undefined;
    const ruleId = owner?.ruleId ?? '';
    if (ruleId && !ruleIds.includes(ruleId)) ruleIds.push(ruleId);
  }
  const closingEdge =
    edgeAnchor.get(`${cycle[cycle.length - 2]}→${cycle[cycle.length - 1]}`) ??
    [...edgeAnchor.values()][0]!;
  issues.push({
    path: `content.fieldShowRules[${closingEdge.ruleIndex}].fields[${closingEdge.fieldIndex}]`,
    message: `显隐规则存在循环依赖：${cycle.join(' → ')}（涉及规则 ${ruleIds.join('、') || '未知'}）`,
  });
}

function validateTab(
  rawTab: unknown,
  path: string,
  topLevelNames: Set<string>,
  nodeNames: Set<string>,
  placedReferences: Map<string, string>,
  issues: FormSchemaIssue[],
): void {
  if (!isPlainObject(rawTab)) {
    issues.push({ path, message: '标签页必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(rawTab, ['name', 'title', 'type', 'field_layout'], path, issues);
  const name = validateStableLayoutName(rawTab.name, '_tab_', `${path}.name`, issues);
  if (name) {
    if (nodeNames.has(name)) {
      issues.push({ path: `${path}.name`, message: `标签页键「${name}」重复` });
    } else {
      nodeNames.add(name);
    }
  }
  if (rawTab.type !== 'tab') {
    issues.push({ path: `${path}.type`, message: '标签页 type 必须固定为 tab' });
  }
  if (typeof rawTab.title !== 'string' || rawTab.title.trim() === '') {
    issues.push({ path: `${path}.title`, message: '标签页标题不能为空' });
  } else if (rawTab.title.length > FORM_PROTOCOL_LIMITS.labelMaxLength) {
    issues.push({
      path: `${path}.title`,
      message: `标签页标题不能超过 ${FORM_PROTOCOL_LIMITS.labelMaxLength} 个字符`,
    });
  }
  if (!Array.isArray(rawTab.field_layout)) {
    issues.push({ path: `${path}.field_layout`, message: '标签页 field_layout 必须是数组' });
    return;
  }
  rawTab.field_layout.forEach((rawReference, index) => {
    const refPath = `${path}.field_layout[${index}]`;
    if (typeof rawReference !== 'string' || rawReference === '') {
      issues.push({ path: refPath, message: '标签页字段引用必须是非空字符串' });
      return;
    }
    if (!topLevelNames.has(rawReference)) {
      issues.push({ path: refPath, message: `字段引用「${rawReference}」不是顶层字段` });
      return;
    }
    registerPlacement(rawReference, refPath, placedReferences, issues);
  });
}

function validateStableLayoutName(
  value: unknown,
  prefix: '_layout_' | '_tab_',
  path: string,
  issues: FormSchemaIssue[],
): string | null {
  if (typeof value !== 'string' || !value.startsWith(prefix) || !WIDGET_NAME_PATTERN.test(value)) {
    issues.push({ path, message: `${prefix === '_layout_' ? '布局' : '标签页'}键格式不正确` });
    return null;
  }
  if (value.length > FORM_PROTOCOL_LIMITS.widgetNameMaxLength) {
    issues.push({ path, message: '稳定键不能超过 64 个字符' });
    return null;
  }
  return value;
}

function registerPlacement(
  reference: string,
  path: string,
  placements: Map<string, string>,
  issues: FormSchemaIssue[],
): void {
  const previous = placements.get(reference);
  if (previous) {
    issues.push({ path, message: `引用「${reference}」重复，已在 ${previous} 使用` });
    return;
  }
  placements.set(reference, path);
}

/**
 * 校验单个字段项。seenNames 是当前作用域（顶层或某个子表单）内已见的 widgetName
 * 集合——字典 1.4：顶层全表单唯一，子表单按作用域唯一。
 */
function validateItem(
  input: unknown,
  path: string,
  issues: FormSchemaIssue[],
  seenNames: Set<string>,
  fieldIDScope: Map<string, string>,
): void {
  if (!isPlainObject(input)) {
    issues.push({ path, message: '字段项必须是 JSON 对象' });
    return;
  }
  rejectUnknownKeys(
    input,
    ['widget', 'label', 'description', 'labelHidden', 'lineWidth'],
    path,
    issues,
  );

  const type = isPlainObject(input.widget) ? String(input.widget.type ?? '') : '';

  validateLabel(input.label, type, `${path}.label`, issues);
  if (typeof input.description !== 'string') {
    issues.push({
      path: `${path}.description`,
      message: 'description 必须是字符串（空串即「无」）',
    });
  } else if (input.description.length > FORM_PROTOCOL_LIMITS.descriptionMaxLength) {
    issues.push({
      path: `${path}.description`,
      message: `说明不能超过 ${FORM_PROTOCOL_LIMITS.descriptionMaxLength} 个字符`,
    });
  }
  if (typeof input.labelHidden !== 'boolean') {
    issues.push({ path: `${path}.labelHidden`, message: 'labelHidden 必须是布尔值' });
  }
  if (!isInteger(input.lineWidth)) {
    issues.push({ path: `${path}.lineWidth`, message: 'lineWidth 必须是整数' });
  } else if (
    input.lineWidth < FORM_PROTOCOL_LIMITS.lineWidthRange.min ||
    input.lineWidth > FORM_PROTOCOL_LIMITS.lineWidthRange.max
  ) {
    issues.push({
      path: `${path}.lineWidth`,
      message: `lineWidth 必须在 ${FORM_PROTOCOL_LIMITS.lineWidthRange.min}–${FORM_PROTOCOL_LIMITS.lineWidthRange.max} 之间`,
    });
  } else if (type === 'subform' && input.lineWidth !== 12) {
    issues.push({ path: `${path}.lineWidth`, message: '子表单必须固定占整行（lineWidth=12）' });
  }
  validateWidget(input.widget, `${path}.widget`, issues, seenNames, fieldIDScope);
}

function validateLabel(
  label: unknown,
  type: string,
  path: string,
  issues: FormSchemaIssue[],
): void {
  const spec = (WIDGET_SPECS as Record<string, { labelOptional?: boolean } | undefined>)[type];
  const optional = spec?.labelOptional ?? false;
  if (typeof label !== 'string') {
    issues.push({ path, message: 'label 必须是字符串' });
    return;
  }
  if (!optional && label.trim() === '') {
    issues.push({ path, message: 'label 不能为空' });
  }
  if (label.length > FORM_PROTOCOL_LIMITS.labelMaxLength) {
    issues.push({
      path,
      message: `label 不能超过 ${FORM_PROTOCOL_LIMITS.labelMaxLength} 个字符`,
    });
  }
}

function validateWidget(
  input: unknown,
  path: string,
  issues: FormSchemaIssue[],
  seenNames: Set<string>,
  fieldIDScope: Map<string, string>,
): void {
  if (!isPlainObject(input)) {
    issues.push({ path, message: 'widget 必须是 JSON 对象' });
    return;
  }
  const type = input.type;
  if (typeof type !== 'string' || !(type in WIDGET_SPECS)) {
    issues.push({
      path: `${path}.type`,
      message: `未知的控件类型：${typeof type === 'string' ? type : '非字符串'}`,
    });
    return;
  }
  const spec = WIDGET_SPECS[type as FormWidgetType];
  // 未知键拒绝：公共四键 + 本类型专属属性表之外的键全部非法。
  const allowedKeys = [
    'type',
    'widgetName',
    'fieldId',
    'enable',
    'visible',
    'allowBlank',
    ...Object.keys(spec.props),
  ];
  rejectUnknownKeys(input, allowedKeys, path, issues);

  if (typeof input.widgetName !== 'string' || input.widgetName === '') {
    issues.push({ path: `${path}.widgetName`, message: 'widgetName 必须是非空字符串' });
  } else if (
    input.widgetName.length > FORM_PROTOCOL_LIMITS.widgetNameMaxLength ||
    !WIDGET_NAME_PATTERN.test(input.widgetName)
  ) {
    issues.push({
      path: `${path}.widgetName`,
      message: 'widgetName 必须是 1–64 位的字母/数字/下划线标识符（且以字母或下划线开头）',
    });
  } else if (seenNames.has(input.widgetName)) {
    issues.push({
      path: `${path}.widgetName`,
      message: `字段键「${input.widgetName}」在当前作用域内重复`,
    });
  } else {
    seenNames.add(input.widgetName);
  }

  // v8 契约冻结（物理表存储 §4.1）：值字段必须携带不可变 fieldId，表单内
  //（含全部子表单）全局唯一；布局/按钮无记录值不参与物理模型，不分配。
  const widgetType = type;
  if (widgetType !== 'separator' && widgetType !== 'button') {
    if (typeof input.fieldId !== 'string' || input.fieldId === '') {
      issues.push({ path: `${path}.fieldId`, message: '字段缺少不可变标识 fieldId' });
    } else if (!FIELD_ID_PATTERN.test(input.fieldId)) {
      issues.push({ path: `${path}.fieldId`, message: 'fieldId 必须是 10 位小写字母/数字' });
    } else if (fieldIDScope.has(input.fieldId)) {
      issues.push({
        path: `${path}.fieldId`,
        message: `fieldId「${input.fieldId}」重复，已在 ${fieldIDScope.get(input.fieldId)} 使用`,
      });
    } else {
      fieldIDScope.set(input.fieldId, path);
    }
  }

  for (const key of ['enable', 'visible', 'allowBlank'] as const) {
    if (typeof input[key] !== 'boolean') {
      issues.push({ path: `${path}.${key}`, message: `${key} 必须是布尔值（不允许 null/缺省）` });
    }
  }

  for (const [key, propSpec] of Object.entries(spec.props)) {
    if (!(key in input)) {
      if (propSpec.required) {
        issues.push({ path: `${path}.${key}`, message: `缺少必填属性 ${key}` });
      }
      continue;
    }
    validateWidgetProp(input[key], propSpec, `${path}.${key}`, issues, fieldIDScope);
  }

  // 类型间交叉约束（字典逐条对应的 min≤max 系列）。
  validateWidgetCrossRules(input as Record<string, unknown>, type as FormWidgetType, path, issues);
}

function validateWidgetProp(
  value: unknown,
  spec: WidgetPropSpec,
  path: string,
  issues: FormSchemaIssue[],
  fieldIDScope: Map<string, string>,
): void {
  switch (spec.kind) {
    case 'boolean':
      if (typeof value !== 'boolean') {
        issues.push({ path, message: `${path.split('.').pop()} 必须是布尔值` });
      }
      return;
    case 'string':
      if (typeof value !== 'string') {
        issues.push({ path, message: `${path.split('.').pop()} 必须是字符串` });
      } else if (spec.maxLen !== undefined && value.length > spec.maxLen) {
        issues.push({ path, message: `${path.split('.').pop()} 不能超过 ${spec.maxLen} 个字符` });
      }
      return;
    case 'integer':
      if (value === null) return; // 未启用语义（与缺省一致，字典 1.2）
      if (!isInteger(value)) {
        issues.push({ path, message: `${path.split('.').pop()} 必须是整数（null 表示未启用）` });
      } else if (!inRange(value, spec)) {
        issues.push({
          path,
          message: `${path.split('.').pop()} 不在允许范围 ${rangeText(spec)} 内`,
        });
      }
      return;
    case 'number':
      if (value === null) return; // 未启用语义
      if (typeof value !== 'number' || !Number.isFinite(value)) {
        issues.push({
          path,
          message: `${path.split('.').pop()} 必须是有限数值（null 表示未启用）`,
        });
      } else if (!inRange(value, spec)) {
        issues.push({
          path,
          message: `${path.split('.').pop()} 不在允许范围 ${rangeText(spec)} 内`,
        });
      }
      return;
    case 'enum':
      if (typeof value !== 'string' || !spec.values?.includes(value)) {
        issues.push({
          path,
          message: `${path.split('.').pop()} 必须是以下枚举值之一：${spec.values?.join(' / ') ?? ''}`,
        });
      }
      return;
    case 'decimal':
      // 数值字段族的 min/max/defaultValue：decimal string（null=未启用），
      // 形状拒绝指数记法与正号（canonical 协议 §49）；位数约束在交叉规则复核。
      if (value === null) return;
      if (typeof value !== 'string' || !DECIMAL_TEXT_PATTERN.test(value)) {
        issues.push({
          path,
          message: `${path.split('.').pop()} 必须是十进制数字字符串（null 表示未启用）`,
        });
      }
      return;
    case 'stringArray':
      if (!Array.isArray(value)) {
        issues.push({ path, message: `${path.split('.').pop()} 必须是字符串数组` });
        return;
      }
      if (spec.maxItems !== undefined && value.length > spec.maxItems) {
        issues.push({ path, message: `${path.split('.').pop()} 条目数不能超过 ${spec.maxItems}` });
      }
      value.forEach((entry, index) => {
        if (typeof entry !== 'string' || entry === '') {
          issues.push({ path: `${path}[${index}]`, message: '数组条目必须是非空字符串' });
        }
      });
      return;
    case 'options':
      validateOptions(value, path, issues);
      return;
    case 'widgetItems':
      validateSubformItems(value, path, issues, fieldIDScope);
      return;
    case 'stickyColumn':
      validateStickyColumn(value, path, issues);
      return;
    case 'linkFilters':
      validateLinkFilters(value, path, issues);
      return;
    case 'linkSorts':
      validateLinkSorts(value, path, issues);
      return;
    case 'linkMappings':
      validateLinkMappings(value, path, issues);
      return;
    case 'expression':
      validateAggregationExpression(value, path, issues);
      return;
    case 'snRules':
      validateSnRules(value, path, issues);
      return;
    case 'buttonAction':
      validateButtonAction(value, path, issues);
      return;
  }
}

function validateStickyColumn(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!isPlainObject(value)) {
    issues.push({ path, message: '冻结列配置必须是 {enable, limit} 对象' });
    return;
  }
  rejectUnknownKeys(value, ['enable', 'limit'], path, issues);
  if (typeof value.enable !== 'boolean') {
    issues.push({ path: `${path}.enable`, message: 'enable 必须是布尔值' });
  }
  if (!isInteger(value.limit) || value.limit < 1 || value.limit > 5) {
    issues.push({ path: `${path}.limit`, message: 'limit 必须是 1–5 之间的整数' });
  }
}

function validateOptions(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!Array.isArray(value)) {
    issues.push({ path, message: 'options 必须是数组' });
    return;
  }
  if (
    value.length < WIDGET_OPTION_LIMITS.minItems ||
    value.length > WIDGET_OPTION_LIMITS.maxItems
  ) {
    issues.push({
      path,
      message: `选项数量必须在 ${WIDGET_OPTION_LIMITS.minItems}–${WIDGET_OPTION_LIMITS.maxItems} 之间`,
    });
  }
  const seenValues = new Set<string>();
  value.forEach((option, index) => {
    const optionPath = `${path}[${index}]`;
    if (!isPlainObject(option)) {
      issues.push({ path: optionPath, message: '选项必须是 {label, value} 对象' });
      return;
    }
    rejectUnknownKeys(option, ['label', 'value'], optionPath, issues);
    for (const key of ['label', 'value'] as const) {
      if (typeof option[key] !== 'string' || option[key] === '') {
        issues.push({ path: `${optionPath}.${key}`, message: `选项 ${key} 必须是非空字符串` });
      } else if (option[key].length > WIDGET_OPTION_LIMITS.textMaxLength) {
        issues.push({
          path: `${optionPath}.${key}`,
          message: `选项 ${key} 不能超过 ${WIDGET_OPTION_LIMITS.textMaxLength} 个字符`,
        });
      }
    }
    if (typeof option.value === 'string') {
      if (seenValues.has(option.value)) {
        issues.push({ path: `${optionPath}.value`, message: `选项值「${option.value}」重复` });
      }
      seenValues.add(option.value);
    }
  });
}

function validateSubformItems(
  value: unknown,
  path: string,
  issues: FormSchemaIssue[],
  fieldIDScope: Map<string, string>,
): void {
  if (!Array.isArray(value)) {
    issues.push({ path, message: '子表单 items 必须是数组' });
    return;
  }
  if (value.length > FORM_PROTOCOL_LIMITS.subformMaxItems) {
    issues.push({
      path,
      message: `子表单字段数不能超过 ${FORM_PROTOCOL_LIMITS.subformMaxItems}`,
    });
  }
  // 子作用域独立命名空间；先校验子项类型白名单再递归复用 validateItem。
  const scopeNames = new Set<string>();
  value.forEach((child, index) => {
    const childPath = `${path}[${index}]`;
    const childType =
      isPlainObject(child) && isPlainObject(child.widget) ? child.widget.type : undefined;
    if (
      typeof childType === 'string' &&
      !SUBFORM_ALLOWED_WIDGET_TYPES.includes(childType as FormWidgetType)
    ) {
      issues.push({
        path: `${childPath}.widget.type`,
        message: `子表单内不允许使用控件「${childType}」`,
      });
      return;
    }
    validateItem(child, childPath, issues, scopeNames, fieldIDScope);
  });
}

function validateLinkFilters(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!Array.isArray(value)) {
    issues.push({ path, message: 'filters 必须是数组' });
    return;
  }
  const ops = ['eq', 'ne', 'gt', 'lt', 'ge', 'le', 'contains'];
  value.forEach((filter, index) => {
    const filterPath = `${path}[${index}]`;
    if (!isPlainObject(filter)) {
      issues.push({ path: filterPath, message: '过滤条件必须是 {field, op, value} 对象' });
      return;
    }
    rejectUnknownKeys(filter, ['field', 'op', 'value'], filterPath, issues);
    if (typeof filter.field !== 'string' || filter.field === '') {
      issues.push({ path: `${filterPath}.field`, message: '过滤条件 field 必须是非空字符串' });
    }
    if (typeof filter.op !== 'string' || !ops.includes(filter.op)) {
      issues.push({
        path: `${filterPath}.op`,
        message: `过滤条件 op 必须是以下枚举值之一：${ops.join(' / ')}`,
      });
    }
  });
}

function validateLinkSorts(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!Array.isArray(value)) {
    issues.push({ path, message: 'sorts 必须是数组' });
    return;
  }
  value.forEach((sort, index) => {
    const sortPath = `${path}[${index}]`;
    if (!isPlainObject(sort)) {
      issues.push({ path: sortPath, message: '排序项必须是 {field, order} 对象' });
      return;
    }
    rejectUnknownKeys(sort, ['field', 'order'], sortPath, issues);
    if (typeof sort.field !== 'string' || sort.field === '') {
      issues.push({ path: `${sortPath}.field`, message: '排序项 field 必须是非空字符串' });
    }
    if (sort.order !== 'asc' && sort.order !== 'desc') {
      issues.push({ path: `${sortPath}.order`, message: '排序项 order 必须是 asc / desc' });
    }
  });
}

function validateLinkMappings(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!Array.isArray(value)) {
    issues.push({ path, message: 'mappings 必须是数组' });
    return;
  }
  value.forEach((mapping, index) => {
    const mappingPath = `${path}[${index}]`;
    if (!isPlainObject(mapping)) {
      issues.push({ path: mappingPath, message: '映射项必须是 {source, target} 对象' });
      return;
    }
    rejectUnknownKeys(mapping, ['source', 'target'], mappingPath, issues);
    for (const key of ['source', 'target'] as const) {
      if (typeof mapping[key] !== 'string' || mapping[key] === '') {
        issues.push({ path: `${mappingPath}.${key}`, message: `映射项 ${key} 必须是非空字符串` });
      }
    }
  });
}

function validateAggregationExpression(
  value: unknown,
  path: string,
  issues: FormSchemaIssue[],
): void {
  if (value === null) return; // 未启用
  if (!isPlainObject(value)) {
    issues.push({ path, message: 'expression 必须是 {op, source, field?} 对象或 null' });
    return;
  }
  rejectUnknownKeys(value, ['op', 'source', 'field'], path, issues);
  const ops = ['sum', 'avg', 'count', 'min', 'max'];
  if (typeof value.op !== 'string' || !ops.includes(value.op)) {
    issues.push({
      path: `${path}.op`,
      message: `聚合 op 必须是以下枚举值之一：${ops.join(' / ')}`,
    });
  }
  if (typeof value.source !== 'string' || value.source === '') {
    issues.push({ path: `${path}.source`, message: '聚合 source 必须是非空字符串（源字段键）' });
  }
  if (value.field !== undefined && (typeof value.field !== 'string' || value.field === '')) {
    issues.push({ path: `${path}.field`, message: '聚合 field 必须是非空字符串或省略' });
  }
  if (value.op === 'count' && value.field !== undefined) {
    issues.push({ path: `${path}.field`, message: 'op=count 时不允许携带 field' });
  }
}

/** yyyy / MM / dd 与 -、/ 是流水号日期片段允许的全部格式语法。 */
function isValidSnDateFormat(value: unknown): value is string {
  if (typeof value !== 'string' || value.length === 0 || value.length > 32) return false;
  const tokens = value.match(/yyyy|MM|dd|[-/]/g);
  return tokens?.join('') === value && /yyyy|MM|dd/.test(value);
}

function formatContainsDateTokens(format: string, ...tokens: string[]): boolean {
  return tokens.every((token) => format.includes(token));
}

function validateSnRules(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!Array.isArray(value) || value.length === 0 || value.length > 10) {
    issues.push({ path, message: 'rules 必须是包含 1–10 个片段的数组' });
    return;
  }
  let counters = 0;
  value.forEach((part, index) => {
    const partPath = `${path}[${index}]`;
    if (!isPlainObject(part)) {
      issues.push({ path: partPath, message: '流水号片段必须是对象' });
      return;
    }
    if (part.type === 'counter') {
      counters += 1;
      rejectUnknownKeys(
        part,
        ['type', 'digits', 'fixedWidth', 'resetCycle', 'initialValue'],
        partPath,
        issues,
      );
      if (!isInteger(part.digits) || part.digits < 3 || part.digits > 8)
        issues.push({ path: `${partPath}.digits`, message: '计数位数必须是 3–8 的整数' });
      if (typeof part.fixedWidth !== 'boolean')
        issues.push({ path: `${partPath}.fixedWidth`, message: 'fixedWidth 必须是布尔值' });
      if (!['none', 'daily', 'monthly', 'yearly'].includes(String(part.resetCycle)))
        issues.push({
          path: `${partPath}.resetCycle`,
          message: 'resetCycle 必须是 none / daily / monthly / yearly',
        });
      if (!isInteger(part.initialValue) || part.initialValue < 1 || part.initialValue > 99_999_999)
        issues.push({
          path: `${partPath}.initialValue`,
          message: 'initialValue 必须是 1–99999999 的整数',
        });
      return;
    }
    if (part.type === 'submittedAt') {
      rejectUnknownKeys(part, ['type', 'format', 'formatType'], partPath, issues);
      if (!isValidSnDateFormat(part.format))
        issues.push({ path: `${partPath}.format`, message: '日期格式仅支持 yyyy、MM、dd 与 -、/' });
      if (
        part.formatType !== undefined &&
        !['preset', 'custom'].includes(String(part.formatType))
      ) {
        issues.push({
          path: `${partPath}.formatType`,
          message: 'formatType 必须是 preset / custom',
        });
      }
      return;
    }
    if (part.type === 'literal') {
      rejectUnknownKeys(part, ['type', 'value'], partPath, issues);
      if (typeof part.value !== 'string' || part.value.length > 32)
        issues.push({
          path: `${partPath}.value`,
          message: '固定字符必须是不超过 32 个字符的字符串',
        });
      return;
    }
    if (part.type === 'field') {
      rejectUnknownKeys(part, ['type', 'fieldId'], partPath, issues);
      if (typeof part.fieldId !== 'string' || !FIELD_ID_PATTERN.test(part.fieldId))
        issues.push({ path: `${partPath}.fieldId`, message: 'fieldId 必须是 10 位小写字母/数字' });
      return;
    }
    issues.push({
      path: `${partPath}.type`,
      message: '流水号片段 type 必须是 counter / submittedAt / literal / field',
    });
  });
  if (counters !== 1) issues.push({ path, message: '流水号必须且只能包含一个自动计数片段' });
}

/** 规则引用和重置周期约束需要在顶层字段完成解析后校验。 */
function validateSerialNumberRules(items: unknown[], issues: FormSchemaIssue[]): void {
  const fieldsByID = new Map<string, { type: string; path: string }>();
  items.forEach((rawItem, index) => {
    if (!isPlainObject(rawItem) || !isPlainObject(rawItem.widget)) return;
    if (typeof rawItem.widget.fieldId === 'string' && typeof rawItem.widget.type === 'string') {
      fieldsByID.set(rawItem.widget.fieldId, {
        type: rawItem.widget.type,
        path: `content.items[${index}].widget`,
      });
    }
  });
  items.forEach((rawItem, index) => {
    if (
      !isPlainObject(rawItem) ||
      !isPlainObject(rawItem.widget) ||
      rawItem.widget.type !== 'sn' ||
      !Array.isArray(rawItem.widget.rules)
    )
      return;
    const dateFormats = rawItem.widget.rules.reduce<string[]>((formats, part) => {
      if (isPlainObject(part) && part.type === 'submittedAt' && isValidSnDateFormat(part.format))
        formats.push(part.format);
      return formats;
    }, []);
    rawItem.widget.rules.forEach((part, partIndex) => {
      if (!isPlainObject(part) || part.type !== 'field') return;
      const target = fieldsByID.get(part.fieldId as string);
      const path = `content.items[${index}].widget.rules[${partIndex}].fieldId`;
      if (!target) issues.push({ path, message: '流水号引用了不存在的表单字段' });
      else if (
        ![
          'text',
          'textarea',
          'number',
          'decimal',
          'money',
          'percent',
          'datetime',
          'radiogroup',
          'combo',
        ].includes(target.type)
      )
        issues.push({ path, message: `字段类型「${target.type}」暂不支持拼接到流水号` });
      else if (target.path === `content.items[${index}].widget`)
        issues.push({ path, message: '流水号不能引用自身' });
    });
    const counter = rawItem.widget.rules.find(
      (part) => isPlainObject(part) && part.type === 'counter',
    );
    if (!isPlainObject(counter) || counter.resetCycle === 'none') return;
    const compatible =
      (counter.resetCycle === 'daily' &&
        dateFormats.some((format) => formatContainsDateTokens(format, 'yyyy', 'MM', 'dd'))) ||
      (counter.resetCycle === 'monthly' &&
        dateFormats.some((format) => formatContainsDateTokens(format, 'yyyy', 'MM'))) ||
      (counter.resetCycle === 'yearly' &&
        dateFormats.some((format) => formatContainsDateTokens(format, 'yyyy')));
    if (!compatible)
      issues.push({
        path: `content.items[${index}].widget.rules`,
        message: '启用周期重置时必须包含与重置周期匹配的提交日期片段，避免号码重复',
      });
  });
}

function validateButtonAction(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!isPlainObject(value)) {
    issues.push({ path, message: 'action 必须是 {type} 对象' });
    return;
  }
  rejectUnknownKeys(value, ['type'], path, issues);
  const types = ['none', 'submit'];
  if (typeof value.type !== 'string' || !types.includes(value.type)) {
    issues.push({
      path: `${path}.type`,
      message: `action.type 必须是以下枚举值之一：${types.join(' / ')}`,
    });
  }
}

/** 类型间交叉约束与 defaultValue 静态复核（字典 §3 逐控件「min ≤ max」系列）。 */
function validateWidgetCrossRules(
  widget: Record<string, unknown>,
  type: FormWidgetType,
  path: string,
  issues: FormSchemaIssue[],
): void {
  const minMaxOf = (minKey: string, maxKey: string) => {
    const min = widget[minKey];
    const max = widget[maxKey];
    if (isFiniteNumber(min) && isFiniteNumber(max) && min > max) {
      issues.push({
        path: `${path}.${maxKey}`,
        message: `${maxKey} 不能小于 ${minKey}`,
      });
    }
  };
  switch (type) {
    case 'sn':
      if (widget.enable !== false) {
        issues.push({ path: `${path}.enable`, message: '流水号必须不可填写（enable=false）' });
      }
      if (widget.allowBlank !== true) {
        issues.push({ path: `${path}.allowBlank`, message: '流水号由系统生成，必须允许为空' });
      }
      break;
    case 'text':
    case 'textarea':
      minMaxOf('minLength', 'maxLength');
      break;
    case 'number': {
      minMaxOf('min', 'max');
      const min = widget.min;
      const max = widget.max;
      const def = widget.defaultValue;
      // defaultValue 静态复核：必须落在启用中的数值范围内（执行仍延后至 P5，但非法默认值不得入库）。
      if (isFiniteNumber(def)) {
        if (isFiniteNumber(min) && def < min) {
          issues.push({ path: `${path}.defaultValue`, message: 'defaultValue 不能小于 min' });
        }
        if (isFiniteNumber(max) && def > max) {
          issues.push({ path: `${path}.defaultValue`, message: 'defaultValue 不能大于 max' });
        }
      }
      break;
    }
    case 'decimal':
    case 'money':
    case 'percent':
      validateNumericFamilyCrossRules(widget as { type: NumericWidgetType }, path, issues);
      break;
    case 'user':
    case 'usergroup': {
      if (widget.scope === 'department') {
        const deps = widget.departments;
        if (!Array.isArray(deps) || deps.length === 0) {
          issues.push({
            path: `${path}.departments`,
            message: 'scope=department 时 departments 必须是非空数组',
          });
        }
      }
      break;
    }
    case 'subform':
      minMaxOf('minRowCount', 'maxRowCount');
      break;
    default:
      break;
  }
  // 选项类控件的 defaultValue 必须命中选项 value（字符串或字符串数组两形态）。
  if (
    OPTION_WIDGET_TYPES.has(type) &&
    widget.defaultValue !== undefined &&
    widget.defaultValue !== null
  ) {
    const optionValues = new Set<string>(
      Array.isArray(widget.options)
        ? widget.options.filter(isPlainOption).map((option) => option.value as string)
        : [],
    );
    const defaultValue = widget.defaultValue;
    const values = Array.isArray(defaultValue) ? defaultValue : [defaultValue];
    for (const entry of values) {
      if (typeof entry !== 'string' || !optionValues.has(entry)) {
        issues.push({
          path: `${path}.defaultValue`,
          message: 'defaultValue 必须是选项 value 之一',
        });
        break;
      }
    }
  }
}

// ---- 助手 ----

/**
 * 数值字段族交叉约束（与 Go 镜像逐字一致）：
 * 有效 scale ≤ 有效 precision；min ≤ max；defaultValue 落在范围内；
 * min/max/defaultValue 自身须满足 scale/precision 位数约束（防不可满足范围）。
 * 属性级形状已在 validateWidgetProp(kind=decimal) 拒绝，此处防御式读取。
 */
function validateNumericFamilyCrossRules(
  widget: { type: NumericWidgetType },
  path: string,
  issues: FormSchemaIssue[],
): void {
  const record = widget as Record<string, unknown>;
  const declaredInteger = (key: string): number | null => {
    const value = record[key];
    return typeof value === 'number' && Number.isInteger(value) ? value : null;
  };
  const precision = effectiveNumericPrecision({
    type: widget.type,
    precision: declaredInteger('precision'),
  });
  const scale = effectiveNumericScale({ type: widget.type, scale: declaredInteger('scale') });
  if (scale > precision) {
    issues.push({ path: `${path}.scale`, message: 'scale 不能大于 precision' });
  }
  const decimalOf = (key: string): string | null => {
    const value = record[key];
    return typeof value === 'string' && DECIMAL_TEXT_PATTERN.test(value) ? value : null;
  };
  const min = decimalOf('min');
  const max = decimalOf('max');
  const defaultValue = decimalOf('defaultValue');
  const bounds: ReadonlyArray<[string, string | null]> = [
    ['min', min],
    ['max', max],
    ['defaultValue', defaultValue],
  ];
  for (const [key, text] of bounds) {
    if (text === null) continue;
    const issue = decimalDigitIssue(text, precision, scale);
    if (issue === 'scale') {
      issues.push({ path: `${path}.${key}`, message: `${key} 最多支持 ${scale} 位小数` });
    } else if (issue === 'precision') {
      issues.push({
        path: `${path}.${key}`,
        message: `${key} 整数位最多 ${precision - scale} 位`,
      });
    }
  }
  if (min !== null && max !== null && compareDecimalText(min, max) === 1) {
    issues.push({ path: `${path}.max`, message: 'max 不能小于 min' });
  }
  if (defaultValue !== null) {
    if (min !== null && compareDecimalText(defaultValue, min) === -1) {
      issues.push({ path: `${path}.defaultValue`, message: 'defaultValue 不能小于 min' });
    }
    if (max !== null && compareDecimalText(defaultValue, max) === 1) {
      issues.push({ path: `${path}.defaultValue`, message: 'defaultValue 不能大于 max' });
    }
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isPlainOption(value: unknown): value is { label: unknown; value: string } {
  return isPlainObject(value) && typeof value.value === 'string';
}

function isInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value);
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

function inRange(value: number, spec: WidgetPropSpec): boolean {
  if (spec.min !== undefined && value < spec.min) return false;
  if (spec.max !== undefined && value > spec.max) return false;
  return true;
}

function rangeText(spec: WidgetPropSpec): string {
  const min = spec.min ?? '-∞';
  const max = spec.max ?? '+∞';
  return `${min}–${max}`;
}

/** 未知键拒绝：keys 超出白名单的键逐个产出路径错误（不静默丢弃）。 */
function rejectUnknownKeys(
  target: Record<string, unknown>,
  allowed: readonly string[],
  path: string,
  issues: FormSchemaIssue[],
): void {
  for (const key of Object.keys(target)) {
    if (!allowed.includes(key)) {
      issues.push({ path: path === '' ? key : `${path}.${key}`, message: `未知属性「${key}」` });
    }
  }
}
