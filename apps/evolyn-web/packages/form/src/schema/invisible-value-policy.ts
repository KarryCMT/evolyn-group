/**
 * 不可见字段赋值策略解析与客户端预演（v6，
 * docs/低代码平台/表单设计器/不可见字段赋值前后端设计方案.md）。
 *
 * 无 UI 副作用的共享模块：设计器（选择框/对话框/摘要卡）与运行时（提交信封
 * 预演、策略调试预览）都经此消费协议，禁止在组件内分散写策略判断。值决议的
 * 权威在服务端（发布快照 + 锁定基线 + 派生执行器），本模块只做协议解析与
 * 展示层预演；后端 Go 侧 invisible_value_policy.go 按同一语义镜像。
 */

import { emptyWidgetValue, isLayoutWidgetType } from './codec';
import {
  DEFAULT_SUBMIT_RULE,
  SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES,
  SUBMIT_RULE_LABELS,
  SUBMIT_RULE_RECOMPUTE_SUPPORTED,
} from './dictionary';
import type { FormContent, FormJsonValue, FormWidgetType, SubmitRule } from './types';

/** 策略解析的防御式只读视图：非法片段回退默认值，不向上抛错。 */
export interface InvisibleValuePolicyView {
  /** 表单默认策略（submitRule）。 */
  submitRule: SubmitRule;
  /** 字段级例外映射（widget_submit_rules 原样窄化）。 */
  specialRules: Readonly<Record<string, SubmitRule>>;
}

/**
 * 解析 content 的赋值策略配置：合法文档由 validate.ts 保证形状；历史快照或
 * 外部数据缺键时回退默认「空值」，保证读取侧永不因策略缺键失败。
 */
export function readInvisibleValuePolicy(content: FormContent): InvisibleValuePolicyView {
  const raw = content as unknown as Record<string, unknown>;
  const submitRule = isSubmitRule(raw.submitRule) ? raw.submitRule : DEFAULT_SUBMIT_RULE;
  const specialRules: Record<string, SubmitRule> = {};
  const rawRules = raw.widget_submit_rules;
  if (isPlainRecord(rawRules)) {
    for (const [key, value] of Object.entries(rawRules)) {
      if (isSubmitRule(value)) specialRules[key] = value;
    }
  }
  return { submitRule, specialRules };
}

/** 策略解析（设计方案 §3.3）：特殊规则优先于默认策略。 */
export function resolveSubmitStrategy(
  policy: InvisibleValuePolicyView,
  widgetName: string,
): SubmitRule {
  return policy.specialRules[widgetName] ?? policy.submitRule;
}

/** 顶层控件是否可配置赋值策略：具备普通用户提交值语义（§3.2）。 */
export function isSubmitRuleEligibleType(type: FormWidgetType | string): boolean {
  return (SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES as readonly string[]).includes(type);
}

/** 字段是否参与赋值策略决议：非布局控件即可处理（布局节点不进入值表）。 */
export function isSubmitRuleProcessableType(type: FormWidgetType | string): boolean {
  return !isLayoutWidgetType(type);
}

/** 策略展示名（保持原值 / 空值 / 始终重新计算）。 */
export function submitRuleLabel(rule: SubmitRule | number): string {
  return SUBMIT_RULE_LABELS[rule] ?? String(rule);
}

/** 「始终重新计算」是否可配置（派生执行器交付前恒为 false，P3 解除）。 */
export function isRecomputeConfigurable(): boolean {
  return SUBMIT_RULE_RECOMPUTE_SUPPORTED;
}

/**
 * 归一化特殊规则映射：丢弃与默认策略相同的冗余项（§3.1——设计器切换默认
 * 策略后自动移除对应冗余映射，校验器对残留冗余拒绝保存）。
 */
export function normalizeWidgetSubmitRules(
  rules: Readonly<Record<string, SubmitRule>>,
  submitRule: SubmitRule,
): Record<string, SubmitRule> {
  const normalized: Record<string, SubmitRule> = {};
  for (const [key, value] of Object.entries(rules)) {
    if (isSubmitRule(value) && value !== submitRule) normalized[key] = value;
  }
  return normalized;
}

/**
 * 客户端值预演（§5.2）：预测服务端对不可见字段的写入结果。仅用于展示与
 * 调试预览——clear 得到类型空值；preserve 在无基线的新建场景同样为空值；
 * recompute 的结果只能由服务端执行器产出，客户端一律不可预演。
 */
export function previewInvisibleValue(
  strategy: SubmitRule,
  widgetType: FormWidgetType,
  baseline: FormJsonValue | undefined,
): FormJsonValue | null {
  if (strategy === 1) {
    return baseline === undefined
      ? (emptyWidgetValue(widgetType) as FormJsonValue | null)
      : baseline;
  }
  return emptyWidgetValue(widgetType) as FormJsonValue | null;
}

function isSubmitRule(value: unknown): value is SubmitRule {
  return typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 3;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
