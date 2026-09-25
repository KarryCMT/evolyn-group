import { describe, expect, it } from 'vitest';
import {
  firstAvailableTriggerFieldId,
  uniqueFormTriggerActions,
  uniqueFormTriggerConditions,
} from '../triggerSelections';

describe('form trigger unique selections', () => {
  it('同一种触发动作只保留首项', () => {
    const actions = uniqueFormTriggerActions([
      { id: 'create_1', type: 'create' },
      { id: 'create_2', type: 'create' },
      { id: 'delete_1', type: 'delete' },
    ]);

    expect(actions.map((action) => action.id)).toEqual(['create_1', 'delete_1']);
  });

  it('同一个条件字段只保留首项', () => {
    const conditions = uniqueFormTriggerConditions([
      { id: 'name_1', fieldId: 'employee_name', operator: 'equals', values: ['张三'] },
      { id: 'name_2', fieldId: 'employee_name', operator: 'not-equals', values: ['李四'] },
      { id: 'phone_1', fieldId: 'contact_phone', operator: 'is-not-empty', values: [] },
    ]);

    expect(conditions.map((condition) => condition.id)).toEqual(['name_1', 'phone_1']);
  });

  it('新增条件默认选择第一个未使用字段', () => {
    const fieldId = firstAvailableTriggerFieldId(
      ['employee_name', 'contact_phone', 'department'],
      [
        { id: 'name', fieldId: 'employee_name', operator: 'equals', values: [] },
        { id: 'phone', fieldId: 'contact_phone', operator: 'equals', values: [] },
      ],
    );

    expect(fieldId).toBe('department');
  });
});
