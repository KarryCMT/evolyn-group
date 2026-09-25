import type { FormTriggerAction, FormTriggerCondition } from './types';

/**
 * 表单触发动作按动作类型唯一；历史数据若包含重复项，保留最先配置的一项。
 */
export function uniqueFormTriggerActions(
  actions: readonly FormTriggerAction[],
): FormTriggerAction[] {
  const selectedTypes = new Set<FormTriggerAction['type']>();
  return actions.flatMap((action) => {
    if (selectedTypes.has(action.type)) return [];
    selectedTypes.add(action.type);
    return [{ ...action, fieldIds: action.fieldIds ? [...action.fieldIds] : undefined }];
  });
}

/**
 * 触发条件按字段唯一；空字段不参与去重，避免误删尚未完成的历史配置。
 */
export function uniqueFormTriggerConditions(
  conditions: readonly FormTriggerCondition[],
): FormTriggerCondition[] {
  const selectedFieldIds = new Set<string>();
  return conditions.flatMap((condition) => {
    if (condition.fieldId && selectedFieldIds.has(condition.fieldId)) return [];
    if (condition.fieldId) selectedFieldIds.add(condition.fieldId);
    return [{ ...condition, values: [...condition.values] }];
  });
}

/** 返回尚未被其他条件使用的第一个字段。 */
export function firstAvailableTriggerFieldId(
  fieldIds: readonly string[],
  conditions: readonly FormTriggerCondition[],
): string | undefined {
  const selectedFieldIds = new Set(
    conditions.map((condition) => condition.fieldId).filter(Boolean),
  );
  return fieldIds.find((fieldId) => !selectedFieldIds.has(fieldId));
}
