/**
 * 数值字段族语义（Phase 4，docs/低代码平台/表单设计器/表单数值计算运行时
 * 功能设计-V1.0.md §17/§36/§58-4）。
 *
 * decimal/money/percent 三种控件的属性约束、有效默认与值校验语义在本文件
 * 收口，validate.ts（保存/发布校验）与 codec.ts（提交值终审）共用；
 * 后端 internal/platform/form/service/numeric_field.go 按同一语义镜像，
 * 修改本文件必须同步 Go 镜像与两端测试。
 *
 * 值协议：业务十进制一律 canonical decimal string（设计 §15/§27），
 * 禁止 number 直传（float64 会丢精度）；空值 null。现有 number 控件保持
 * JS number 语义服务简单计数场景，不在本族范围内。
 */

import { RoundingMode, numeric } from '@evolyn.do/numeric';
import type { FormWidgetType } from './types';

/** 数值字段族的三种控件类型。 */
export const NUMERIC_WIDGET_TYPES = ['decimal', 'money', 'percent'] as const;
export type NumericWidgetType = (typeof NUMERIC_WIDGET_TYPES)[number];

/**
 * 精度护栏（与 @evolyn.do/numeric DEFAULT_NUMERIC_CONTEXT 的 precision 40 /
 * maxScale 18 对齐）：precision 是有效数字总位数，scale 是小数位上限。
 */
export const NUMERIC_FIELD_LIMITS = {
  precisionMin: 1,
  precisionMax: 40,
  scaleMin: 0,
  scaleMax: 18,
} as const;

/** 计算链舍入模式枚举（设计 §11；与 @evolyn.do/numeric RoundingMode 逐字一致）。 */
export const NUMERIC_ROUNDING_MODES: readonly string[] = Object.values(RoundingMode);

/**
 * 未显式配置时的有效精度默认（发布期物理列 NUMERIC(p,s) 与值校验共用同一
 * 解析结果，保证两端、草稿与已发布快照口径一致；设计 §17 示例值）。
 */
export const NUMERIC_FIELD_DEFAULTS: Readonly<
  Record<NumericWidgetType, { precision: number; scale: number }>
> = {
  decimal: { precision: 20, scale: 6 },
  money: { precision: 20, scale: 2 },
  percent: { precision: 10, scale: 6 },
};

/**
 * 十进制文本输入形状：可选负号 + 整数位 + 可选小数位。
 * 拒绝指数记法、正号、裸小数点与前导/尾随空白（canonical 协议 §49 无指数）。
 */
export const DECIMAL_TEXT_PATTERN = /^-?\d+(\.\d+)?$/;

export function isNumericWidgetType(type: FormWidgetType | string): type is NumericWidgetType {
  return (NUMERIC_WIDGET_TYPES as readonly string[]).includes(type);
}

/** 数值字段族控件的最小属性视图（min/max/defaultValue 为 decimal string）。 */
interface NumericBoundsSource {
  type: NumericWidgetType;
  precision?: number | null;
  scale?: number | null;
}

/** 有效精度：显式配置优先，否则按控件类型取默认（发布与终审共用）。 */
export function effectiveNumericPrecision(widget: NumericBoundsSource): number {
  if (widget.precision !== null && widget.precision !== undefined) return widget.precision;
  return NUMERIC_FIELD_DEFAULTS[widget.type].precision;
}

/** 有效小数位：显式配置优先，否则按控件类型取默认。 */
export function effectiveNumericScale(widget: NumericBoundsSource): number {
  if (widget.scale !== null && widget.scale !== undefined) return widget.scale;
  return NUMERIC_FIELD_DEFAULTS[widget.type].scale;
}

/**
 * 小数位计数（按值语义）：尾随零不计位（"1.500" 计 1 位），
 * 与后端 numeric.Decimal 的 scale 语义一致。
 */
export function decimalFractionDigits(text: string): number {
  const dot = text.indexOf('.');
  if (dot < 0) return 0;
  let end = text.length;
  while (end > dot + 1 && text[end - 1] === '0') end -= 1;
  return end - dot - 1;
}

/** 整数位计数（按值语义）：前导零不计位（"007" 计 1 位）；零值恒为 0 位。 */
export function decimalIntegerDigits(text: string): number {
  const negative = text.startsWith('-');
  const dot = text.indexOf('.');
  const intText = text.slice(negative ? 1 : 0, dot < 0 ? text.length : dot);
  const trimmed = intText.replace(/^0+(?=\d)/, '');
  return trimmed === '0' ? 0 : trimmed.length;
}

/** 位数约束违规码：'scale' 小数位超限 / 'precision' 整数位超限 / null 通过。 */
export type DecimalDigitIssue = 'scale' | 'precision' | null;

export function decimalDigitIssue(
  text: string,
  precision: number,
  scale: number,
): DecimalDigitIssue {
  if (decimalFractionDigits(text) > scale) return 'scale';
  if (decimalIntegerDigits(text) > precision - scale) return 'precision';
  return null;
}

/**
 * 十进制文本比较（-1/0/1）：走 @evolyn.do/numeric（decimal.js 精确比较，
 * 禁止 parseFloat）。入参已通过形状校验；解析失败按不可比较返回 null，
 * 由调用方决定保守结论。
 */
export function compareDecimalText(a: string, b: string): number | null {
  try {
    return numeric.compare(a, b);
  } catch {
    return null;
  }
}
