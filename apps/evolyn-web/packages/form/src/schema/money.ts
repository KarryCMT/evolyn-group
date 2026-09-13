/**
 * 金额字段的币种元数据与展示转换。
 *
 * 金额本体仍按 canonical decimal string 存储，币种仅属于字段 Schema；展示层
 * 只在失焦、确认文案和记录列表加上符号、分组与定长小数，绝不参与公式或查询。
 */
import { numeric } from '@evolyn.do/numeric';
import { DECIMAL_TEXT_PATTERN, effectiveNumericScale, type NumericWidgetType } from './numeric';

export const MONEY_CURRENCY_CODES = [
  'CNY',
  'USD',
  'EUR',
  'GBP',
  'JPY',
  'HKD',
  'KRW',
  'SGD',
  'AUD',
  'CAD',
  'CHF',
  'AED',
] as const;

export type MoneyCurrencyCode = (typeof MONEY_CURRENCY_CODES)[number];

interface MoneyCurrencyMeta {
  label: string;
  symbol: string;
  /** 切换币种时采用的推荐存储与展示小数位，设计者仍可在面板中自行调整。 */
  fractionDigits: number;
}

interface MoneyDisplayWidget {
  type: NumericWidgetType;
  currencyCode?: unknown;
  precision?: number | null;
  scale?: number | null;
}

export const DEFAULT_MONEY_CURRENCY: MoneyCurrencyCode = 'CNY';

export const MONEY_CURRENCIES: Readonly<Record<MoneyCurrencyCode, MoneyCurrencyMeta>> = {
  CNY: { label: '人民币（CNY）', symbol: '¥', fractionDigits: 2 },
  USD: { label: '美元（USD）', symbol: '$', fractionDigits: 2 },
  EUR: { label: '欧元（EUR）', symbol: '€', fractionDigits: 2 },
  GBP: { label: '英镑（GBP）', symbol: '£', fractionDigits: 2 },
  JPY: { label: '日元（JPY）', symbol: '￥', fractionDigits: 0 },
  HKD: { label: '港元（HKD）', symbol: 'HK$', fractionDigits: 2 },
  KRW: { label: '韩元（KRW）', symbol: '₩', fractionDigits: 0 },
  SGD: { label: '新加坡元（SGD）', symbol: 'S$', fractionDigits: 2 },
  AUD: { label: '澳元（AUD）', symbol: 'A$', fractionDigits: 2 },
  CAD: { label: '加元（CAD）', symbol: 'CA$', fractionDigits: 2 },
  CHF: { label: '瑞士法郎（CHF）', symbol: 'CHF', fractionDigits: 2 },
  AED: { label: '阿联酋迪拉姆（AED）', symbol: 'AED', fractionDigits: 2 },
};

export const MONEY_CURRENCY_OPTIONS = MONEY_CURRENCY_CODES.map((value) => ({
  value,
  label: MONEY_CURRENCIES[value].label,
}));

/** 旧金额快照未声明币种时回退为 CNY，确保数据值不迁移也有稳定展示。 */
export function resolveMoneyCurrencyCode(widget: unknown): MoneyCurrencyCode {
  const code =
    typeof widget === 'object' && widget !== null && 'currencyCode' in widget
      ? (widget as { currencyCode?: unknown }).currencyCode
      : undefined;
  return typeof code === 'string' && MONEY_CURRENCY_CODES.includes(code as MoneyCurrencyCode)
    ? (code as MoneyCurrencyCode)
    : DEFAULT_MONEY_CURRENCY;
}

export function moneyCurrencySymbol(widget: unknown): string {
  return MONEY_CURRENCIES[resolveMoneyCurrencyCode(widget)].symbol;
}

export function moneyCurrencyFractionDigits(code: MoneyCurrencyCode): number {
  return MONEY_CURRENCIES[code].fractionDigits;
}

/** 仅格式化数值部分，适用于输入框中固定的币种前缀。 */
export function formatMoneyInputValue(
  value: string | null | undefined,
  widget: MoneyDisplayWidget,
): string {
  if (!value || !DECIMAL_TEXT_PATTERN.test(value)) return value ?? '';
  try {
    const numericValue = numeric.of(value);
    const canonical = numericValue.serialize();
    if (canonical === null) return value;
    const [, sourceFraction = ''] = canonical.split('.');
    // 尚未通过字段 scale 校验的输入不能为了展示被静默舍入，保留原文给校验器提示。
    if (sourceFraction.length > effectiveNumericScale(widget)) return value;
    const fixed = numericValue.toFixed(effectiveNumericScale(widget));
    if (fixed === null) return value;
    const negative = fixed.startsWith('-');
    const absolute = negative ? fixed.slice(1) : fixed;
    const [integer = '', fraction] = absolute.split('.');
    const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    return `${negative ? '-' : ''}${grouped}${fraction === undefined ? '' : `.${fraction}`}`;
  } catch {
    // 编辑中的草稿可能暂未满足数值护栏；原文保留给既有校验器提示。
    return value;
  }
}

/** 完整金额展示，适用于确认文案、记录列表及失焦的只读视图。 */
export function formatMoneyValue(
  value: string | null | undefined,
  widget: MoneyDisplayWidget,
): string {
  if (value && DECIMAL_TEXT_PATTERN.test(value)) {
    try {
      const canonical = numeric.of(value).serialize();
      const [, sourceFraction = ''] = (canonical ?? '').split('.');
      if (sourceFraction.length > effectiveNumericScale(widget)) return value;
    } catch {
      return value;
    }
  }
  const formatted = formatMoneyInputValue(value, widget);
  if (!formatted || !DECIMAL_TEXT_PATTERN.test(value ?? '')) return formatted;
  const negative = formatted.startsWith('-');
  return `${negative ? '-' : ''}${moneyCurrencySymbol(widget)}${negative ? formatted.slice(1) : formatted}`;
}

/** 输入可粘贴带货币符号或千分位的金额，回写前剥离纯展示字符。 */
export function parseMoneyInput(value: string, widget: unknown): string {
  const currency = resolveMoneyCurrencyCode(widget);
  const symbol = MONEY_CURRENCIES[currency].symbol;
  return value
    .trim()
    .replace(/,/g, '')
    .replace(new RegExp(escapeRegExp(symbol), 'g'), '')
    .replace(new RegExp(currency, 'g'), '')
    .trim();
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
