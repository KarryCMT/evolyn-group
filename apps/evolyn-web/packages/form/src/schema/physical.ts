import type { LogicalStorageField, LogicalStorageFieldType } from '@evolyn.do/physical';
import type { FormItem, FormWidgetType } from './types';

/**
 * widgetName 已是表单协议中独立于 label 的稳定记录键；物理表阶段直接将其投影
 * 为 LogicalStorageField.id，禁止再创建第二份会随草稿漂移的字段标识。
 */
const STORAGE_TYPE_BY_WIDGET: Readonly<Partial<Record<FormWidgetType, LogicalStorageFieldType>>> = {
  text: 'shortText',
  textarea: 'longText',
  phone: 'shortText',
  number: 'decimal',
  datetime: 'datetime',
  radiogroup: 'shortText',
  combo: 'shortText',
  checkboxgroup: 'json',
  combocheck: 'json',
  user: 'relation',
  dept: 'relation',
  usergroup: 'json',
  deptgroup: 'json',
  image: 'json',
  upload: 'json',
  address: 'json',
  location: 'json',
  signature: 'json',
  subform: 'json',
  linkquery: 'json',
  linkfield: 'json',
  lookup: 'json',
  aggregation: 'json',
  sn: 'shortText',
  richtext: 'longText',
};

/**
 * 从发布快照或草稿项目生成可交由 Physical Engine 审阅的逻辑字段投影。
 * separator/button 仅负责布局与交互，不拥有记录值，因而不进入任何存储模型。
 */
export function projectPhysicalStorageFields(items: readonly FormItem[]): LogicalStorageField[] {
  return items.flatMap((item) => {
    const type = STORAGE_TYPE_BY_WIDGET[item.widget.type];
    if (!type) return [];
    return [
      {
        id: item.widget.widgetName,
        type,
        required: !item.widget.allowBlank,
      },
    ];
  });
}
