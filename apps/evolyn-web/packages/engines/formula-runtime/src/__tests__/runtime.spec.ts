import { parseFormula } from '@evolyn.do/formula';
import { Numeric, NumericError } from '@evolyn.do/numeric';
import { describe, expect, it } from 'vitest';

import { evaluateFormula } from '../evaluator';
import { FormulaError } from '../errors';

/** 解析并求值：诊断即错 */
function run(source: string, data: Record<string, unknown> = {}) {
  const { diagnostics, ast } = parseFormula(source);
  expect(diagnostics, `公式应可解析: ${source}`).toHaveLength(0);
  expect(ast).toBeDefined();
  return evaluateFormula(ast!, {
    source,
    resolveField: (name) => (name in data ? (data[name] as never) : undefined),
  });
}

/** 数值断言：serialize 为 canonical decimal string */
function expectNumeric(value: unknown, expected: string | null) {
  expect(value).toBeInstanceOf(Numeric);
  expect((value as Numeric).serialize()).toBe(expected);
}

// ---- 设计 §19：数值执行全链路 Numeric ----

describe('数值执行', () => {
  it('字段乘法链：quantity * unitPrice * taxRate（§19 示例）', () => {
    const value = run('$quantity# * $unitPrice# * $taxRate#', {
      quantity: '2',
      unitPrice: '10.05',
      taxRate: '0.13',
    });
    expectNumeric(value, '2.613');
  });

  it('0.1 + 0.2 === 0.3（禁止 Number 中转）', () => {
    expectNumeric(run('0.1 + 0.2'), '0.3');
  });

  it('literal 原文保留（§20）：123.4500 尾零语义进入运行时', () => {
    const value = run('123.4500 * 1');
    expectNumeric(value, '123.45');
    expect(run('123.4500 * 1')).toBeInstanceOf(Numeric);
  });

  it('除法与舍入：100 / 3 四舍五入两位', () => {
    expectNumeric(run('ROUND(100 / 3, 2)'), '33.33');
  });

  it('除零 → FormulaError #DIV/0!（§13 映射）', () => {
    try {
      run('1 / 0');
      throw new Error('应抛出');
    } catch (err) {
      expect(err).toBeInstanceOf(FormulaError);
      const fe = err as FormulaError;
      expect(fe.code).toBe('DIVISION_BY_ZERO');
      expect(fe.alias).toBe('#DIV/0!');
    }
  });
});

// ---- 字段解析与空值传播 ----

describe('字段与空值', () => {
  it('未提供字段 → null 传播（平台默认 NULL 策略）', () => {
    expectNumeric(run('$missing# + 1'), null);
  });

  it('字段值为 null → 传播', () => {
    expectNumeric(run('$a# + $b#', { a: null, b: '5' }), null);
  });

  it('number 型字段兼容入口', () => {
    expectNumeric(run('$n# * 2', { n: 0.1 }), '0.2');
  });
});

// ---- 函数注册表（§21 映射） ----

describe('函数执行', () => {
  it('SUM/AVG/MIN/MAX/PRODUCT 聚合（数组语义）', () => {
    expectNumeric(run('SUM([1.1, 2.2, 3.3])'), '6.6');
    expectNumeric(run('AVERAGE([1, 2, 3])'), '2');
    expectNumeric(run('MIN([3, 1, 2])'), '1');
    expectNumeric(run('MAX([3, 1, 2])'), '3');
    expectNumeric(run('PRODUCT([2, 3, 4])'), '24');
  });

  it('聚合忽略 null 元素（SQL 语义）', () => {
    expectNumeric(run('SUM([$a#, 2])', { a: null }), '2');
  });

  it('ABS/FLOOR/CEILING/INT/MOD/POWER/SQRT', () => {
    expectNumeric(run('ABS(-5.5)'), '5.5');
    expectNumeric(run('FLOOR(-2.1)'), '-3');
    expectNumeric(run('CEILING(2.1)'), '3');
    expectNumeric(run('INT(2.9)'), '2');
    expectNumeric(run('MOD(10, 3)'), '1');
    expectNumeric(run('POWER(2, 10)'), '1024');
    expectNumeric(run('ROUND(SQRT(2), 4)'), '1.4142');
  });

  it('SUMIF 条件语义：数值比较 + 求和范围', () => {
    // range=[10,20,30]，条件 >15 → 20+30
    expectNumeric(run('SUMIF([$a#, $b#, $c#], ">15")', { a: '10', b: '20', c: '30' }), '50');
    // 第三参指定求和范围
    expectNumeric(
      run('SUMIF([$a#, $b#], ">10", [$x#, $y#])', { a: '5', b: '20', x: '100', y: '200' }),
      '200',
    );
  });

  it('SUMPRODUCT 对位乘积求和（子表金额×数量场景）', () => {
    expectNumeric(run('SUMPRODUCT([$q#, $p#])', { q: ['2', '3'], p: ['10.5', '4'] }), '33');
  });

  it('逻辑函数 IF/AND/OR/NOT', () => {
    expect(run('IF(1 > 2, "yes", "no")')).toBe('no');
    expect(run('AND(1 > 0, 2 > 1)')).toBe(true);
    expect(run('OR(1 > 2, 2 > 1)')).toBe(true);
    expect(run('NOT(1 > 2)')).toBe(true);
  });

  it('字段公式文本函数保持空值与 Unicode 语义', () => {
    expect(run('CONCATENATE($code#, "-", UPPER($model#))', { code: 'P100', model: 'blue' })).toBe('P100-BLUE');
    expectNumeric(run('LEN("灵衍云")'), '3');
    expect(run('TRIM("  A   B  ")')).toBe('A B');
    expect(run('ISBLANK($missing#)')).toBe(true);
  });

  it('未实现函数 → FUNCTION_NOT_IMPLEMENTED', () => {
    try {
      run('CONCAT("a", "b")');
      throw new Error('应抛出');
    } catch (err) {
      expect(err).toBeInstanceOf(FormulaError);
      expect((err as FormulaError).code).toBe('FUNCTION_NOT_IMPLEMENTED');
    }
  });
});

// ---- 比较与文本 ----

describe('比较运算', () => {
  it('数值比较走数值谓词（等值判定精度安全）', () => {
    expect(run('0.1 + 0.2 == 0.3')).toBe(true);
    expect(run('2.50 > 2.5')).toBe(false);
    expect(run('"a" == "a"')).toBe(true);
  });

  it('空值参与比较为假（SQL 语义）', () => {
    expect(run('$x# > 0', { x: null })).toBe(false);
    expect(run('$x# == 0', { x: null })).toBe(false);
  });
});

// ---- NumericError → FormulaError 稳定映射 ----

describe('错误映射', () => {
  it('INVALID_NUMBER → #VALUE!', () => {
    try {
      run('SQRT("abc")');
      throw new Error('应抛出');
    } catch (err) {
      expect((err as FormulaError).alias).toBe('#VALUE!');
    }
  });

  it('精度护栏透传 PRECISION_EXCEEDED', () => {
    expect(() => run('123456789012345678901234567890123456789012345 * 1')).toThrow(FormulaError);
    expect(NumericError).toBeDefined();
  });
});
