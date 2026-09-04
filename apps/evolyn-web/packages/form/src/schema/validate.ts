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
  SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES,
  SUBMIT_RULE_LIMITS,
  SUBMIT_RULE_RECOMPUTE_SUPPORTED,
  WIDGET_OPTION_LIMITS,
  WIDGET_SPECS,
  type WidgetPropSpec,
} from './dictionary';
import { isCanonicalDateTime } from './codec';
import {
  type FormSchemaDocument,
  type FormItem,
  type FormWidgetType,
  PUBLISHABLE_WIDGET_TYPES,
  SUBFORM_ALLOWED_WIDGET_TYPES,
} from './types';
import { cloneFormSchema } from './clone';
import {
  createValidationResult,
  type ValidationDiagnostic,
} from '@evolyn.do/validator';

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
        if (type !== undefined && !PUBLISHABLE_WIDGET_TYPES.includes(type)) {
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
      // 子表单的字段数组位于 widget.items，而不是文档根 content.items。
      // 递归直接传入该数组，避免发布校验将子表单控件误作完整表单文档读取。
      collectUnsupportedWidgets(item.widget.items, `${itemsPath}[${index}].widget.items`, issues);
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
  content.items.forEach((item, index) => {
    validateItem(item, `content.items[${index}]`, issues, seenNames);
  });
  validateLayouts(content, seenNames, issues);
  validateFieldShowRules(content, issues);
  validateSubmitRules(content, issues);
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
    case 'snRule':
      validateSnRule(value, path, issues);
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
