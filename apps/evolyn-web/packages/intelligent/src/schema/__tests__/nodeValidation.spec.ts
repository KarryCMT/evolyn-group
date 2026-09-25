import { describe, expect, it } from 'vitest';
import type { IntelligentNode, IntelligentTrigger } from '../types';
import { getIntelligentNodeConfigurationState } from '../nodeValidation';

const trigger: IntelligentTrigger = {
  type: 'form',
  formCode: 'form_employee',
  formName: '员工档案',
  eventName: '新增数据时',
  actions: [{ id: 'create', type: 'create' }],
  conditionMode: 'all',
  conditions: [],
};

function actionNode(patch: Partial<IntelligentNode> = {}): IntelligentNode {
  return {
    id: 'action_1',
    type: 'action',
    name: '新增数据',
    description: '向目标表单新增数据',
    position: { x: 0, y: 0 },
    actionType: 'create-record',
    config: {},
    ...patch,
  };
}

describe('getIntelligentNodeConfigurationState', () => {
  it('marks a newly inserted action as empty and invalid', () => {
    expect(getIntelligentNodeConfigurationState(actionNode(), trigger)).toEqual({
      configured: false,
      empty: true,
    });
  });

  it('accepts a create action after its target form is selected', () => {
    const node = actionNode({
      config: { targetFormCode: 'form_product', targetFormName: '产品管理' },
    });
    expect(getIntelligentNodeConfigurationState(node, trigger).configured).toBe(true);
  });

  it('keeps a partially configured update action in the validation-failed state', () => {
    const node = actionNode({
      actionType: 'update-record',
      name: '修改数据',
      config: { targetFormCode: 'form_employee', targetFormName: '员工档案' },
    });
    expect(getIntelligentNodeConfigurationState(node, trigger)).toEqual({
      configured: false,
      empty: false,
    });
  });

  it('accepts a structurally complete update action', () => {
    const node = actionNode({
      actionType: 'update-record',
      name: '修改数据',
      config: {
        targetFormCode: 'form_employee',
        updateTargetMode: 'form',
        updateFilters: [
          {
            id: 'filter_1',
            targetFieldId: 'employee_name',
            targetWidgetName: 'employeeName',
            operator: 'is-not-empty',
          },
        ],
        fieldAssignments: [
          {
            targetFieldId: 'position',
            targetWidgetName: 'position',
            source: { type: 'custom', value: '产品经理' },
          },
        ],
      },
    });
    expect(getIntelligentNodeConfigurationState(node, trigger).configured).toBe(true);
  });
});
