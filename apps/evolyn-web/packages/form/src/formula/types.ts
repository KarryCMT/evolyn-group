/** 公式 DSL 的值类型；unknown 表示字段或函数的结果需在运行时解析。 */
export type FormulaValueType =
  | 'number'
  | 'text'
  | 'boolean'
  | 'date'
  | 'array'
  | 'member'
  | 'members'
  | 'department'
  | 'departments'
  | 'location'
  | 'rows'
  | 'unknown';

/** 函数目录分类，与公式面板中的分类导航一一对应。 */
export type FormulaFunctionCategory = 'logic' | 'text' | 'math' | 'date' | 'advanced';

export interface FormulaEditorField {
  widgetName: string;
  label: string;
  /** 公式类型系统使用的值类型；不能运算的结构值显式标为 unknown。 */
  valueType: FormulaValueType;
  /** 面板展示类型，如「成员」「数组」；不参与公式类型判断。 */
  displayType?: string;
  /** 当前公式 DSL 是否允许将该字段作为直接操作数插入。 */
  formulaAllowed?: boolean;
}

/**
 * 可支持多个固定参数数量，例如 DATE 可传入 1、3 或 6 个参数。
 * min/max 用于连续区间，arity 用于离散参数数量。
 */
export interface FormulaFunction {
  name: string;
  category: FormulaFunctionCategory;
  description: string;
  syntax: string;
  returnType: FormulaValueType;
  minArgs?: number;
  maxArgs?: number;
  arity?: readonly number[];
}

/** 兼容编辑器的轻量函数契约。 */
export type FormulaEditorFunction = FormulaFunction;

/** 由字段或函数库发起、按光标位置写入编辑器的命令。 */
export interface FormulaEditorInsertion {
  id: number;
  text: string;
  cursorOffset?: number;
}

export interface FormulaDiagnostic {
  from: number;
  to: number;
  severity: 'error' | 'warning';
  message: string;
}
