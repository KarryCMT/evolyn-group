import type { FormItem, FormWidgetType } from '../schema/types';
import type { FormulaEditorField, FormulaValueType } from './types';

/**
 * 控件值形态到公式变量的唯一前端投影表。
 *
 * 这里刻意保留成员、部门、行集合等非标量类型，供变量面板正确展示；是否可作为
 * 当前公式 DSL 的操作数由 formulaAllowed 明确控制，禁止把复杂对象静默当文本。
 */
export interface FormulaWidgetVariableMeta {
  valueType: FormulaValueType;
  displayType: string;
  formulaAllowed: boolean;
}

const UNKNOWN_VARIABLE_META: FormulaWidgetVariableMeta = {
  valueType: 'unknown',
  displayType: '未支持',
  formulaAllowed: false,
};

/** 控件类型 → 公式变量类型；须与 Go 侧 form/formula/context.go 同步。 */
export const FORMULA_WIDGET_VARIABLE_TYPES: Readonly<
  Partial<Record<FormWidgetType, FormulaWidgetVariableMeta>>
> = {
  text: { valueType: 'text', displayType: '文本', formulaAllowed: true },
  textarea: { valueType: 'text', displayType: '文本', formulaAllowed: true },
  phone: { valueType: 'text', displayType: '文本', formulaAllowed: true },
  number: { valueType: 'number', displayType: '数字', formulaAllowed: true },
  datetime: { valueType: 'date', displayType: '时间戳', formulaAllowed: true },
  radiogroup: { valueType: 'text', displayType: '文本', formulaAllowed: true },
  combo: { valueType: 'text', displayType: '文本', formulaAllowed: true },
  checkboxgroup: { valueType: 'array', displayType: '数组', formulaAllowed: true },
  combocheck: { valueType: 'array', displayType: '数组', formulaAllowed: true },
  user: { valueType: 'member', displayType: '成员', formulaAllowed: false },
  usergroup: { valueType: 'members', displayType: '成员数组', formulaAllowed: false },
  dept: { valueType: 'department', displayType: '部门', formulaAllowed: false },
  deptgroup: { valueType: 'departments', displayType: '部门数组', formulaAllowed: false },
  location: { valueType: 'location', displayType: '位置', formulaAllowed: false },
  subform: { valueType: 'rows', displayType: '行集合', formulaAllowed: false },
};

/**
 * 从当前（包括尚未保存的）表单草稿生成公式编辑上下文。
 * separator/button 没有业务值，不能作为变量；其余未开放控件保留为「未支持」
 * 条目，让设计者看见字段存在但不能误以为可以安全计算。
 */
export function projectFormulaContext(items: readonly FormItem[]): FormulaEditorField[] {
  return items
    .filter((item) => item.widget.type !== 'separator' && item.widget.type !== 'button')
    .map((item) => {
      const meta = FORMULA_WIDGET_VARIABLE_TYPES[item.widget.type] ?? UNKNOWN_VARIABLE_META;
      return {
        widgetName: item.widget.widgetName,
        label: item.label || item.widget.widgetName,
        valueType: meta.valueType,
        displayType: meta.displayType,
        formulaAllowed: meta.formulaAllowed,
      };
    });
}
