/**
 * 目标保存协议严格校验器（P1，字段字典 §1/§2/§3）。
 *
 * 校验目标：任何非法输入都产出 JSON Path 级错误（与后端 Go 校验器逐字节一致），
 * 合法输入产出深拷贝后的规范文档（保证「未编辑属性不丢失」由原样落库承接）。
 * 未知键在任何层级都不被静默容忍——拒绝保存并返回精确路径（方案 §1.1）。
 */

import {
  FORM_PROTOCOL_LIMITS,
  WIDGET_OPTION_LIMITS,
  WIDGET_SPECS,
  type WidgetPropSpec,
} from './dictionary';
import type { FormSchemaDocument, FormWidgetType } from './types';
import { PUBLISHABLE_WIDGET_TYPES, SUBFORM_ALLOWED_WIDGET_TYPES } from './types';
import { cloneFormSchema } from './clone';

/** 单条校验问题：path 为 JSON Path（如 content.items[2].widget.options[0].value）。 */
export interface FormSchemaIssue {
  path: string;
  message: string;
}

export interface FormSchemaValidationResult {
  valid: boolean;
  /** 校验通过时返回深拷贝文档；失败时为 null。 */
  document: FormSchemaDocument | null;
  issues: FormSchemaIssue[];
}

/** widgetName 形状约束（字典 1.4）：标识符形，1–64 字符。 */
const WIDGET_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;

const OPTION_WIDGET_TYPES: ReadonlySet<string> = new Set([
  'radiogroup',
  'checkboxgroup',
  'combo',
  'combocheck',
]);

export function validateFormSchema(input: unknown): FormSchemaValidationResult {
  const issues: FormSchemaIssue[] = [];
  validateRoot(input, issues);
  if (issues.length > 0) return { valid: false, document: null, issues };
  // 形状校验通过后再深拷贝：合法文档只含 JSON 安全值，克隆不会抛错。
  return { valid: true, document: cloneFormSchema(input as FormSchemaDocument), issues: [] };
}

/**
 * 发布校验：在结构校验之上叠加能力白名单（字典 §6）。
 * 白名单外控件返回精确路径错误，交由前端提示与后端 FORM_PUBLISH_UNSUPPORTED_FIELD。
 */
export function validatePublishableFormSchema(input: unknown): FormSchemaValidationResult {
  const base = validateFormSchema(input);
  if (!base.valid || !base.document) return base;
  const issues: FormSchemaIssue[] = [];
  collectUnsupportedWidgets(base.document, 'content.items', issues);
  if (issues.length > 0) return { valid: false, document: null, issues };
  return base;
}

function collectUnsupportedWidgets(
  document: FormSchemaDocument,
  itemsPath: string,
  issues: FormSchemaIssue[],
): void {
  document.content.items.forEach((item, index) => {
    const path = `${itemsPath}[${index}].widget.type`;
    if (!PUBLISHABLE_WIDGET_TYPES.includes(item.widget.type)) {
      issues.push({
        path,
        message: `控件「${item.widget.type}」的运行能力尚未开放，暂不能发布`,
      });
    }
    if (item.widget.type === 'subform') {
      collectUnsupportedWidgets(item.widget, `${itemsPath}[${index}].widget.items`, issues);
    }
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
  rejectUnknownKeys(content, ['type', 'items'], 'content', issues);
  if (content.type !== 'form') {
    issues.push({ path: 'content.type', message: 'content.type 必须固定为 "form"' });
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
  content.items.forEach((item, index) => {
    validateItem(item, `content.items[${index}]`, issues, seenNames);
  });
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
  const spec = (WIDGET_SPECS as Record<string, (typeof WIDGET_SPECS)[FormWidgetType] | undefined>)[
    type
  ];

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
  }
  validateWidget(input.widget, `${path}.widget`, issues, seenNames);
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
    validateWidgetProp(input[key], propSpec, `${path}.${key}`, issues);
  }

  // 类型间交叉约束（字典逐条对应的 min≤max 系列）。
  validateWidgetCrossRules(input as Record<string, unknown>, type as FormWidgetType, path, issues);
}

function validateWidgetProp(
  value: unknown,
  spec: WidgetPropSpec,
  path: string,
  issues: FormSchemaIssue[],
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
      validateSubformItems(value, path, issues);
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
    case 'snRule':
      validateSnRule(value, path, issues);
      return;
    case 'buttonAction':
      validateButtonAction(value, path, issues);
      return;
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

function validateSubformItems(value: unknown, path: string, issues: FormSchemaIssue[]): void {
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
    validateItem(child, childPath, issues, scopeNames);
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

function validateSnRule(value: unknown, path: string, issues: FormSchemaIssue[]): void {
  if (!isPlainObject(value)) {
    issues.push({ path, message: 'rule 必须是对象' });
    return;
  }
  rejectUnknownKeys(value, ['prefix', 'dateFmt', 'seqLength', 'resetCycle'], path, issues);
  if (
    value.prefix !== undefined &&
    (typeof value.prefix !== 'string' || value.prefix.length > 32)
  ) {
    issues.push({ path: `${path}.prefix`, message: '流水号前缀必须是 ≤32 字符的字符串' });
  }
  const dateFmts = ['none', 'yyyyMM', 'yyyyMMdd'];
  if (
    value.dateFmt !== undefined &&
    (typeof value.dateFmt !== 'string' || !dateFmts.includes(value.dateFmt))
  ) {
    issues.push({
      path: `${path}.dateFmt`,
      message: `dateFmt 必须是以下枚举值之一：${dateFmts.join(' / ')}`,
    });
  }
  if (
    value.seqLength !== undefined &&
    (!isInteger(value.seqLength) || value.seqLength < 3 || value.seqLength > 8)
  ) {
    issues.push({ path: `${path}.seqLength`, message: 'seqLength 必须是 3–8 的整数' });
  }
  const cycles = ['none', 'daily', 'monthly', 'yearly'];
  if (
    value.resetCycle !== undefined &&
    (typeof value.resetCycle !== 'string' || !cycles.includes(value.resetCycle))
  ) {
    issues.push({
      path: `${path}.resetCycle`,
      message: `resetCycle 必须是以下枚举值之一：${cycles.join(' / ')}`,
    });
  }
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

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isPlainOption(value: unknown): value is { label: unknown; value: unknown } {
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
