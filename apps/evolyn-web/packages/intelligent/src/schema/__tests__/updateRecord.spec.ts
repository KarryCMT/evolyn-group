import { describe, expect, it } from 'vitest';
import type { IntelligentActionConfig, IntelligentFieldOption } from '../types';
import {
  createIntelligentUpdateFilter,
  isIntelligentUnaryUpdateOperator,
  validateIntelligentUpdateConfig,
} from '../updateRecord';

const fields: IntelligentFieldOption[] = [
  {
    fieldId: 'field_name',
    widgetName: 'employee_name',
    label: '员工姓名',
    widgetType: 'text',
    valueKind: 'text',
    required: true,
  },
  {
    fieldId: 'field_department',
    widgetName: 'department',
    label: '所属部门',
    widgetType: 'dept',
    valueKind: 'department',
    required: false,
  },
];

describe('update record configuration', () => {
  it('创建稳定的空筛选条件并识别一元运算符', () => {
    const filter = createIntelligentUpdateFilter();

    expect(filter.id).toMatch(/^update_filter_/);
    expect(filter.operator).toBe('equals');
    expect(isIntelligentUnaryUpdateOperator('is-empty')).toBe(true);
    expect(isIntelligentUnaryUpdateOperator('equals')).toBe(false);
  });

  it('表单模式要求完整筛选条件和至少一个字段赋值', () => {
    const incomplete: IntelligentActionConfig = {
      updateTargetMode: 'form',
      targetFormCode: 'form_employee',
      updateFilters: [createIntelligentUpdateFilter()],
      fieldAssignments: [],
    };
    expect(validateIntelligentUpdateConfig(incomplete, fields)).toEqual({
      target: true,
      filters: false,
      assignments: false,
      complete: false,
    });

    const complete: IntelligentActionConfig = {
      ...incomplete,
      updateFilters: [
        {
          id: 'filter_1',
          targetFieldId: 'field_name',
          targetWidgetName: 'employee_name',
          operator: 'equals',
          source: { type: 'custom', value: '张三' },
        },
      ],
      fieldAssignments: [
        {
          targetFieldId: 'field_department',
          targetWidgetName: 'department',
          source: { type: 'node-field', nodeId: 'trigger_1', field: 'department' },
        },
      ],
    };
    expect(validateIntelligentUpdateConfig(complete, fields).complete).toBe(true);
  });

  it('节点模式不需要筛选条件，但拒绝重复目标字段和空节点字段来源', () => {
    const config: IntelligentActionConfig = {
      updateTargetMode: 'node',
      targetNodeId: 'create_1',
      targetFormCode: 'form_employee',
      fieldAssignments: [
        {
          targetFieldId: 'field_name',
          targetWidgetName: 'employee_name',
          source: { type: 'node-field', nodeId: '', field: '' },
        },
      ],
    };

    expect(validateIntelligentUpdateConfig(config, fields)).toMatchObject({
      target: true,
      filters: true,
      assignments: false,
      complete: false,
    });
  });
});
