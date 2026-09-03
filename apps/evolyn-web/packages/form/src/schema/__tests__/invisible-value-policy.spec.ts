import { describe, expect, it } from 'vitest';
import {
  isRecomputeConfigurable,
  isSubmitRuleEligibleType,
  isSubmitRuleProcessableType,
  normalizeWidgetSubmitRules,
  previewInvisibleValue,
  readInvisibleValuePolicy,
  resolveSubmitStrategy,
  submitRuleLabel,
} from '../invisible-value-policy';
import type { FormContent, SubmitRule } from '../types';

/** 构造携带策略键的表单内容（缺键/脏键场景直接删改）。 */
function contentWith(overrides: Partial<Record<string, unknown>> = {}): FormContent {
  const content: Record<string, unknown> = {
    type: 'form',
    layout: 'normal',
    items: [],
    layout_fields: [],
    field_layout: [],
    fieldShowRules: [],
    submitRule: 2,
    widget_submit_rules: {},
  };
  Object.assign(content, overrides);
  return content as unknown as FormContent;
}

describe('readInvisibleValuePolicy 协议解析', () => {
  it('合法文档解析默认策略与例外映射', () => {
    const policy = readInvisibleValuePolicy(
      contentWith({ submitRule: 1, widget_submit_rules: { _widget_a: 2, _widget_b: 1 } }),
    );
    expect(policy.submitRule).toBe(1);
    expect(policy.specialRules).toEqual({ _widget_a: 2, _widget_b: 1 });
  });

  it('缺键或非法片段防御式回退默认「空值」', () => {
    const policy = readInvisibleValuePolicy(
      contentWith({ submitRule: '2', widget_submit_rules: { _widget_a: '1', _widget_b: 4 } }),
    );
    expect(policy.submitRule).toBe(2);
    expect(policy.specialRules).toEqual({});
  });
});

describe('resolveSubmitStrategy 策略解析', () => {
  it('特殊规则优先于默认策略（§3.3）', () => {
    const policy = { submitRule: 2 as SubmitRule, specialRules: { _widget_a: 1 as SubmitRule } };
    expect(resolveSubmitStrategy(policy, '_widget_a')).toBe(1);
    expect(resolveSubmitStrategy(policy, '_widget_other')).toBe(2);
  });
});

describe('可处理性判定', () => {
  it('仅具备值语义的开放控件可配置特殊规则', () => {
    for (const type of ['text', 'textarea', 'number', 'datetime', 'radiogroup', 'usergroup']) {
      expect(isSubmitRuleEligibleType(type)).toBe(true);
    }
    for (const type of ['separator', 'button', 'richtext', 'sn', 'linkquery', 'subform', 'dept']) {
      expect(isSubmitRuleEligibleType(type)).toBe(false);
    }
  });

  it('布局节点不参与策略决议，其余字段均可处理', () => {
    expect(isSubmitRuleProcessableType('separator')).toBe(false);
    expect(isSubmitRuleProcessableType('button')).toBe(false);
    expect(isSubmitRuleProcessableType('sn')).toBe(true);
  });

  it('recompute 在派生执行器交付前不可配置', () => {
    expect(isRecomputeConfigurable()).toBe(false);
  });
});

describe('normalizeWidgetSubmitRules 归一化', () => {
  it('剔除与默认策略相同的冗余项', () => {
    expect(normalizeWidgetSubmitRules({ _widget_a: 2, _widget_b: 1, _widget_c: 3 }, 2)).toEqual({
      _widget_b: 1,
      _widget_c: 3,
    });
  });
});

describe('submitRuleLabel / previewInvisibleValue 展示预演', () => {
  it('策略展示名固定', () => {
    expect(submitRuleLabel(1)).toBe('保持原值');
    expect(submitRuleLabel(2)).toBe('空值');
    expect(submitRuleLabel(3)).toBe('始终重新计算');
  });

  it('clear/preserve 预演：类型空值与基线保留；recompute 客户端不可预演', () => {
    expect(previewInvisibleValue(2, 'checkboxgroup', ['a'])).toEqual([]);
    expect(previewInvisibleValue(2, 'text', '旧值')).toBeNull();
    expect(previewInvisibleValue(1, 'text', '旧值')).toBe('旧值');
    // preserve 新建无基线 → 类型空值（§3.2 设计器提示口径）。
    expect(previewInvisibleValue(1, 'text', undefined)).toBeNull();
    // recompute 的值由服务端执行器产出，客户端预演一律空值（不信任前端旧值）。
    expect(previewInvisibleValue(3, 'number', 42)).toBeNull();
  });
});
