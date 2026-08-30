/**
 * 目标协议字段字典（TS 侧唯一事实，docs/低代码平台/表单设计器/目标协议字段字典.md）。
 *
 * 每种 widget.type 的属性集、取值约束与默认值以声明式描述（WIDGET_SPECS）维护，
 * validate.ts 按此执行严格校验；后端 Go 校验器（internal/platform/form）按同一张表
 * 镜像实现，保证前后端对同一 JSON 的校验结论一致（P1 验收条件）。
 * 修改本表必须同步：字段字典文档、Go 侧镜像表、发布白名单与测试用例。
 */

import type { FormItem, FormLayoutMode, FormWidgetOption, FormWidgetType } from './types';

/** 表单布局切换时批量投影到普通字段的 12 栅格宽度。 */
export const FORM_LAYOUT_LINE_WIDTH: Readonly<Record<FormLayoutMode, number>> = {
  normal: 12,
  'grid-2': 6,
  'grid-3': 4,
  'grid-4': 3,
};

/** 字段属性面板开放的宽度指令；1/2 与 2/3 按产品协议均写入 6。 */
export const FORM_FIELD_WIDTH_OPTIONS = [
  { label: '1/4', value: 3 },
  { label: '1/3', value: 4 },
  { label: '1/2', value: 6 },
  { label: '2/3', value: 6 },
  { label: '3/4', value: 9 },
  { label: '整行', value: 12 },
] as const;

/** 控件分组（字段字典 §4.2 的四组）。 */
export type FormWidgetGroupKey = 'basic' | 'orgfile' | 'relation' | 'interactive';

/** 属性取值类别：校验器按类别执行类型收窄与约束检查。 */
export type WidgetPropKind =
  | 'boolean'
  | 'string'
  | 'integer'
  | 'number'
  | 'enum'
  | 'stringArray'
  | 'options'
  | 'widgetItems'
  | 'stickyColumn'
  | 'expression'
  | 'snRule'
  | 'linkFilters'
  | 'linkSorts'
  | 'linkMappings'
  | 'buttonAction';

/** 单个控件属性的声明式约束。 */
export interface WidgetPropSpec {
  kind: WidgetPropKind;
  /** 必填键（缺省即非法）；非必填键缺省时的默认值见 default。 */
  required?: boolean;
  /** 枚举取值集合（kind=enum 时必填）。 */
  values?: readonly string[];
  /** 字符串长度上限（kind=string）。 */
  maxLen?: number;
  /** 数值下限（含）；integer/number 生效。 */
  min?: number;
  /** 数值上限（含）；integer/number 生效。 */
  max?: number;
  /** stringArray 条目上限。 */
  maxItems?: number;
}

export interface WidgetSpec {
  /** 控件中文名（素材面板/属性面板展示）。 */
  label: string;
  group: FormWidgetGroupKey;
  /** 值形态（运行时归类与提交校验使用，字段字典 §4）。 */
  valueKind: 'string' | 'number' | 'stringArray' | 'none' | 'object' | 'rows' | 'fileRefs';
  /** label 允许为空串（布局类控件）。 */
  labelOptional?: boolean;
  /** 专属属性表（不含 type/widgetName/enable/visible/allowBlank 四个公共必填键）。 */
  props: Readonly<Record<string, WidgetPropSpec>>;
}

/** 选项数组公共约束（字段字典 §3 开头约定）。 */
export const WIDGET_OPTION_LIMITS = {
  minItems: 1,
  maxItems: 200,
  textMaxLength: 100,
} as const;

/** 公共文本/数值上限（字段字典 §1/§2）。 */
export const FORM_PROTOCOL_LIMITS = {
  maxItems: 500,
  subformMaxItems: 200,
  maxLayouts: 50,
  maxTabsPerLayout: 20,
  labelMaxLength: 64,
  descriptionMaxLength: 500,
  widgetNameMaxLength: 64,
  lineWidthRange: { min: 1, max: 12 } as const,
  placeholderMaxLength: 100,
} as const;

/** 控件字典：27 种类型的完整声明。 */
export const WIDGET_SPECS: Readonly<Record<FormWidgetType, WidgetSpec>> = {
  text: {
    label: '单行文本',
    group: 'basic',
    valueKind: 'string',
    props: {
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      minLength: { kind: 'integer', min: 0, max: 1000 },
      maxLength: { kind: 'integer', min: 1, max: 1000 },
      format: { kind: 'enum', values: ['', 'email'] },
      defaultValue: { kind: 'string', maxLen: 1000 },
    },
  },
  textarea: {
    label: '多行文本',
    group: 'basic',
    valueKind: 'string',
    props: {
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      minLength: { kind: 'integer', min: 0, max: 2000 },
      maxLength: { kind: 'integer', min: 1, max: 2000 },
      autoHeight: { kind: 'boolean' },
      defaultValue: { kind: 'string', maxLen: 2000 },
    },
  },
  number: {
    label: '数字',
    group: 'basic',
    valueKind: 'number',
    props: {
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      min: { kind: 'number' },
      max: { kind: 'number' },
      precision: { kind: 'integer', min: 0, max: 8 },
      defaultValue: { kind: 'number' },
    },
  },
  datetime: {
    label: '日期时间',
    group: 'basic',
    valueKind: 'string',
    props: {
      format: { kind: 'enum', values: ['date', 'datetime', 'month', 'time'] },
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      defaultValue: { kind: 'string', maxLen: 32 },
    },
  },
  radiogroup: {
    label: '单选组',
    group: 'basic',
    valueKind: 'string',
    props: {
      options: { kind: 'options', required: true },
      layout: { kind: 'enum', values: ['vertical', 'horizontal'] },
      defaultValue: { kind: 'string', maxLen: WIDGET_OPTION_LIMITS.textMaxLength },
    },
  },
  checkboxgroup: {
    label: '复选组',
    group: 'basic',
    valueKind: 'stringArray',
    props: {
      options: { kind: 'options', required: true },
      layout: { kind: 'enum', values: ['vertical', 'horizontal'] },
      defaultValue: { kind: 'stringArray', maxItems: WIDGET_OPTION_LIMITS.maxItems },
    },
  },
  combo: {
    label: '下拉框',
    group: 'basic',
    valueKind: 'string',
    props: {
      options: { kind: 'options', required: true },
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      filterable: { kind: 'boolean' },
      defaultValue: { kind: 'string', maxLen: WIDGET_OPTION_LIMITS.textMaxLength },
    },
  },
  combocheck: {
    label: '下拉多选框',
    group: 'basic',
    valueKind: 'stringArray',
    props: {
      options: { kind: 'options', required: true },
      placeholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
      defaultValue: { kind: 'stringArray', maxItems: WIDGET_OPTION_LIMITS.maxItems },
    },
  },
  separator: {
    label: '分割线',
    group: 'basic',
    valueKind: 'none',
    labelOptional: true,
    props: {
      style: { kind: 'enum', values: ['solid', 'dashed'] },
    },
  },
  user: {
    label: '成员选择',
    group: 'orgfile',
    valueKind: 'string',
    props: {
      scope: { kind: 'enum', values: ['tenant', 'department'] },
      departments: { kind: 'stringArray', maxItems: 100 },
      defaultValue: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.widgetNameMaxLength },
    },
  },
  usergroup: {
    label: '成员多选',
    group: 'orgfile',
    valueKind: 'stringArray',
    props: {
      scope: { kind: 'enum', values: ['tenant', 'department'] },
      departments: { kind: 'stringArray', maxItems: 100 },
      defaultValue: { kind: 'stringArray', maxItems: 200 },
    },
  },
  dept: {
    label: '部门选择',
    group: 'orgfile',
    valueKind: 'string',
    props: {
      includeChildren: { kind: 'boolean' },
      defaultValue: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.widgetNameMaxLength },
    },
  },
  deptgroup: {
    label: '部门多选',
    group: 'orgfile',
    valueKind: 'stringArray',
    props: {
      includeChildren: { kind: 'boolean' },
      defaultValue: { kind: 'stringArray', maxItems: 200 },
    },
  },
  image: {
    label: '图片',
    group: 'orgfile',
    valueKind: 'fileRefs',
    props: {
      maxCount: { kind: 'integer', min: 1, max: 20 },
      maxSizeMB: { kind: 'integer', min: 1, max: 50 },
      accept: { kind: 'stringArray', maxItems: 20 },
    },
  },
  upload: {
    label: '附件',
    group: 'orgfile',
    valueKind: 'fileRefs',
    props: {
      maxCount: { kind: 'integer', min: 1, max: 20 },
      maxSizeMB: { kind: 'integer', min: 1, max: 100 },
      accept: { kind: 'stringArray', maxItems: 20 },
    },
  },
  address: {
    label: '地址',
    group: 'orgfile',
    valueKind: 'object',
    props: {
      level: { kind: 'enum', values: ['province', 'city', 'district', 'detail'] },
      detailPlaceholder: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.placeholderMaxLength },
    },
  },
  location: {
    label: '定位',
    group: 'orgfile',
    valueKind: 'object',
    props: {
      radius: { kind: 'number', min: 10, max: 5000 },
    },
  },
  signature: {
    label: '签名',
    group: 'orgfile',
    valueKind: 'string',
    props: {},
  },
  phone: {
    label: '手机号',
    group: 'orgfile',
    valueKind: 'string',
    props: {
      areaCode: { kind: 'string', maxLen: 5 },
      verification: { kind: 'boolean' },
    },
  },
  subform: {
    label: '子表单',
    group: 'relation',
    valueKind: 'rows',
    props: {
      items: { kind: 'widgetItems', required: true },
      subformCreate: { kind: 'boolean', required: true },
      subformInsert: { kind: 'boolean', required: true },
      subformEdit: { kind: 'boolean', required: true },
      subformDelete: { kind: 'boolean', required: true },
      quickFill: { kind: 'boolean', required: true },
      pcStickyColumn: { kind: 'stickyColumn', required: true },
      mobileStickyColumn: { kind: 'stickyColumn', required: true },
      mobileViewStyle: { kind: 'enum', values: ['vertical', 'horizontal'], required: true },
      mobileSummaryFieldCount: { kind: 'integer', min: 1, max: 5, required: true },
      minRowCount: { kind: 'integer', min: 0, max: 200 },
      maxRowCount: { kind: 'integer', min: 1, max: 200 },
    },
  },
  linkquery: {
    label: '关联查询',
    group: 'relation',
    valueKind: 'stringArray',
    props: {
      targetForm: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.widgetNameMaxLength },
      multiple: { kind: 'boolean' },
      filters: { kind: 'linkFilters', maxItems: 20 },
      sorts: { kind: 'linkSorts', maxItems: 3 },
    },
  },
  linkfield: {
    label: '关联字段',
    group: 'relation',
    valueKind: 'string',
    props: {
      targetForm: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.widgetNameMaxLength },
      displayFields: { kind: 'stringArray', maxItems: 20 },
    },
  },
  lookup: {
    label: '数据联动',
    group: 'relation',
    valueKind: 'string',
    props: {
      targetForm: { kind: 'string', maxLen: FORM_PROTOCOL_LIMITS.widgetNameMaxLength },
      mappings: { kind: 'linkMappings', maxItems: 20 },
    },
  },
  aggregation: {
    label: '聚合计算',
    group: 'relation',
    valueKind: 'number',
    props: {
      expression: { kind: 'expression' },
      precision: { kind: 'integer', min: 0, max: 8 },
      displayMode: { kind: 'enum', values: ['plain', 'percent'] },
    },
  },
  sn: {
    label: '流水号',
    group: 'relation',
    valueKind: 'string',
    props: {
      rule: { kind: 'snRule' },
    },
  },
  richtext: {
    label: '富文本',
    group: 'interactive',
    valueKind: 'string',
    props: {
      toolbar: { kind: 'stringArray', maxItems: 30 },
    },
  },
  button: {
    label: '按钮',
    group: 'interactive',
    valueKind: 'none',
    labelOptional: true,
    props: {
      text: { kind: 'string', maxLen: 32 },
      action: { kind: 'buttonAction' },
    },
  },
};

/** 分组展示序（素材面板分组标题）。 */
export const WIDGET_GROUP_META: ReadonlyArray<{ key: FormWidgetGroupKey; title: string }> = [
  { key: 'basic', title: '基础字段' },
  { key: 'orgfile', title: '组织与文件' },
  { key: 'relation', title: '业务与关联' },
  { key: 'interactive', title: '富交互' },
];

/** 按类型取控件中文名。 */
export function widgetTypeLabel(type: string): string {
  const spec = (WIDGET_SPECS as Record<string, WidgetSpec | undefined>)[type];
  return spec ? spec.label : type;
}

let widgetNameSeed = 0;

/** 生成稳定字段键：`_widget_` 前缀 + 时间戳 + 递增序号（字典 1.4 的设计器约定）。 */
export function generateWidgetName(): string {
  widgetNameSeed = (widgetNameSeed + 1) % 10000;
  return `_widget_${Date.now()}${String(widgetNameSeed).padStart(4, '0')}`;
}

/** 生成表单级布局与标签页稳定键；二者与 widgetName 共享顶层引用命名空间。 */
export function generateLayoutName(): string {
  widgetNameSeed = (widgetNameSeed + 1) % 10000;
  return `_layout_${Date.now()}${String(widgetNameSeed).padStart(4, '0')}`;
}

export function generateTabName(): string {
  widgetNameSeed = (widgetNameSeed + 1) % 10000;
  return `_tab_${Date.now()}${String(widgetNameSeed).padStart(4, '0')}`;
}

const defaultOptions = (): FormWidgetOption[] => [
  { label: '选项1', value: '选项1' },
  { label: '选项2', value: '选项2' },
];

/** 按类型生成新字段项（素材面板点击/拖入时使用；默认值与字段字典逐项一致）。 */
export function createWidgetItem(type: FormWidgetType): FormItem {
  const spec = WIDGET_SPECS[type];
  const widget: Record<string, unknown> = {
    type,
    widgetName: generateWidgetName(),
    enable: true,
    visible: true,
    allowBlank: true,
  };
  // 仅选项类控件带必填 options；数值上限类属性缺省即「未启用」，不预写。
  if (spec.props.options?.required) widget.options = defaultOptions();
  if (type === 'subform') {
    Object.assign(widget, {
      items: [],
      subformCreate: true,
      subformInsert: true,
      subformEdit: true,
      subformDelete: true,
      quickFill: true,
      pcStickyColumn: { enable: true, limit: 1 },
      mobileStickyColumn: { enable: false, limit: 1 },
      mobileViewStyle: 'vertical',
      mobileSummaryFieldCount: 3,
    });
  }
  return {
    widget: widget as FormItem['widget'],
    label: spec.label,
    description: '',
    labelHidden: false,
    lineWidth: FORM_PROTOCOL_LIMITS.lineWidthRange.max,
  };
}

/** 深拷贝字段项并换新 widgetName（设计器「复制字段」动作）。 */
export function copyWidgetItem(item: FormItem): FormItem {
  const clone = JSON.parse(JSON.stringify(item)) as FormItem;
  clone.widget.widgetName = generateWidgetName();
  clone.label = `${item.label} copy`;
  return clone;
}
