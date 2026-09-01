/**
 * 字段值编解码与校验（字段字典 §4）。
 *
 * 浏览器与后端（internal/platform/form 的 Go 值校验器）共享同一套语义与错误文案，
 * 浏览器校验只改善交互，服务端复核才是最终裁决（方案 §3.7）。
 * 当前已实现基础字段与成员单选/多选；其余控件在发布白名单开放前不参与值校验。
 */

import type {
  FormItem,
  FormItemWidget,
  FormJsonValue,
  FormWidgetOption,
  FormWidgetType,
} from './types';

/** 无值控件（不进入值表、不校验、不提交）。 */
const LAYOUT_WIDGET_TYPES: ReadonlySet<string> = new Set(['separator', 'button']);

/** 多值控件（空值为空数组而非 null）。 */
const MULTI_VALUE_WIDGET_TYPES: ReadonlySet<string> = new Set([
  'checkboxgroup',
  'combocheck',
  'usergroup',
  'deptgroup',
  'linkquery',
]);

/** 选择语义控件（空值提示用「请选择」，其余用「请输入」；与后端文案一致）。 */
const CHOOSING_WIDGET_TYPES: ReadonlySet<string> = new Set([
  'datetime',
  'radiogroup',
  'checkboxgroup',
  'combo',
  'combocheck',
  'user',
  'usergroup',
]);

const MAX_MEMBER_SELECTION_COUNT = 200;

export function isLayoutWidgetType(type: string): boolean {
  return LAYOUT_WIDGET_TYPES.has(type);
}

export function isMultiValueWidgetType(type: string): boolean {
  return MULTI_VALUE_WIDGET_TYPES.has(type);
}

/** 空值判定：null/undefined/空串/空数组均视为未填写（与运行时历史口径一致）。 */
export function isEmptyWidgetValue(value: unknown): boolean {
  if (value === null || value === undefined || value === '') return true;
  return Array.isArray(value) && value.length === 0;
}

/** 类型化的空值：多选为空数组，其余为 null。 */
export function emptyWidgetValue(type: FormWidgetType): FormJsonValue {
  return isMultiValueWidgetType(type) ? [] : null;
}

// ---- 防御式配置读取（运行时消费外部数据，禁止裸取值参与计算） ----

export function readWidgetStringConfig(widget: FormItemWidget, key: string): string | undefined {
  const value = (widget as unknown as Record<string, unknown>)[key];
  return typeof value === 'string' ? value : undefined;
}

export function readWidgetNumberConfig(widget: FormItemWidget, key: string): number | undefined {
  const value = (widget as unknown as Record<string, unknown>)[key];
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

export function readWidgetBooleanConfig(widget: FormItemWidget, key: string): boolean | undefined {
  const value = (widget as unknown as Record<string, unknown>)[key];
  return typeof value === 'boolean' ? value : undefined;
}

export function readWidgetOptions(widget: FormItemWidget): FormWidgetOption[] {
  const value = (widget as unknown as Record<string, unknown>).options;
  if (!Array.isArray(value)) return [];
  return value.filter(
    (option): option is FormWidgetOption =>
      typeof option === 'object' &&
      option !== null &&
      !Array.isArray(option) &&
      typeof (option as FormWidgetOption).label === 'string' &&
      typeof (option as FormWidgetOption).value === 'string',
  );
}

// ---- 值归一化 ----

/**
 * 按控件类型归一化值，防止脏数据（历史版本、外部写入）进入值表：
 * 单选文本类收敛 string|null、数字收敛有限 number|null、成员及其他多选收敛 string[]；
 * 未开放运行能力的控件保留 JSON 安全原值，由后续阶段的专属校验接管。
 */
export function normalizeWidgetValue(widget: FormItemWidget, value: unknown): FormJsonValue {
  switch (widget.type) {
    case 'text':
    case 'textarea':
    case 'datetime':
    case 'radiogroup':
    case 'combo':
    case 'user':
      return typeof value === 'string' ? value : null;
    case 'number':
      if (typeof value === 'number' && Number.isFinite(value)) return value;
      if (typeof value === 'string' && value.trim() !== '' && Number.isFinite(Number(value))) {
        return Number(value);
      }
      return null;
    case 'checkboxgroup':
    case 'combocheck':
    case 'usergroup':
      if (!Array.isArray(value)) return [];
      return value.filter((entry): entry is string => typeof entry === 'string');
    default:
      return isLooseJsonValue(value) ? value : null;
  }
}

// ---- 值校验（错误文案与后端逐字一致） ----

/**
 * 字段级值校验：类型、长度、格式、范围与选项命中。
 * 跨字段约束、权限与服务端计算不在这一层（方案 §3.7）。
 */
export function validateWidgetValue(item: FormItem, value: unknown): string[] {
  const widget = item.widget;
  if (widget.type === 'separator' || widget.type === 'button') return [];
  if (isEmptyWidgetValue(value)) {
    return widget.allowBlank ? [] : [`请${choosingVerb(widget.type)}${item.label}`];
  }
  switch (widget.type) {
    case 'text':
      return validateTextValue(item.label, widget, value);
    case 'textarea':
      return validateTextValue(item.label, widget, value);
    case 'number':
      return validateNumberValue(item.label, widget, value);
    case 'datetime':
      return validateDateTimeValue(item.label, widget, value);
    case 'radiogroup':
    case 'combo':
      return validateSingleOptionValue(item.label, widget, value);
    case 'checkboxgroup':
    case 'combocheck':
      return validateMultiOptionValue(item.label, widget, value);
    case 'user':
      return validateMemberValue(item.label, value);
    case 'usergroup':
      return validateMemberGroupValue(item.label, value);
    default:
      // 未开放运行能力的控件：结构校验在保存/发布侧执行，值校验随各阶段落地。
      return [];
  }
}

/** 成员 ID 由运行时选择器产出；值层只约束稳定的字符串标识形状。 */
function validateMemberValue(label: string, value: unknown): string[] {
  return typeof value === 'string' && value.trim() !== '' ? [] : [`${label}的值类型不正确`];
}

/** 多成员字段保留选择顺序，但不允许空标识、重复成员或超出协议上限。 */
function validateMemberGroupValue(label: string, value: unknown): string[] {
  if (!Array.isArray(value)) return [`${label}的值类型不正确`];
  if (value.length > MAX_MEMBER_SELECTION_COUNT)
    return [`${label}最多选择 ${MAX_MEMBER_SELECTION_COUNT} 名成员`];
  const selected = new Set<string>();
  for (const memberID of value) {
    if (typeof memberID !== 'string' || memberID.trim() === '') return [`${label}的值类型不正确`];
    if (selected.has(memberID)) return [`${label}的值存在重复成员`];
    selected.add(memberID);
  }
  return [];
}

function choosingVerb(type: string): string {
  return CHOOSING_WIDGET_TYPES.has(type) ? '选择' : '输入';
}

function validateTextValue(
  label: string,
  widget: Extract<FormItemWidget, { type: 'text' | 'textarea' }>,
  value: unknown,
): string[] {
  if (typeof value !== 'string') return [`${label}的值类型不正确`];
  const errors: string[] = [];
  const minLength = widget.minLength ?? undefined;
  const maxLength = widget.maxLength ?? undefined;
  if (minLength !== undefined && minLength !== null && value.length < minLength) {
    errors.push(`${label}最少输入 ${minLength} 个字符`);
  }
  if (maxLength !== undefined && maxLength !== null && value.length > maxLength) {
    errors.push(`${label}不能超过 ${maxLength} 个字符`);
  }
  if (widget.format === 'email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    errors.push(`${label}格式不正确`);
  }
  return errors;
}

function validateNumberValue(
  label: string,
  widget: Extract<FormItemWidget, { type: 'number' }>,
  value: unknown,
): string[] {
  if (typeof value !== 'number' || !Number.isFinite(value)) return [`${label}的值类型不正确`];
  const errors: string[] = [];
  if (widget.min !== null && widget.min !== undefined && value < widget.min) {
    errors.push(`${label}不能小于 ${widget.min}`);
  }
  if (widget.max !== null && widget.max !== undefined && value > widget.max) {
    errors.push(`${label}不能大于 ${widget.max}`);
  }
  const precision = widget.precision ?? undefined;
  if (precision !== undefined && precision !== null) {
    const scaled = value * 10 ** precision;
    if (Math.abs(scaled - Math.round(scaled)) > 1e-9) {
      errors.push(`${label}最多支持 ${precision} 位小数`);
    }
  }
  return errors;
}

function validateDateTimeValue(
  label: string,
  widget: Extract<FormItemWidget, { type: 'datetime' }>,
  value: unknown,
): string[] {
  if (typeof value !== 'string') return [`${label}的值类型不正确`];
  return isCanonicalDateTime(value, widget.format ?? 'datetime')
    ? []
    : [`${label}的日期格式不正确`];
}

function validateSingleOptionValue(
  label: string,
  widget: Extract<FormItemWidget, { type: 'radiogroup' | 'combo' }>,
  value: unknown,
): string[] {
  if (typeof value !== 'string') return [`${label}的值类型不正确`];
  return readWidgetOptions(widget).some((option) => option.value === value)
    ? []
    : [`${label}的值不在选项范围内`];
}

function validateMultiOptionValue(
  label: string,
  widget: Extract<FormItemWidget, { type: 'checkboxgroup' | 'combocheck' }>,
  value: unknown,
): string[] {
  if (!Array.isArray(value)) return [`${label}的值类型不正确`];
  const optionValues = new Set(readWidgetOptions(widget).map((option) => option.value));
  if (value.some((entry) => typeof entry !== 'string' || !optionValues.has(entry))) {
    return [`${label}的值不在选项范围内`];
  }
  if (new Set(value).size !== value.length) {
    return [`${label}的值存在重复选项`];
  }
  return [];
}

// ---- 日期时间形状校验（按字典 §3 datetime 的规范形状，真实日历校验） ----

/**
 * 校验值是否符合所选 format 的规范形状且为真实日期时间：
 * date→YYYY-MM-DD、datetime→YYYY-MM-DD HH:mm:ss、month→YYYY-MM、time→HH:mm。
 * 禁止依赖 Date.parse（各引擎对非 ISO 形状与溢出值的解析行为不一致）。
 */
export function isCanonicalDateTime(
  value: string,
  format: 'date' | 'datetime' | 'month' | 'time',
): boolean {
  switch (format) {
    case 'date':
      return matchDateTimeShape(value, /^(\d{4})-(\d{2})-(\d{2})$/, (m) =>
        isRealDate(Number(m[1]), Number(m[2]), Number(m[3])),
      );
    case 'datetime':
      return matchDateTimeShape(
        value,
        /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/,
        (m) =>
          isRealDate(Number(m[1]), Number(m[2]), Number(m[3])) &&
          Number(m[4]) <= 23 &&
          Number(m[5]) <= 59 &&
          Number(m[6]) <= 59,
      );
    case 'month':
      return matchDateTimeShape(
        value,
        /^(\d{4})-(\d{2})$/,
        (m) => Number(m[2]) >= 1 && Number(m[2]) <= 12,
      );
    case 'time':
      return matchDateTimeShape(
        value,
        /^(\d{2}):(\d{2})$/,
        (m) => Number(m[1]) <= 23 && Number(m[2]) <= 59,
      );
  }
}

function matchDateTimeShape(
  value: string,
  pattern: RegExp,
  check: (match: RegExpMatchArray) => boolean,
): boolean {
  const match = value.match(pattern);
  return match !== null && check(match);
}

function isRealDate(year: number, month: number, day: number): boolean {
  if (month < 1 || month > 12 || day < 1) return false;
  const leap = (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
  const daysInMonth = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return day <= daysInMonth[month - 1];
}

// ---- JSON 安全兜底 ----

function isLooseJsonValue(value: unknown): value is FormJsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') {
    return true;
  }
  if (typeof value === 'number') return Number.isFinite(value);
  if (Array.isArray(value)) return value.every(isLooseJsonValue);
  if (typeof value !== 'object') return false;
  return Object.values(value).every(isLooseJsonValue);
}
