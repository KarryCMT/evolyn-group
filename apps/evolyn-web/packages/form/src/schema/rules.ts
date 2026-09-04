/**
 * 字段显隐规则编译器与求值器（v5，docs/低代码平台/表单设计器/字段显隐规则设计方案.md）。
 *
 * 纯 TypeScript、无副作用：设计期校验（validate.ts）负责结构与依赖图合法性，
 * 本模块只消费已通过校验（或防御性跳过非法片段）的规则，产出编译结果供
 * 桌面/移动运行时与提交终审共用。后端 Go 求值器（internal/platform/form 的
 * rules.go）按同一语义镜像，两侧对同一组 fixture 的求值结论必须一致。
 *
 * 求值安全语义（设计方案 §4.2）：
 * - 条件源不可见（静态隐藏、权限隐藏或被上游规则隐藏）时该条件视为「不满足」，
 *   不读取其隐藏值，避免通过下游字段反推出无权字段的值；
 * - 上游隐藏时其全部下游自然隐藏（按拓扑序传播）；
 * - 除 isEmpty/notEmpty 外，空值一律令条件不成立（杜绝「未填写即满足 ne/notIn」）。
 */

import { FIELD_SHOW_EMPTY_METHODS } from './dictionary';
import { isEmptyWidgetValue } from './codec';
import type { FieldShowCondition, FieldShowRule, FormContent, FormJsonValue } from './types';
import {
  compileRuleGraph,
  downstreamRuleTargets,
  evaluateRuleGraph,
  isEmptyCompiledRuleGraph,
  type CompiledRuleGraph,
  type RuleGraphRule,
} from '@evolyn.do/rule';

/** 编译后的单条规则：协议字段的窄化视图。 */
export type CompiledFieldShowRule = RuleGraphRule<FieldShowCondition>;
export type CompiledFieldShowRules = CompiledRuleGraph<FieldShowCondition>;

/** 求值输入：字段值读取与基础可达性（静态 visible ∧ 权限可见）由调用方注入。 */
export interface FieldShowEvaluationContext {
  /** 读取字段当前值；不可见/无值字段返回 undefined 或空值均可。 */
  valueOf: (field: string) => unknown;
  /** 字段基础可达性（不含规则本身）；不可达的条件源条件视为不成立。 */
  isBaseVisible: (field: string) => boolean;
  /** 当前登录成员 ID；未提供（匿名/未注入）时不参与 includeCurrentMember。 */
  currentMemberId?: string;
}

/** 从表单内容编译显隐规则；空/畸形规则集返回空编译结果（不抛错）。 */
export function compileFieldShowRules(content: FormContent): CompiledFieldShowRules {
  const rawRules = Array.isArray(content.fieldShowRules) ? content.fieldShowRules : [];
  return compileRuleGraph(rawRules.map(toRuleGraphRule).filter(isDefined));
}

/** 规则是否为空编译（无任何目标字段）：运行时可据此整体跳过求值。 */
export function isEmptyCompiledRules(compiled: CompiledFieldShowRules): boolean {
  return isEmptyCompiledRuleGraph(compiled);
}

/**
 * 全量求值：按拓扑序计算每个目标字段的「规则可见性」（不含静态 visible 与
 * 权限，合成由调用方完成）。返回 Map 仅包含规则目标字段。
 */
export function evaluateFieldShowRules(
  compiled: CompiledFieldShowRules,
  context: FieldShowEvaluationContext,
): Map<string, boolean> {
  return evaluateRuleGraph(compiled, {
    isBaseVisible: context.isBaseVisible,
    matchRule: (rule, visibility) =>
      matchFieldShowRule(rule, {
        valueOf: context.valueOf,
        isFieldVisible: visibility.isFieldVisible,
        currentMemberId: context.currentMemberId,
      }),
  });
}

/**
 * 定向重算闭包：自发生变化的字段出发，沿依赖图收集全部受影响的下游目标
 * （含多级），按拓扑序返回。单字段修改只重算该闭包，不做全量规则扫描
 * （设计方案 §6.1 / 验收 §8.1 的性能约定）。
 */
export function downstreamTargets(
  compiled: CompiledFieldShowRules,
  changedField: string,
): readonly string[] {
  return downstreamRuleTargets(compiled, changedField);
}

/**
 * 单规则匹配（增量重算复用）：isFieldVisible 返回条件源的当前有效可见性
 * （静态 ∧ 权限 ∧ 已计算的规则可见性），不可见条件一律不成立。
 */
export function matchFieldShowRule(
  rule: CompiledFieldShowRule,
  context: Omit<FieldShowEvaluationContext, 'isBaseVisible'> & {
    isFieldVisible: (field: string) => boolean;
  },
): boolean {
  if (rule.conditions.length === 0) return false;
  const results = rule.conditions.map((condition) => {
    if (!context.isFieldVisible(condition.field)) return false;
    return matchFieldShowCondition(
      condition,
      context.valueOf(condition.field),
      context.currentMemberId,
    );
  });
  return rule.rel === 'and' ? results.every(Boolean) : results.some(Boolean);
}

/**
 * 单条件求值：值类型不匹配按「条件不成立」处理（防御式，正常输入经
 * normalizeWidgetValue 收敛不会触达）。比较集合为 value ∪ {当前成员}。
 */
export function matchFieldShowCondition(
  condition: FieldShowCondition,
  rawValue: unknown,
  currentMemberId?: string,
): boolean {
  const empty = isEmptyWidgetValue(rawValue);
  if (condition.method === 'isEmpty') return empty;
  if (condition.method === 'notEmpty') return !empty;
  if (empty) return false;

  const expected = expectedValues(condition, currentMemberId);
  switch (condition.method) {
    case 'eq':
    case 'in':
      return scalarIncludes(condition.type, rawValue, expected);
    case 'ne':
    case 'notIn':
      return !scalarIncludes(condition.type, rawValue, expected);
    case 'contains':
      return readText(rawValue) !== null && readText(rawValue)!.includes(textOf(expected[0]));
    case 'notContains':
      return readText(rawValue) === null
        ? false
        : !readText(rawValue)!.includes(textOf(expected[0]));
    case 'gt':
      return compareOrdered(condition.type, rawValue, expected[0]) > 0;
    case 'gte':
      return compareOrdered(condition.type, rawValue, expected[0]) >= 0;
    case 'lt':
      return compareOrdered(condition.type, rawValue, expected[0]) < 0;
    case 'lte':
      return compareOrdered(condition.type, rawValue, expected[0]) <= 0;
    case 'between': {
      if (expected.length < 2) return false;
      const lower = compareOrdered(condition.type, rawValue, expected[0]);
      const upper = compareOrdered(condition.type, rawValue, expected[1]);
      return lower >= 0 && upper <= 0;
    }
    case 'containsAny':
    case 'containsAll':
    case 'containsNone': {
      const selected = readStringArray(rawValue);
      if (selected === null) return false;
      const expectedSet = new Set(expected.map(textOf));
      const hits = selected.filter((entry) => expectedSet.has(entry)).length;
      if (condition.method === 'containsAny') return hits > 0;
      if (condition.method === 'containsAll') return hits === expectedSet.size;
      return hits === 0;
    }
    default:
      return false;
  }
}

// ---- 编译助手 ----

/** 防御式编译单条规则：结构不完整即跳过（合法性由设计期校验单独保证）。 */
function toRuleGraphRule(raw: FieldShowRule): CompiledFieldShowRule | undefined {
  if (!raw || typeof raw !== 'object') return undefined;
  const rel = raw.filter?.rel === 'or' ? 'or' : 'and';
  const conditions = Array.isArray(raw.filter?.cond)
    ? raw.filter.cond.filter(
        (condition): condition is FieldShowCondition =>
          Boolean(condition) &&
          typeof condition === 'object' &&
          typeof condition.field === 'string' &&
          condition.field !== '' &&
          typeof condition.method === 'string',
      )
    : [];
  const targets = Array.isArray(raw.fields)
    ? raw.fields.filter((field): field is string => typeof field === 'string' && field !== '')
    : [];
  if (conditions.length === 0 || targets.length === 0) return undefined;
  if (typeof raw.id !== 'string' || raw.id === '') return undefined;
  return { id: raw.id, rel, conditions, targets };
}

function isDefined<T>(value: T | undefined): value is T {
  return value !== undefined;
}

// ---- 条件求值助手 ----

/** 组装比较集合：value 常量 + includeCurrentMember 注入的当前成员。 */
function expectedValues(condition: FieldShowCondition, currentMemberId?: string): FormJsonValue[] {
  const values = Array.isArray(condition.value) ? condition.value.filter(isComparableValue) : [];
  const merged = [...values];
  if (condition.includeCurrentMember && currentMemberId && !merged.includes(currentMemberId)) {
    merged.push(currentMemberId);
  }
  return merged;
}

/** 参与比较的常量形状：字符串或有限数值。 */
function isComparableValue(value: unknown): value is FormJsonValue {
  if (typeof value === 'string') return value !== '';
  return typeof value === 'number' && Number.isFinite(value);
}

/** 单值语义控件的集合命中判断（eq/ne 与 in/notIn 在实现上同构）。 */
function scalarIncludes(
  type: string,
  rawValue: unknown,
  expected: readonly FormJsonValue[],
): boolean {
  if (type === 'number') {
    const num = readNumber(rawValue);
    return num !== null && expected.some((entry) => typeof entry === 'number' && entry === num);
  }
  const text = readText(rawValue);
  return text !== null && expected.some((entry) => textOf(entry) === text);
}

/** 有序比较：number 按数值、datetime 按规范形状字符串字典序（同格式可比）。 */
function compareOrdered(
  type: string,
  rawValue: unknown,
  expected: FormJsonValue | undefined,
): number {
  if (expected === undefined) return 0;
  if (type === 'number') {
    const left = readNumber(rawValue);
    const right = readNumber(expected);
    if (left === null || right === null) return NaN;
    return left - right;
  }
  const left = readText(rawValue);
  const right = textOf(expected);
  if (left === null) return NaN;
  return left < right ? -1 : left > right ? 1 : 0;
}

function readText(value: unknown): string | null {
  return typeof value === 'string' ? value : null;
}

function readNumber(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

function readStringArray(value: unknown): string[] | null {
  if (!Array.isArray(value)) return null;
  return value.filter((entry): entry is string => typeof entry === 'string');
}

function textOf(value: FormJsonValue | undefined): string {
  return typeof value === 'string' ? value : value === undefined ? '' : String(value);
}

/** 空方法判定复用字典集合（isEmpty/notEmpty 不携带 value）。 */
export function isEmptyShowMethod(method: string): boolean {
  return FIELD_SHOW_EMPTY_METHODS.has(method);
}
