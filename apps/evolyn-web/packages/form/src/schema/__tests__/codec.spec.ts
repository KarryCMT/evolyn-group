import { describe, expect, it } from 'vitest';
import type { FormItem } from '../types';
import {
  emptyWidgetValue,
  isEmptyWidgetValue,
  isCanonicalDateTime,
  normalizeWidgetValue,
  validateWidgetValue,
} from '../codec';

function item(widget: Record<string, unknown>, extras: { label?: string } = {}): FormItem {
  return {
    widget: { enable: true, visible: true, ...widget } as FormItem['widget'],
    label: extras.label ?? '字段',
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

describe('normalizeWidgetValue 归一化', () => {
  it('文本类收敛 string|null', () => {
    const widget = { type: 'text', widgetName: '_widget_t', allowBlank: true };
    expect(normalizeWidgetValue(widget as never, 'ok')).toBe('ok');
    expect(normalizeWidgetValue(widget as never, 3)).toBeNull();
  });

  it('数字收敛有限 number|null（数字字符串转换）', () => {
    const widget = { type: 'number', widgetName: '_widget_n', allowBlank: true };
    expect(normalizeWidgetValue(widget as never, '3.5')).toBe(3.5);
    expect(normalizeWidgetValue(widget as never, 'abc')).toBeNull();
    expect(normalizeWidgetValue(widget as never, Number.NaN)).toBeNull();
  });

  it('多选收敛 string[]', () => {
    const widget = { type: 'combocheck', widgetName: '_widget_m', allowBlank: true };
    expect(normalizeWidgetValue(widget as never, ['a', 1, 'b'])).toEqual(['a', 'b']);
    expect(normalizeWidgetValue(widget as never, 'a')).toEqual([]);
  });

  it('空值形态与 isEmptyWidgetValue', () => {
    expect(emptyWidgetValue('combocheck')).toEqual([]);
    expect(emptyWidgetValue('text')).toBeNull();
    expect(isEmptyWidgetValue(null)).toBe(true);
    expect(isEmptyWidgetValue('')).toBe(true);
    expect(isEmptyWidgetValue([])).toBe(true);
    expect(isEmptyWidgetValue('a')).toBe(false);
  });
});

describe('validateWidgetValue 值校验（与后端文案一致）', () => {
  it('allowBlank=false 时空值必填，输入/选择动词区分', () => {
    const text = item({ type: 'text', widgetName: '_widget_t', allowBlank: false });
    expect(validateWidgetValue(text, null)).toEqual(['请输入字段']);
    const combo = item({
      type: 'combo',
      widgetName: '_widget_c',
      allowBlank: false,
      options: [{ label: 'A', value: 'a' }],
    });
    expect(validateWidgetValue(combo, null)).toEqual(['请选择字段']);
    expect(validateWidgetValue(combo, [])).toEqual(['请选择字段']);
  });

  it('allowBlank=true 时空值直接通过', () => {
    const text = item({ type: 'text', widgetName: '_widget_t', allowBlank: true, minLength: 2 });
    expect(validateWidgetValue(text, null)).toEqual([]);
  });

  it('文本长度与邮箱格式', () => {
    const text = item({
      type: 'text',
      widgetName: '_widget_t',
      allowBlank: true,
      minLength: 2,
      maxLength: 4,
    });
    expect(validateWidgetValue(text, 'a')).toEqual(['字段最少输入 2 个字符']);
    expect(validateWidgetValue(text, 'abcde')).toEqual(['字段不能超过 4 个字符']);
    const email = item({
      type: 'text',
      widgetName: '_widget_e',
      allowBlank: true,
      format: 'email',
    });
    expect(validateWidgetValue(email, 'not-mail')).toEqual(['字段格式不正确']);
    expect(validateWidgetValue(email, 'a@b.c')).toEqual([]);
  });

  it('数字范围与小数位', () => {
    const number = item({
      type: 'number',
      widgetName: '_widget_n',
      allowBlank: true,
      min: 0,
      max: 10,
      precision: 1,
    });
    expect(validateWidgetValue(number, -1)).toEqual(['字段不能小于 0']);
    expect(validateWidgetValue(number, 11)).toEqual(['字段不能大于 10']);
    expect(validateWidgetValue(number, 1.23)).toEqual(['字段最多支持 1 位小数']);
    expect(validateWidgetValue(number, 1.2)).toEqual([]);
  });

  it('日期时间按 format 形状与真实日历校验', () => {
    const datetime = item({
      type: 'datetime',
      widgetName: '_widget_d',
      allowBlank: true,
      format: 'datetime',
    });
    expect(validateWidgetValue(datetime, '2026-02-30 10:00:00')).toEqual(['字段的日期格式不正确']);
    expect(validateWidgetValue(datetime, '2026-02-28 23:59:59')).toEqual([]);
    expect(isCanonicalDateTime('2024-02-29', 'date')).toBe(true);
    expect(isCanonicalDateTime('2023-02-29', 'date')).toBe(false);
    expect(isCanonicalDateTime('2026-13', 'month')).toBe(false);
    expect(isCanonicalDateTime('24:00', 'time')).toBe(false);
  });

  it('选项命中与多选去重', () => {
    const combo = item({
      type: 'combo',
      widgetName: '_widget_c',
      allowBlank: true,
      options: [
        { label: 'A', value: 'a' },
        { label: 'B', value: 'b' },
      ],
    });
    expect(validateWidgetValue(combo, 'c')).toEqual(['字段的值不在选项范围内']);
    const multi = item({
      type: 'checkboxgroup',
      widgetName: '_widget_m',
      allowBlank: true,
      options: [
        { label: 'A', value: 'a' },
        { label: 'B', value: 'b' },
      ],
    });
    expect(validateWidgetValue(multi, ['a', 'a'])).toEqual(['字段的值存在重复选项']);
    expect(validateWidgetValue(multi, ['a', 'b'])).toEqual([]);
  });

  it('布局项不做值校验；成员字段校验选择值形状', () => {
    const separator = item({ type: 'separator', widgetName: '_widget_s', allowBlank: true });
    expect(validateWidgetValue(separator, 'anything')).toEqual([]);
    const user = item({ type: 'user', widgetName: '_widget_u', allowBlank: false });
    expect(validateWidgetValue(user, null)).toEqual(['请选择字段']);
    expect(validateWidgetValue(user, { id: 'm1' })).toEqual(['字段的值类型不正确']);
    expect(validateWidgetValue(user, 'm1')).toEqual([]);

    const usergroup = item({ type: 'usergroup', widgetName: '_widget_ug', allowBlank: false });
    expect(validateWidgetValue(usergroup, [])).toEqual(['请选择字段']);
    expect(validateWidgetValue(usergroup, ['m1', 'm1'])).toEqual(['字段的值存在重复成员']);
    expect(validateWidgetValue(usergroup, ['m1', 'm2'])).toEqual([]);
  });
});
