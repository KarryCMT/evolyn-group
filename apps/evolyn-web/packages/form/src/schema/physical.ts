import type { LogicalStorageField, LogicalStorageFieldType } from '@evolyn.do/physical';
import type { FormItem, FormWidgetType } from './types';

/**
 * 字段身份投影（物理表存储 §4.1 契约冻结后口径）：v8 起以不可变 fieldId 为
 * LogicalStorageField.id（物理列名 f_<fieldId> 的推导来源）；v7 及更早的草稿
 * 读取时回落 widgetName 仅为展示，不参与物理列推导。
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
        id: item.widget.fieldId ?? item.widget.widgetName,
        type,
        required: !item.widget.allowBlank,
      },
    ];
  });
}
