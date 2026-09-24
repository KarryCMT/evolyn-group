import { describe, expect, it } from 'vitest';
import type { IntelligentFieldOption, IntelligentSourceFieldGroup } from '../types';
import {
  intelligentFieldRecommendationScore,
  isIntelligentFieldCompatible,
  quickFillIntelligentAssignments,
} from '../createRecord';

function field(
  widgetName: string,
  label: string,
  valueKind: IntelligentFieldOption['valueKind'] = 'text',
): IntelligentFieldOption {
  return {
    fieldId: `id_${widgetName}`,
    widgetName,
    label,
    widgetType: valueKind,
    valueKind,
    required: false,
  };
}

describe('create record field assignments', () => {
  it('只允许语义兼容的字段来源', () => {
    expect(isIntelligentFieldCompatible(field('name', '姓名'), field('title', '标题'))).toBe(true);
    expect(
      isIntelligentFieldCompatible(
        field('birthday', '出生日期', 'date'),
        field('name', '姓名'),
      ),
    ).toBe(false);
    expect(
      intelligentFieldRecommendationScore(
        field('birthday', '出生日期', 'date'),
        field('source_birthday', '出生日期', 'date'),
      ),
    ).toBe(90);
  });

  it('快捷填充只补齐空字段且不覆盖人工配置', () => {
    const targets = [field('name', '姓名'), field('birthday', '出生日期', 'date')];
    const groups: IntelligentSourceFieldGroup[] = [
      {
        nodeId: 'trigger_1',
        nodeName: '触发数据',
        fields: [field('name', '姓名'), field('birthday', '出生日期', 'date')],
      },
    ];
    const existing = [
      {
        targetFieldId: targets[0]!.fieldId,
        targetWidgetName: targets[0]!.widgetName,
        source: { type: 'custom' as const, value: '固定姓名' },
      },
    ];

    const result = quickFillIntelligentAssignments(targets, groups, existing);

    expect(result.filledCount).toBe(1);
    expect(result.assignments).toHaveLength(2);
    expect(result.assignments[0]).toEqual(existing[0]);
    expect(result.assignments[1]).toMatchObject({
      targetWidgetName: 'birthday',
      source: { type: 'node-field', nodeId: 'trigger_1', field: 'birthday' },
    });
  });
});
