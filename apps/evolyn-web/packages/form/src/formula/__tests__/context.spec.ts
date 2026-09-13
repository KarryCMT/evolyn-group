import { describe, expect, it } from 'vitest';
import { collectFormulaDiagnostics } from '../analyzer';
import { projectFormulaContext, projectSubformFormulaContext } from '../context';
import type { FormItem } from '../../schema/types';

function item(type: FormItem['widget']['type'], widgetName: string, label = widgetName): FormItem {
  return {
    label,
    description: '',
    labelHidden: false,
    lineWidth: 12,
    widget: {
      type,
      widgetName,
      enable: true,
      visible: true,
      allowBlank: true,
    } as FormItem['widget'],
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
      expect.objectContaining({
        widgetName: '_widget_name',
        valueType: 'text',
        displayType: '文本',
        formulaAllowed: true,
      }),
      expect.objectContaining({
        widgetName: '_widget_amount',
        valueType: 'number',
        displayType: '数字',
        formulaAllowed: true,
      }),
      expect.objectContaining({
        widgetName: '_widget_date',
        valueType: 'date',
        displayType: '时间戳',
        formulaAllowed: true,
      }),
      expect.objectContaining({
        widgetName: '_widget_tags',
        valueType: 'array',
        displayType: '数组',
        formulaAllowed: true,
      }),
      expect.objectContaining({
        widgetName: '_widget_owner',
        valueType: 'member',
        displayType: '成员',
        formulaAllowed: false,
      }),
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

  it('将金额字段作为带币种量纲的可计算变量投影', () => {
    const cny = item('money', '_widget_cny', '人民币金额');
    if (cny.widget.type !== 'money') throw new Error('expected money widget');
    cny.widget.currencyCode = 'CNY';
    const fields = projectFormulaContext([cny]);
    expect(fields).toEqual([
      expect.objectContaining({
        widgetName: '_widget_cny',
        displayType: '金额',
        formulaAllowed: true,
        currencyCode: 'CNY',
      }),
    ]);
  });

  it('将子表单子项投影为不可插入的数组变量', () => {
    const child = item('text', '_widget_product', '商品名称');
    const subform = {
      ...item('subform', '_widget_order_lines', '订单明细'),
      widget: {
        ...item('subform', '_widget_order_lines', '订单明细').widget,
        type: 'subform' as const,
        items: [child, item('separator', '_separator')],
        subformCreate: true,
        subformInsert: true,
        subformEdit: true,
        subformDelete: true,
        quickFill: false,
        pcStickyColumn: { enable: false, limit: 1 },
        mobileStickyColumn: { enable: false, limit: 1 },
        mobileViewStyle: 'vertical' as const,
        mobileSummaryFieldCount: 1,
      },
    } satisfies FormItem;

    expect(projectSubformFormulaContext([subform])).toEqual([
      {
        parentWidgetName: '_widget_order_lines',
        widgetName: '_widget_product',
        label: '订单明细.商品名称',
        valueType: 'array',
        displayType: '数组',
        formulaAllowed: false,
      },
    ]);
  });
});
