import { describe, expect, it } from 'vitest';
import {
  compileFieldShowRules,
  downstreamTargets,
  evaluateFieldShowRules,
  matchFieldShowCondition,
  type FieldShowEvaluationContext,
} from '../rules';
import type { FieldShowCondition, FieldShowRule, FormContent } from '../types';

/** 构造条件（默认 text 源字段）。 */
function condition(overrides: Partial<FieldShowCondition> = {}): FieldShowCondition {
  return { field: '_widget_src', type: 'text', method: 'eq', value: ['甲'], ...overrides };
}

/** 构造规则：单条件显示单个目标。 */
function rule(overrides: Partial<FieldShowRule> = {}, id = '_rule_1'): FieldShowRule {
  return {
    id,
    filter: { rel: 'and', cond: [condition()] },
    fields: ['_widget_target'],
    ...overrides,
  };
}

function contentWith(rules: FieldShowRule[]): FormContent {
  return {
    type: 'form',
    layout: 'normal',
    items: [],
    layout_fields: [],
    field_layout: [],
    fieldShowRules: rules,
    submitRule: 2,
    widget_submit_rules: {},
  };
}

/** 全量求值快捷入口：值表 + 基础可见性（默认全可见）。 */
function evaluate(
  rules: FieldShowRule[],
  values: Record<string, unknown>,
  options: Partial<FieldShowEvaluationContext> = {},
): Map<string, boolean> {
  return evaluateFieldShowRules(compileFieldShowRules(contentWith(rules)), {
    valueOf: (field) => values[field],
    isBaseVisible: () => true,
    ...options,
  });
}

describe('matchFieldShowCondition 条件矩阵', () => {
  it('text：eq/ne/contains/notContains 与空值语义', () => {
    const src = { field: '_widget_src', type: 'text' } as const;
    expect(matchFieldShowCondition({ ...src, method: 'eq', value: ['甲'] }, '甲')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'eq', value: ['甲'] }, '乙')).toBe(false);
    expect(matchFieldShowCondition({ ...src, method: 'ne', value: ['甲'] }, '乙')).toBe(true);
    expect(
      matchFieldShowCondition({ ...src, method: 'contains', value: ['工时'] }, '总工时统计'),
    ).toBe(true);
    expect(
      matchFieldShowCondition({ ...src, method: 'notContains', value: ['工时'] }, '备注'),
    ).toBe(true);
    // 空值只允许 isEmpty/notEmpty 成立，其余一律不成立。
    expect(matchFieldShowCondition({ ...src, method: 'ne', value: ['甲'] }, null)).toBe(false);
    expect(matchFieldShowCondition({ ...src, method: 'isEmpty' }, '')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'isEmpty' }, [])).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'notEmpty' }, '')).toBe(false);
  });

  it('number：有序比较与 between 区间（含边界）', () => {
    const src = { field: '_widget_src', type: 'number' } as const;
    expect(matchFieldShowCondition({ ...src, method: 'gt', value: [10] }, 10.5)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'gte', value: [10] }, 10)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'lt', value: [10] }, 9.9)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'lte', value: [10] }, 10)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'between', value: [10, 20] }, 15)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'between', value: [10, 20] }, 10)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'between', value: [10, 20] }, 20)).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'between', value: [10, 20] }, 20.1)).toBe(
      false,
    );
    // 类型不匹配按不成立处理。
    expect(matchFieldShowCondition({ ...src, method: 'eq', value: [10] }, '10')).toBe(false);
  });

  it('datetime：规范形状字符串按字典序比较', () => {
    const src = { field: '_widget_src', type: 'datetime' } as const;
    expect(
      matchFieldShowCondition({ ...src, method: 'gte', value: ['2026-01-01'] }, '2026-01-02'),
    ).toBe(true);
    expect(
      matchFieldShowCondition(
        { ...src, method: 'between', value: ['2026-01-01', '2026-12-31'] },
        '2026-06-15',
      ),
    ).toBe(true);
    expect(
      matchFieldShowCondition({ ...src, method: 'lt', value: ['2026-01-01'] }, '2025-12-31'),
    ).toBe(true);
  });

  it('单选语义：eq/ne 与 in/notIn（radiogroup/combo/user/dept 同构）', () => {
    const src = { field: '_widget_src', type: 'radiogroup' } as const;
    expect(matchFieldShowCondition({ ...src, method: 'eq', value: ['A'] }, 'A')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'ne', value: ['A'] }, 'B')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'in', value: ['A', 'B'] }, 'B')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'notIn', value: ['A', 'B'] }, 'C')).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'notIn', value: ['A', 'B'] }, null)).toBe(
      false,
    );
  });

  it('多选语义：containsAny/containsAll/containsNone', () => {
    const src = { field: '_widget_src', type: 'checkboxgroup' } as const;
    expect(
      matchFieldShowCondition({ ...src, method: 'containsAny', value: ['A', 'B'] }, ['B', 'C']),
    ).toBe(true);
    expect(
      matchFieldShowCondition({ ...src, method: 'containsAny', value: ['A', 'B'] }, ['C']),
    ).toBe(false);
    expect(
      matchFieldShowCondition({ ...src, method: 'containsAll', value: ['A', 'B'] }, [
        'B',
        'A',
        'C',
      ]),
    ).toBe(true);
    expect(
      matchFieldShowCondition({ ...src, method: 'containsAll', value: ['A', 'B'] }, ['A']),
    ).toBe(false);
    expect(
      matchFieldShowCondition({ ...src, method: 'containsNone', value: ['A', 'B'] }, ['C', 'D']),
    ).toBe(true);
    expect(matchFieldShowCondition({ ...src, method: 'containsNone', value: ['A'] }, ['A'])).toBe(
      false,
    );
    expect(matchFieldShowCondition({ ...src, method: 'isEmpty' }, [])).toBe(true);
  });

  it('includeCurrentMember：当前成员注入比较集合；未注入时不参与', () => {
    const src = { field: '_widget_src', type: 'user' } as const;
    const eq = { ...src, method: 'eq' as const, value: ['member_a'], includeCurrentMember: true };
    expect(matchFieldShowCondition(eq, 'member_current', 'member_current')).toBe(true);
    expect(matchFieldShowCondition(eq, 'member_current', undefined)).toBe(false);
    expect(matchFieldShowCondition(eq, 'member_a')).toBe(true);
    const group = {
      ...src,
      type: 'usergroup' as const,
      method: 'containsAny' as const,
      value: ['member_a'],
      includeCurrentMember: true,
    };
    expect(matchFieldShowCondition(group, ['member_x', 'member_current'], 'member_current')).toBe(
      true,
    );
    expect(matchFieldShowCondition(group, ['member_x'], 'member_current')).toBe(false);
  });
});

describe('evaluateFieldShowRules 规则求值', () => {
  it('and 需全部成立，or 任一成立', () => {
    const andRule = rule({
      filter: {
        rel: 'and',
        cond: [
          condition({ method: 'eq', value: ['甲'] }),
          condition({ field: '_widget_num', type: 'number', method: 'gt', value: [10] }),
        ],
      },
    });
    expect(evaluate([andRule], { _widget_src: '甲', _widget_num: 20 }).get('_widget_target')).toBe(
      true,
    );
    expect(evaluate([andRule], { _widget_src: '甲', _widget_num: 5 }).get('_widget_target')).toBe(
      false,
    );
    const orRule = rule({ filter: { rel: 'or', cond: andRule.filter.cond } });
    expect(evaluate([orRule], { _widget_src: '乙', _widget_num: 20 }).get('_widget_target')).toBe(
      true,
    );
    expect(evaluate([orRule], { _widget_src: '乙', _widget_num: 5 }).get('_widget_target')).toBe(
      false,
    );
  });

  it('条件源基础不可见（静态/权限隐藏）时条件视为不成立', () => {
    const result = evaluate(
      [rule()],
      { _widget_src: '甲' },
      {
        isBaseVisible: (field) => field !== '_widget_src',
      },
    );
    expect(result.get('_widget_target')).toBe(false);
  });

  it('多级 A→B→C 按拓扑序传播：上游隐藏时下游自然隐藏', () => {
    const rules: FieldShowRule[] = [
      rule(
        {
          filter: { rel: 'and', cond: [condition({ method: 'eq', value: ['是'] })] },
          fields: ['_widget_b'],
        },
        '_rule_b',
      ),
      {
        id: '_rule_c',
        filter: {
          rel: 'and',
          cond: [{ field: '_widget_b', type: 'text', method: 'notEmpty' }],
        },
        fields: ['_widget_c'],
      },
    ];
    // A=是 → B 显示；B 已填 → C 显示。
    const shown = evaluate(rules, { _widget_src: '是', _widget_b: '内容' });
    expect(shown.get('_widget_b')).toBe(true);
    expect(shown.get('_widget_c')).toBe(true);
    // A≠是 → B 隐藏 → B 的条件源不可见 → C 隐藏（不读取其隐藏值）。
    const hidden = evaluate(rules, { _widget_src: '否', _widget_b: '内容' });
    expect(hidden.get('_widget_b')).toBe(false);
    expect(hidden.get('_widget_c')).toBe(false);
  });

  it('畸形规则片段被防御性跳过，不阻断其余规则', () => {
    const broken = rule({ id: '' });
    const healthy = rule(
      { filter: { rel: 'and', cond: [condition({ method: 'eq', value: ['甲'] })] } },
      '_rule_ok',
    );
    const compiled = compileFieldShowRules(contentWith([broken, healthy]));
    expect(compiled.rules.map((entry) => entry.id)).toEqual(['_rule_ok']);
  });
});

describe('downstreamTargets 定向重算闭包', () => {
  it('仅收集变更字段的下游目标，按拓扑序返回；无关字段不进入闭包', () => {
    const rules: FieldShowRule[] = [
      rule({ filter: { rel: 'and', cond: [condition()] }, fields: ['_widget_b'] }, '_rule_b'),
      {
        id: '_rule_c',
        filter: { rel: 'and', cond: [{ field: '_widget_b', type: 'text', method: 'notEmpty' }] },
        fields: ['_widget_c'],
      },
      {
        id: '_rule_d',
        filter: {
          rel: 'and',
          cond: [{ field: '_widget_other', type: 'text', method: 'notEmpty' }],
        },
        fields: ['_widget_d'],
      },
    ];
    const compiled = compileFieldShowRules(contentWith(rules));
    expect(downstreamTargets(compiled, '_widget_src')).toEqual(['_widget_b', '_widget_c']);
    expect(downstreamTargets(compiled, '_widget_b')).toEqual(['_widget_c']);
    expect(downstreamTargets(compiled, '_widget_other')).toEqual(['_widget_d']);
    expect(downstreamTargets(compiled, '_widget_b_c_none')).toEqual([]);
  });
});
