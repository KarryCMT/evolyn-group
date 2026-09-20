import { describe, expect, it } from 'vitest';
import {
  compareDecimalText,
  DECIMAL_TEXT_PATTERN,
  decimalDigitIssue,
  decimalFractionDigits,
  decimalIntegerDigits,
  effectiveNumericPrecision,
  effectiveNumericScale,
} from '../numeric';
import { formatPercentRatio, parsePercentInput, usesPercentRatio } from '../percent';
import {
  formatMoneyInputValue,
  formatMoneyValue,
  parseMoneyInput,
  resolveMoneyCurrencyCode,
} from '../money';
import { createWidgetItem } from '../dictionary';
import { normalizeWidgetValue, validateWidgetValue } from '../codec';
import { validateFormSchema, validatePublishableFormSchema } from '../validate';
import { projectPhysicalStorageFields } from '../physical';
import { PUBLISHABLE_WIDGET_TYPES } from '../types';
import type { DecimalFamilyWidget, FormItem } from '../types';

let fieldIdSeed = 0;

/** 数值字段族测试字段项工厂：fieldId 唯一，属性按需覆盖。 */
function numericItem(
  type: 'decimal' | 'money' | 'percent',
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  fieldIdSeed += 1;
  return {
    widget: {
      type,
      widgetName: '_widget_n1',
      fieldId: String(fieldIdSeed).padStart(10, '0'),
      enable: true,
      visible: true,
      allowBlank: true,
      ...overrides,
    },
    label: '数值',
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

function documentWith(items: unknown[]): unknown {
  const field_layout = items.flatMap((item) => {
    const name = (item as { widget?: { widgetName?: unknown } })?.widget?.widgetName;
    return typeof name === 'string' ? [name] : [];
  });
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items,
      layout_fields: [],
      field_layout,
      fieldShowRules: [],
      submitRule: 2,
      widget_submit_rules: {},
      validators: [],
      preSubmitConfirm: {
        enable: false,
        title: '确认继续提交吗？',
        content: '请确认填写内容无误后继续提交。',
      },
      formEvents: [],
    },
  };
}

function issuesOf(input: unknown): { path: string; message: string }[] {
  return validateFormSchema(input).issues;
}

describe('数值字段族语义助手', () => {
  it('有效精度/小数位：显式配置优先，否则按类型取默认', () => {
    const decimal = { type: 'decimal' as const, precision: null, scale: null };
    expect(effectiveNumericPrecision(decimal)).toBe(20);
    expect(effectiveNumericScale(decimal)).toBe(6);
    const money = { type: 'money' as const, precision: null, scale: null };
    expect(effectiveNumericScale(money)).toBe(2);
    const percent = { type: 'percent' as const, precision: null, scale: null };
    expect(effectiveNumericPrecision(percent)).toBe(10);
    expect(effectiveNumericScale(percent)).toBe(6);
    expect(effectiveNumericPrecision({ ...decimal, precision: 30 })).toBe(30);
    expect(effectiveNumericScale({ ...money, scale: 4 })).toBe(4);
  });

  it('位数计数按值语义：尾随零/前导零不计位', () => {
    expect(decimalFractionDigits('1.500')).toBe(1);
    expect(decimalFractionDigits('100')).toBe(0);
    expect(decimalFractionDigits('0.000')).toBe(0);
    expect(decimalIntegerDigits('007.5')).toBe(1);
    expect(decimalIntegerDigits('0')).toBe(0);
    expect(decimalIntegerDigits('-0.5')).toBe(0);
    expect(decimalIntegerDigits('123.45')).toBe(3);
    expect(decimalDigitIssue('1.234', 20, 2)).toBe('scale');
    expect(decimalDigitIssue('1234567890123456789.5', 20, 2)).toBe('precision');
    expect(decimalDigitIssue('12.34', 20, 2)).toBeNull();
  });

  it('十进制文本形状：拒绝指数记法、正号、裸小数点与空白', () => {
    for (const ok of ['-3.5', '0', '100.125', '007.500']) {
      expect(DECIMAL_TEXT_PATTERN.test(ok)).toBe(true);
    }
    for (const bad of ['1e3', '+1', '.5', '3.', ' 1', '1 ', 'abc', '1.2.3', '']) {
      expect(DECIMAL_TEXT_PATTERN.test(bad)).toBe(false);
    }
  });

  it('十进制比较走精确值序（与 Go 镜像对拍）', () => {
    expect(compareDecimalText('0.1', '0.10000000000000000000001')).toBe(-1);
    expect(compareDecimalText('100', '99.999999999999999999999')).toBe(1);
    expect(compareDecimalText('-1.5', '-1.50')).toBe(0);
  });

  it('百分比界面值与协议比例精确互转，非法草稿原文保留给校验器处理', () => {
    expect(formatPercentRatio('0.15')).toBe('15');
    expect(formatPercentRatio('1')).toBe('100');
    expect(formatPercentRatio('0.123456')).toBe('12.3456');
    expect(parsePercentInput('15')).toBe('0.15');
    expect(parsePercentInput('12.3456')).toBe('0.123456');
    expect(parsePercentInput('1e2')).toBe('1e2');
    expect(usesPercentRatio({ type: 'percent', percentValueMode: 'ratio' })).toBe(true);
    expect(usesPercentRatio({ type: 'percent' })).toBe(false);
  });

  it('金额按字段币种精确分组与定长展示，输入可剥离展示字符', () => {
    const cny = { type: 'money' as const, currencyCode: 'CNY' as const, scale: 2 };
    const jpy = { type: 'money' as const, currencyCode: 'JPY' as const, scale: 0 };
    expect(formatMoneyInputValue('1234567890123456.5', cny)).toBe('1,234,567,890,123,456.50');
    expect(formatMoneyValue('1234.5', cny)).toBe('¥1,234.50');
    expect(formatMoneyValue('-1234', jpy)).toBe('-￥1,234');
    expect(formatMoneyValue('-1234.5', jpy)).toBe('-1234.5');
    expect(parseMoneyInput(' HK$ 1,234.50 ', { currencyCode: 'HKD' })).toBe('1234.50');
    expect(resolveMoneyCurrencyCode({})).toBe('CNY');
  });
});

describe('数值字段族 schema 校验', () => {
  it('接受三种类型的最小合法形态并可发布', () => {
    for (const type of ['decimal', 'money', 'percent'] as const) {
      const input = documentWith([numericItem(type)]);
      expect(validateFormSchema(input).issues).toEqual([]);
      expect(validatePublishableFormSchema(input).valid).toBe(true);
    }
    expect(PUBLISHABLE_WIDGET_TYPES).toContain('decimal');
    expect(PUBLISHABLE_WIDGET_TYPES).toContain('money');
    expect(PUBLISHABLE_WIDGET_TYPES).toContain('percent');
  });

  it('min/max/defaultValue 拒绝非十进制字符串形状', () => {
    const issues = issuesOf(documentWith([numericItem('decimal', { min: '1e3' })]));
    expect(issues).toContainEqual({
      path: 'content.items[0].widget.min',
      message: 'min 必须是十进制数字字符串（null 表示未启用）',
    });
    expect(issuesOf(documentWith([numericItem('money', { defaultValue: 12.5 })]))).toContainEqual({
      path: 'content.items[0].widget.defaultValue',
      message: 'defaultValue 必须是十进制数字字符串（null 表示未启用）',
    });
  });

  it('precision/scale 超出护栏即拒绝', () => {
    expect(issuesOf(documentWith([numericItem('decimal', { precision: 41 })]))).toContainEqual({
      path: 'content.items[0].widget.precision',
      message: 'precision 不在允许范围 1–40 内',
    });
    expect(issuesOf(documentWith([numericItem('money', { scale: 19 })]))).toContainEqual({
      path: 'content.items[0].widget.scale',
      message: 'scale 不在允许范围 0–18 内',
    });
  });

  it('rounding 只接受七种平台舍入模式', () => {
    expect(
      issuesOf(documentWith([numericItem('percent', { rounding: 'HALF_CEIL' })])),
    ).toContainEqual({
      path: 'content.items[0].widget.rounding',
      message:
        'rounding 必须是以下枚举值之一：UP / DOWN / CEIL / FLOOR / HALF_UP / HALF_DOWN / HALF_EVEN',
    });
    const ok = documentWith([numericItem('percent', { rounding: 'HALF_EVEN' })]);
    expect(validateFormSchema(ok).issues).toEqual([]);
  });

  it('交叉规则：有效 scale 不得大于有效 precision', () => {
    // precision 显式 5 时，缺省 scale=6 生效后超限，必须拒绝。
    const issues = issuesOf(documentWith([numericItem('decimal', { precision: 5 })]));
    expect(issues).toContainEqual({
      path: 'content.items[0].widget.scale',
      message: 'scale 不能大于 precision',
    });
  });

  it('交叉规则：min ≤ max 与 defaultValue 范围（decimal 值序）', () => {
    expect(
      issuesOf(documentWith([numericItem('money', { min: '100.5', max: '100.4' })])),
    ).toContainEqual({
      path: 'content.items[0].widget.max',
      message: 'max 不能小于 min',
    });
    expect(
      issuesOf(
        documentWith([numericItem('money', { min: '10', max: '20', defaultValue: '20.01' })]),
      ),
    ).toContainEqual({
      path: 'content.items[0].widget.defaultValue',
      message: 'defaultValue 不能大于 max',
    });
  });

  it('交叉规则：min/max/defaultValue 自身受位数约束（防不可满足范围）', () => {
    expect(issuesOf(documentWith([numericItem('money', { min: '0.123' })]))).toContainEqual({
      path: 'content.items[0].widget.min',
      message: 'min 最多支持 2 位小数',
    });
    expect(
      issuesOf(documentWith([numericItem('percent', { max: '12345678901234.5' })])),
    ).toContainEqual({
      path: 'content.items[0].widget.max',
      message: 'max 整数位最多 4 位',
    });
  });

  it('未知属性与显隐条件值形状按协议拒绝', () => {
    expect(issuesOf(documentWith([numericItem('decimal', { currency: 'CNY' })]))).toContainEqual({
      path: 'content.items[0].widget.currency',
      message: '未知属性「currency」',
    });
    const input = documentWith([numericItem('money')]) as {
      content: {
        fieldShowRules: {
          id: string;
          filter: { rel: string; cond: unknown[] };
          fields: string[];
        }[];
      };
    };
    input.content.fieldShowRules = [
      {
        id: '_field_show_rule_1',
        filter: {
          rel: 'and',
          cond: [
            {
              field: '_widget_n1',
              type: 'money',
              method: 'gt',
              value: [100],
            },
          ],
        },
        fields: [],
      },
    ];
    expect(issuesOf(input)).toContainEqual({
      path: 'content.fieldShowRules[0].filter.cond[0].value[0]',
      message: 'value 条目必须是十进制数字字符串',
    });
    expect(
      validateFormSchema(documentWith([numericItem('money', { currencyCode: 'USD' })])).issues,
    ).toEqual([]);
    expect(issuesOf(documentWith([numericItem('money', { currencyCode: 'BTC' })]))).toContainEqual({
      path: 'content.items[0].widget.currencyCode',
      message:
        'currencyCode 必须是以下枚举值之一：CNY / USD / EUR / GBP / JPY / HKD / KRW / SGD / AUD / CAD / CHF / AED',
    });
    expect(
      issuesOf(documentWith([numericItem('decimal', { currencyCode: 'CNY' })])),
    ).toContainEqual({
      path: 'content.items[0].widget.currencyCode',
      message: '未知属性「currencyCode」',
    });
  });

  it('新建字段按类型预写 precision/scale（物理列形态显式化）', () => {
    const decimal = createWidgetItem('decimal').widget as DecimalFamilyWidget;
    expect(decimal.precision).toBe(20);
    expect(decimal.scale).toBe(6);
    const money = createWidgetItem('money').widget as DecimalFamilyWidget;
    expect(money.precision).toBe(20);
    expect(money.scale).toBe(2);
    expect(money.currencyCode).toBe('CNY');
    const percent = createWidgetItem('percent').widget as DecimalFamilyWidget;
    expect(percent.precision).toBe(10);
    expect(percent.scale).toBe(6);
    expect(percent.min).toBe('0');
    expect(percent.max).toBe('1');
    expect(percent.percentValueMode).toBe('ratio');
    expect(validateFormSchema(documentWith([createWidgetItem('percent')])).issues).toEqual([]);
    expect(validateFormSchema(documentWith([createWidgetItem('money')])).issues).toEqual([]);
  });
});

describe('数值字段族值校验', () => {
  function item(type: 'decimal' | 'money' | 'percent', overrides = {}): FormItem {
    return numericItem(type, overrides) as unknown as FormItem;
  }

  it('合法 decimal string 通过；类型与形状错误逐字回填', () => {
    expect(validateWidgetValue(item('decimal'), '123.45')).toEqual([]);
    expect(validateWidgetValue(item('decimal'), '-0.5')).toEqual([]);
    expect(validateWidgetValue(item('decimal'), null)).toEqual([]);
    expect(
      validateWidgetValue(
        {
          ...item('decimal'),
          widget: { ...item('decimal').widget, allowBlank: false } as DecimalFamilyWidget,
        },
        null,
      ),
    ).toEqual(['请输入数值']);
    expect(validateWidgetValue(item('decimal'), 123.45)).toEqual(['数值的值类型不正确']);
    expect(validateWidgetValue(item('decimal'), '1e3')).toEqual(['数值格式不正确']);
  });

  it('位数与范围按有效精度复核（尾随零不计位）', () => {
    // money 缺省 scale=2：'1.500' 值语义 1.5 位，通过；'1.234' 拒绝。
    expect(validateWidgetValue(item('money'), '1.500')).toEqual([]);
    expect(validateWidgetValue(item('money'), '1.234')).toEqual(['数值最多支持 2 位小数']);
    // percent 缺省 precision=10/scale=6：整数位上限 4。
    expect(validateWidgetValue(item('percent'), '12345.5')).toEqual(['数值整数位最多 4 位']);
    expect(validateWidgetValue(item('money', { min: '10.5', max: '20' }), '10.49')).toEqual([
      '数值不能小于 10.5',
    ]);
    expect(validateWidgetValue(item('money', { min: '10.5', max: '20' }), '20.5')).toEqual([
      '数值不能大于 20',
    ]);
  });

  it('归一化：decimal string 原样保留，其余收敛 null', () => {
    const widget = item('decimal').widget as DecimalFamilyWidget;
    expect(normalizeWidgetValue(widget, '123.45')).toBe('123.45');
    expect(normalizeWidgetValue(widget, 123.45)).toBeNull();
    expect(normalizeWidgetValue(widget, undefined)).toBeNull();
  });

  it('物理投影：三种类型落 decimal 逻辑存储类型', () => {
    const fields = projectPhysicalStorageFields([item('decimal'), item('money'), item('percent')]);
    expect(fields.map((field) => field.type)).toEqual(['decimal', 'decimal', 'decimal']);
  });
});
