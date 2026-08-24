import type { FormDesignSelectorDefaultValue } from '../types';

/** 表单设计器支持的低代码控件标识。 */
export const formWidgetNames = {
  text: 'input',
  number: 'number',
  datetime: 'datetime',
  select: 'selectGroup',
  member: 'userGroup',
  department: 'deptGroup',
  subform: 'subforms',
} as const;

export const getFormSelectorValueKey = (widgetName?: string): 'users' | 'departs' | null => {
  if (widgetName === formWidgetNames.member) return 'users';
  if (widgetName === formWidgetNames.department) return 'departs';
  return null;
};

export const createFormSelectorDefaultValue = (
  widgetName?: string,
): FormDesignSelectorDefaultValue => {
  // 成员/部门字段默认值按控件标识生成，始终保存为 id 数组聚合对象。
  const valueKey = getFormSelectorValueKey(widgetName);
  return valueKey ? { [valueKey]: [] } : {};
};

export const isFormSubformWidget = (widgetName?: string) => {
  return widgetName === formWidgetNames.subform;
};

export const isFormSelectWidget = (widgetName?: string) => {
  return widgetName === formWidgetNames.select;
};

export const isFormTextWidget = (widgetName?: string) => {
  return widgetName === formWidgetNames.text;
};

/** @deprecated 旧插件模块迁移期间的兼容别名，新表单代码使用 Form 命名。 */
export const pluginWidgetNames = formWidgetNames;
/** @deprecated */
export const getPluginSelectorValueKey = getFormSelectorValueKey;
/** @deprecated */
export const createPluginSelectorDefaultValue = createFormSelectorDefaultValue;
/** @deprecated */
export const isPluginSubformWidget = isFormSubformWidget;
/** @deprecated */
export const isPluginSelectWidget = isFormSelectWidget;
/** @deprecated */
export const isPluginTextWidget = isFormTextWidget;
