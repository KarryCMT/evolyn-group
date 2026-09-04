import { describe, expect, it } from 'vitest';
import { collectFormulaDiagnostics } from '../analyzer';
import { projectFormulaContext } from '../context';
import type { FormItem } from '../../schema/types';

function item(type: FormItem['widget']['type'], widgetName: string, label = widgetName): FormItem {
  return {
    label,
    description: '',
    labelHidden: false,
    lineWidth: 12,
    widget: { type, widgetName, enable: true, visible: true, allowBlank: true } as FormItem['widget'],
  };
}

describe('projectFormulaContext', () => {
  it('按控件真实值形态投影变量，并排除无值控件', () => {
    const fields = projectFormulaContext([
      item('text', '_widget_name', '姓名'),
      item('number', '_widget_amount', '金额'),
      item('datetime', '_widget_date', '日期'),
      item('checkboxgroup', '_widget_tags', '标签'),
      item('user', '_widget_owner', '负责人'),
      item('separator', '_separator'),
      item('button', '_button'),
    ]);

    expect(fields).toEqual([
      expect.objectContaining({ widgetName: '_widget_name', valueType: 'text', displayType: '文本', formulaAllowed: true }),
      expect.objectContaining({ widgetName: '_widget_amount', valueType: 'number', displayType: '数字', formulaAllowed: true }),
      expect.objectContaining({ widgetName: '_widget_date', valueType: 'date', displayType: '时间戳', formulaAllowed: true }),
      expect.objectContaining({ widgetName: '_widget_tags', valueType: 'array', displayType: '数组', formulaAllowed: true }),
      expect.objectContaining({ widgetName: '_widget_owner', valueType: 'member', displayType: '成员', formulaAllowed: false }),
    ]);
  });

  it('拒绝手动粘贴当前 DSL 未支持的结构字段', () => {
    const fields = projectFormulaContext([item('user', '_widget_owner', '负责人')]);
    expect(collectFormulaDiagnostics('$_widget_owner# == 1', fields)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ message: '字段“负责人”的类型暂不支持参与公式计算' }),
      ]),
    );
  });
});
