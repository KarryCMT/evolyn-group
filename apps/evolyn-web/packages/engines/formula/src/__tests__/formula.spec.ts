import { describe, expect, it } from 'vitest';
import { collectFormulaDiagnostics } from '../analyzer';
import { FORMULA_FUNCTIONS } from '../catalog';
import { parseFormula } from '../parser';
import type { FormulaEditorField } from '../types';

const fields: FormulaEditorField[] = [
  { widgetName: 'amount', label: '金额', valueType: 'number' },
];

describe('Formula Engine', () => {
  it('解析稳定的字段引用和函数调用 AST', () => {
    const parsed = parseFormula('IF($amount# >= 100, SUM([1, 2]), 0)');

    expect(parsed.diagnostics).toEqual([]);
    expect(parsed.ast).toMatchObject({ kind: 'call', name: 'IF' });
  });

  it('以结构化诊断拒绝未知字段和不匹配的函数参数', () => {
    const diagnostics = collectFormulaDiagnostics('DATE($missing#, 9)', fields, FORMULA_FUNCTIONS);

    expect(diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ message: '未找到字段“missing”' }),
        expect.objectContaining({ message: expect.stringContaining('DATE 参数个数不符合要求') }),
      ]),
    );
  });
});
