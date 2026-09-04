import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { collectFormulaDiagnostics } from '../analyzer';
import { FORMULA_FUNCTIONS } from '../catalog';
import FormulaFunctionLibrary from '../FormulaFunctionLibrary.vue';
import { parseFormula } from '../parser';
import type { FormulaEditorField } from '../types';

const fields: FormulaEditorField[] = [
  { widgetName: '_widget_name', label: '姓名', valueType: 'text', displayType: '文本' },
  { widgetName: '_widget_amount', label: '金额', valueType: 'number', displayType: '数字' },
];

describe('formula catalog', () => {
  it('覆盖规范中的五类函数及全部已登记函数名', () => {
    const names = new Set(FORMULA_FUNCTIONS.map((item) => item.name));
    expect(FORMULA_FUNCTIONS).toHaveLength(87);
    [
      'AND',
      'FALSE',
      'IF',
      'IFS',
      'NOT',
      'OR',
      'TRUE',
      'XOR',
      'CONCATENATE',
      'CHAR',
      'EXACT',
      'IP',
      'ISEMPTY',
      'JOIN',
      'LEFT',
      'LEN',
      'LOWER',
      'MID',
      'REPLACE',
      'REPT',
      'RIGHT',
      'RMBCAP',
      'SEARCH',
      'SPLIT',
      'TEXT',
      'TRIM',
      'UNION',
      'UPPER',
      'VALUE',
      'ABS',
      'AVERAGE',
      'CEILING',
      'COS',
      'COT',
      'COUNT',
      'COUNTIF',
      'FIXED',
      'FLOOR',
      'INT',
      'LARGE',
      'LOG',
      'MAX',
      'MIN',
      'MOD',
      'POWER',
      'PRODUCT',
      'RAND',
      'ROUND',
      'SIN',
      'SMALL',
      'SQRT',
      'SUM',
      'SUMIF',
      'SUMIFS',
      'SUMPRODUCT',
      'TAN',
      'DATE',
      'DATEDIF',
      'DATEDELTA',
      'DAY',
      'DAYS',
      'DAYS360',
      'HOUR',
      'ISOWEEKNUM',
      'MINUTE',
      'MONTH',
      'NETWORKDAYS',
      'NOW',
      'SECOND',
      'SYSTIME',
      'TIME',
      'TIMESTAMP',
      'TODAY',
      'WEEKDAY',
      'WEEKNUM',
      'WORKDAY',
      'YEAR',
      'DISTANCE',
      'GETUSERNAME',
      'INDEX',
      'MAPX',
      'RECNO',
      'TEXTDEPT',
      'TEXTLOCATION',
      'TEXTPHONE',
      'TEXTUSER',
      'UUID',
    ].forEach((name) => expect(names.has(name)).toBe(true));
  });
});

describe('formula parser and analyzer', () => {
  it('解析字段、数组、运算符和嵌套函数', () => {
    const parsed = parseFormula('IF($_widget_amount# >= 100, SUM([1, 2, 3]), 0)');
    expect(parsed.diagnostics).toEqual([]);
    expect(parsed.ast).toMatchObject({ kind: 'call', name: 'IF' });
  });

  it('对文档中的可变 DATE 参数数量进行严格检查', () => {
    expect(collectFormulaDiagnostics('DATE(2026, 9)', fields, FORMULA_FUNCTIONS)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ message: expect.stringContaining('DATE 参数个数不符合要求') }),
      ]),
    );
    expect(collectFormulaDiagnostics('DATE(2026, 9, 4)', fields, FORMULA_FUNCTIONS)).toEqual([]);
  });

  it('拒绝未知函数、未知字段和不完整的字符串', () => {
    const diagnostics = collectFormulaDiagnostics(
      'UNKNOWN($_widget_lost#, "未闭合)',
      fields,
      FORMULA_FUNCTIONS,
    );
    expect(diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ message: '未收录函数“UNKNOWN”' }),
        expect.objectContaining({ message: '未找到字段“_widget_lost”' }),
        expect.objectContaining({ message: '字符串缺少结束引号' }),
      ]),
    );
  });

  it('为缺少运算符的相邻表达式提供可展示的语法诊断', () => {
    const diagnostics = collectFormulaDiagnostics(
      '$_widget_name# $_widget_amount#',
      fields,
      FORMULA_FUNCTIONS,
    );

    expect(diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          severity: 'error',
          message: '此处不应继续出现公式内容',
        }),
      ]),
    );
  });
});

describe('FormulaFunctionLibrary', () => {
  it('支持函数分组展开收起，并突出当前选中函数', async () => {
    const wrapper = mount(FormulaFunctionLibrary, { props: { functions: FORMULA_FUNCTIONS } });
    const headings = wrapper.findAll('.formula-function-library__group-heading');
    const commonHeading = headings.find((item) => item.text().includes('常用函数'));
    const mathHeading = headings.find((item) => item.text().includes('数学函数'));

    expect(commonHeading?.attributes('aria-expanded')).toBe('true');
    await commonHeading?.trigger('click');
    expect(commonHeading?.attributes('aria-expanded')).toBe('false');

    await mathHeading?.trigger('click');
    expect(mathHeading?.attributes('aria-expanded')).toBe('true');
    const abs = wrapper
      .findAll('.formula-function-library__group-items button')
      .find((item) => item.text().startsWith('ABS'));
    expect(abs?.classes()).toContain('is-active');
  });
});
