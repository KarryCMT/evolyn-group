import type { FormDesignFieldDefaultValue, FormDesignTemplateField } from '../../types';

/** 子表单目前支持的列赋值模式。 */
export type FormSubformColumnMode = 'custom' | 'depend';

/** 子表单列绑定配置，赋值模式作用于整列。 */
export interface FormSubformColumnBinding {
  field: string;
  widgetName: string;
  fieldDataType: string;
  mode: FormSubformColumnMode;
}

/** 子表单自定义值单元格。 */
export interface FormSubformCustomCellBinding {
  sourceData: FormDesignFieldDefaultValue | null;
}

/** 子表单字段值引用单元格。 */
export interface FormSubformDependCellBinding {
  linkNodeId: string;
  dependField: string;
  beforeDependField: string;
  dependParentKey: string | null;
}

/** 子表单单元格绑定配置。 */
export type FormSubformCellBinding = FormSubformCustomCellBinding | FormSubformDependCellBinding;

/** 子表单单行绑定配置。 */
export interface FormSubformRowBinding {
  /** 稳定行标识只用于配置编辑，不应传给插件执行参数。 */
  rowKey: string;
  cells: Record<string, FormSubformCellBinding>;
}

/** 子表单逐行赋值配置。 */
export interface FormSubformBindingValue {
  assignMode: 'row';
  valueVersion: 1;
  columns: FormSubformColumnBinding[];
  rows: FormSubformRowBinding[];
}

/** 字段值选择插槽参数，由业务侧提供具体的流程字段树组件。 */
export interface FormSubformDependSlotProps {
  cell: FormSubformDependCellBinding;
  childField: FormDesignTemplateField;
  column: FormSubformColumnBinding;
  row: FormSubformRowBinding;
  rowIndex: number;
  updateCell: (value: FormSubformDependCellBinding) => void;
}
