import {
  FORM_SCHEMA_VERSION,
  type FormDocument,
  type FormFieldGroup,
  type FormFieldPreset,
  type FormKind,
} from './types';

/** 字段预设仅定义 Schema 默认值；字段组件和属性面板由后续渲染器注册。 */
const presets: readonly FormFieldPreset[] = [
  { type: 'single-line-text', title: '单行文本', group: 'common', defaultLabel: '单行文本' },
  { type: 'multi-line-text', title: '多行文本', group: 'common', defaultLabel: '多行文本' },
  { type: 'number', title: '数字', group: 'common', defaultLabel: '数字' },
  { type: 'date-time', title: '日期时间', group: 'common', defaultLabel: '日期时间' },
  { type: 'radio-group', title: '单选按钮组', group: 'common', defaultLabel: '单选按钮组' },
  { type: 'checkbox-group', title: '复选框组', group: 'common', defaultLabel: '复选框组' },
  { type: 'select', title: '下拉框', group: 'common', defaultLabel: '下拉框' },
  { type: 'multi-select', title: '下拉复选框', group: 'common', defaultLabel: '下拉复选框' },
  { type: 'member', title: '成员单选', group: 'common', defaultLabel: '成员单选' },
  { type: 'members', title: '成员多选', group: 'common', defaultLabel: '成员多选' },
  { type: 'department', title: '部门单选', group: 'common', defaultLabel: '部门单选' },
  { type: 'departments', title: '部门多选', group: 'common', defaultLabel: '部门多选' },
  { type: 'divider', title: '分割线', group: 'common', defaultLabel: '分割线' },
  { type: 'tabs', title: '多标签页', group: 'common', defaultLabel: '多标签页' },
  { type: 'image', title: '图片', group: 'advanced', defaultLabel: '图片' },
  { type: 'attachment', title: '附件', group: 'advanced', defaultLabel: '附件' },
  { type: 'address', title: '地址', group: 'advanced', defaultLabel: '地址' },
  { type: 'location', title: '定位', group: 'advanced', defaultLabel: '定位' },
  { type: 'sub-form', title: '子表单', group: 'advanced', defaultLabel: '子表单' },
  { type: 'query', title: '查询', group: 'advanced', defaultLabel: '查询' },
  { type: 'data-selector', title: '选择数据', group: 'advanced', defaultLabel: '选择数据' },
  { type: 'signature', title: '手写签名', group: 'advanced', defaultLabel: '手写签名' },
  { type: 'serial-number', title: '流水号', group: 'advanced', defaultLabel: '流水号' },
  { type: 'mobile', title: '手机', group: 'advanced', defaultLabel: '手机' },
  { type: 'text-recognition', title: '文字识别', group: 'advanced', defaultLabel: '文字识别' },
  { type: 'button', title: '按钮', group: 'advanced', defaultLabel: '按钮' },
  { type: 'formula', title: '计算', group: 'advanced', defaultLabel: '计算' },
  { type: 'rich-text', title: '富文本', group: 'advanced', defaultLabel: '富文本' },
  { type: 'related-data', title: '关联数据', group: 'relation', defaultLabel: '关联数据' },
  { type: 'related-query', title: '关联查询', group: 'relation', defaultLabel: '关联查询' },
  { type: 'related-form', title: '关联表单', group: 'relation', defaultLabel: '关联表单' },
];

export const formFieldPresets = presets;

export const formFieldGroups: readonly FormFieldGroup[] = [
  { key: 'common', title: '常用', fields: presets.filter((field) => field.group === 'common') },
  { key: 'advanced', title: '高级', fields: presets.filter((field) => field.group === 'advanced') },
  { key: 'relation', title: '关联', fields: presets.filter((field) => field.group === 'relation') },
];

export function getFormFieldPreset(type: FormFieldPreset['type']) {
  return formFieldPresets.find((preset) => preset.type === type);
}

/** 新建文档不复用任何默认对象，确保多个设计器实例间不存在共享可变状态。 */
export function createEmptyFormDocument(kind: FormKind = 'standard'): FormDocument {
  return {
    version: FORM_SCHEMA_VERSION,
    kind,
    title: '未命名表单',
    fields: [],
    settings: {},
  };
}
