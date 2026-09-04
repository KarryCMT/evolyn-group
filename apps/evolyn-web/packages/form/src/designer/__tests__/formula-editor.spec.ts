import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import {
  collectFormulaDiagnostics,
  FORMULA_FUNCTIONS,
  type FormulaEditorField,
} from '../formula-editor';
import FormulaEditor from '../FormulaEditor.vue';

const fields: FormulaEditorField[] = [
  { widgetName: '_widget_name', label: '姓名', valueType: 'text', displayType: '文本' },
  { widgetName: '_widget_amount', label: '金额', valueType: 'number', displayType: '数字' },
];

describe('formula editor diagnostics', () => {
  it('已注册字段与函数不产生前端诊断', () => {
    const diagnostics = collectFormulaDiagnostics(
      'IF($_widget_amount# > 0, CONCATENATE($_widget_name#), "")',
      fields,
      FORMULA_FUNCTIONS,
    );

    expect(diagnostics).toEqual([]);
  });

  it('标记缺失的函数右括号', () => {
    const diagnostics = collectFormulaDiagnostics(
      'IF($_widget_amount# > 0',
      fields,
      FORMULA_FUNCTIONS,
    );

    expect(diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ severity: 'error', message: '函数 IF 缺少右括号' }),
      ]),
    );
  });

  it('严格标记未收录的字段和函数', () => {
    const diagnostics = collectFormulaDiagnostics(
      'UNKNOWN($_widget_missing#)',
      fields,
      FORMULA_FUNCTIONS,
    );

    expect(diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ severity: 'error', message: '未找到字段“_widget_missing”' }),
        expect.objectContaining({ severity: 'error', message: '未收录函数“UNKNOWN”' }),
      ]),
    );
  });
});

describe('FormulaEditor', () => {
  it('公式正文保留稳定 key，但在编辑区显示字段标签', async () => {
    const wrapper = mount(FormulaEditor, {
      props: {
        modelValue: 'CONCATENATE($_widget_name#)',
        fields,
        functions: FORMULA_FUNCTIONS,
      },
    });
    await nextTick();

    const fieldChip = wrapper.find('.cm-formula-field-chip');
    expect(fieldChip.exists()).toBe(true);
    expect(fieldChip.text()).toBe('姓名');
    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });
});
