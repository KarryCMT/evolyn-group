import { describe, expect, it } from 'vitest';

import {
  createNumericRuntime,
  EmptyValuePolicy,
  numeric,
  NumericError,
  RoundingMode,
} from '../index';

/** 断言抛出指定稳定码的 NumericError */
function expectCode(code: string, fn: () => unknown) {
  try {
    fn();
  } catch (err) {
    expect(err).toBeInstanceOf(NumericError);
    expect((err as NumericError).code).toBe(code);
    return;
  }
  throw new Error(`expected NumericError ${code}, but no error thrown`);
}

// ---- §51 基础 ----

describe('基础运算', () => {
  it('字符串 decimal 直通（0.1 + 0.2 === 0.3）', () => {
    expect(numeric.add('0.1', '0.2').serialize()).toBe('0.3');
  });

  it('大整数不丢精度（设计 §1 的核心诉求）', () => {
    expect(numeric.add('9007199254740993', '1').serialize()).toBe('9007199254740994');
  });

  it('链式 API：含税金额（设计 §9 示例）', () => {
    expect(numeric.of('1000.00').multiply('0.13').round(2).toFixed(2)).toBe('130.00');
  });

  it('减乘模与取负/绝对值', () => {
    expect(numeric.subtract('1.5', '0.75').serialize()).toBe('0.75');
    // 尾零语义（§48/§49）：runtime canonical 输出最短精确表示（decimal.js
    // 构造即去无意义尾零）；定长尾零走 toFixed（展示）或字段 Serialization
    // Policy（存储 scale），三种语义分离
    expect(numeric.multiply('1.10', '1.10').serialize()).toBe('1.21');
    expect(numeric.multiply('1.10', '1.10').toFixed(4)).toBe('1.2100');
    expect(numeric.mod('10', '3').serialize()).toBe('1');
    expect(numeric.negate('5.5').serialize()).toBe('-5.5');
    expect(numeric.abs('-5.5').serialize()).toBe('5.5');
  });

  it('number 兼容入口可用（有限值）', () => {
    expect(numeric.of(0.1).add(0.2).serialize()).toBe('0.3');
  });
});

// ---- §51 除法 ----

describe('除法', () => {
  it('除零抛 DIVISION_BY_ZERO', () => {
    expectCode('DIVISION_BY_ZERO', () => numeric.divide('1', '0'));
    expectCode('DIVISION_BY_ZERO', () => numeric.of('10').mod('0'));
  });

  it('除法不提前截断：100/3 保留高精度，round 后落位（设计 §10）', () => {
    const raw = numeric.divide('100', '3').round(2);
    expect(raw.serialize()).toBe('33.33');
    // 未 round 直接 serialize 超 maxScale 抛错，防止隐式截断
    expectCode('SCALE_EXCEEDED', () => numeric.divide('100', '3').serialize());
  });
});

// ---- §51 正负数 ----

describe('正负数', () => {
  it('符号语义与 -0 规范化', () => {
    expect(numeric.of('-0').serialize()).toBe('0');
    expect(numeric.multiply('-3', '-4').serialize()).toBe('12');
    expect(numeric.subtract('-5', '3').serialize()).toBe('-8');
  });

  it('负数舍入方向', () => {
    expect(numeric.round('-2.5', 0, RoundingMode.HALF_UP).serialize()).toBe('-3');
    expect(numeric.round('-2.5', 0, RoundingMode.HALF_EVEN).serialize()).toBe('-2');
    expect(numeric.round('-2.5', 0, RoundingMode.UP).serialize()).toBe('-3');
    expect(numeric.round('-2.5', 0, RoundingMode.DOWN).serialize()).toBe('-2');
    expect(numeric.ceil('-2.1').serialize()).toBe('-2');
    expect(numeric.floor('-2.1').serialize()).toBe('-3');
  });
});

// ---- §51 极值 ----

describe('极值', () => {
  it('禁止 NaN / Infinity 输入（设计 §14）', () => {
    expectCode('NON_FINITE_NUMBER', () => numeric.of('NaN'));
    expectCode('NON_FINITE_NUMBER', () => numeric.of('Infinity'));
    expectCode('NON_FINITE_NUMBER', () => numeric.of(Number.POSITIVE_INFINITY));
    expectCode('NON_FINITE_NUMBER', () => numeric.of(-Infinity));
  });

  it('指数形态输入合法，canonical 输出展开（设计 §49）', () => {
    expect(numeric.of('1e+3').serialize()).toBe('1000');
    expect(numeric.multiply('1e+20', '1e+20').serialize()).toBe(
      '10000000000000000000000000000000000000000',
    );
  });

  it('精度护栏：有效数字超 40 位抛 PRECISION_EXCEEDED', () => {
    expectCode('PRECISION_EXCEEDED', () =>
      numeric.of('123456789012345678901234567890123456789012345'),
    );
  });
});

// ---- §51 舍入 ----

describe('舍入', () => {
  it('七种模式语义（HALF_UP 平台默认）', () => {
    const v = '2.345';
    expect(numeric.round(v, 2).serialize()).toBe('2.35');
    expect(numeric.round(v, 2, RoundingMode.HALF_DOWN).serialize()).toBe('2.34');
    expect(numeric.round(v, 2, RoundingMode.HALF_EVEN).serialize()).toBe('2.34');
    expect(numeric.round('2.341', 2, RoundingMode.UP).serialize()).toBe('2.35');
    expect(numeric.round('2.349', 2, RoundingMode.DOWN).serialize()).toBe('2.34');
    expect(numeric.round('2.341', 2, RoundingMode.CEIL).serialize()).toBe('2.35');
    expect(numeric.round('2.349', 2, RoundingMode.FLOOR).serialize()).toBe('2.34');
  });

  it('scale 越界抛 INVALID_ARGUMENT', () => {
    expectCode('INVALID_ARGUMENT', () => numeric.round('1.23', -1));
    expectCode('INVALID_ARGUMENT', () => numeric.round('1.23', 19));
  });

  it('尾零语义三分离：canonical 最短表示 / toFixed 定长展示（设计 §48/§49）', () => {
    expect(numeric.of('3.330000').serialize()).toBe('3.33');
    expect(numeric.of('3.3').toFixed(4)).toBe('3.3000');
    expect(numeric.of('3.345').round(3).toFixed(3)).toBe('3.345');
  });
});

// ---- §51 聚合 ----

describe('聚合', () => {
  it('sum/average/min/max/product', () => {
    const values = ['1.1', '2.2', '3.3'];
    expect(numeric.sum(values).serialize()).toBe('6.6');
    expect(numeric.average(values).serialize()).toBe('2.2');
    expect(numeric.min(values)?.serialize()).toBe('1.1');
    expect(numeric.max(values)?.serialize()).toBe('3.3');
    expect(numeric.product(['2', '3', '4']).serialize()).toBe('24');
  });

  it('空数组：min/max 为 null，sum/average/product 为空值', () => {
    expect(numeric.min([])).toBeNull();
    expect(numeric.max([])).toBeNull();
    expect(numeric.sum([]).serialize()).toBeNull();
    expect(numeric.average([]).serialize()).toBeNull();
  });

  it('NULL 元素按 SQL 语义跳过', () => {
    expect(numeric.sum(['1', null, '2']).serialize()).toBe('3');
    expect(numeric.average(['1', null, '3']).serialize()).toBe('2');
    expect(numeric.min([null, '5'])?.serialize()).toBe('5');
  });
});

// ---- §51 空值 ----

describe('空值策略（设计 §12）', () => {
  it('NULL（平台默认）：空值传播，比较判定为假', () => {
    expect(numeric.of(null).serialize()).toBeNull();
    expect(numeric.of('').serialize()).toBeNull();
    expect(numeric.add(null, '1').serialize()).toBeNull();
    expect(numeric.add('1', undefined).serialize()).toBeNull();
    expect(numeric.of(null).eq('0')).toBe(false);
    expect(numeric.of('1').gt(null)).toBe(false);
    expect(numeric.of(null).add('1').round(2).serialize()).toBeNull();
  });

  it('ZERO：公式兼容模式（电子表格语义）', () => {
    const sheet = createNumericRuntime({ emptyValuePolicy: EmptyValuePolicy.ZERO });
    expect(sheet.of(null).serialize()).toBe('0');
    expect(sheet.add(null, '1').serialize()).toBe('1');
    expect(sheet.sum(['1', null, '2']).serialize()).toBe('3');
  });

  it('ERROR：严格模式直接报错', () => {
    const strict = createNumericRuntime({ emptyValuePolicy: EmptyValuePolicy.ERROR });
    expectCode('INVALID_NUMBER', () => strict.of(null));
    expectCode('INVALID_NUMBER', () => strict.of(''));
  });
});

// ---- §51 异常 ----

describe('异常与稳定码（设计 §13）', () => {
  it('INVALID_NUMBER：垃圾文本', () => {
    expectCode('INVALID_NUMBER', () => numeric.of('abc'));
    expectCode('INVALID_NUMBER', () => numeric.of('12.3.4'));
  });

  it('INVALID_ARGUMENT：sqrt 负数 / compare 空值', () => {
    expectCode('INVALID_ARGUMENT', () => numeric.sqrt('-1'));
    expectCode('INVALID_ARGUMENT', () => numeric.compare(null, '1'));
  });

  it('比较三态与链式谓词', () => {
    expect(numeric.compare('2.5', '2.50')).toBe(0);
    expect(numeric.compare('-1', '1')).toBe(-1);
    expect(numeric.eq('0.10', '0.1')).toBe(true);
    expect(numeric.gte('2', '2')).toBe(true);
    expect(numeric.lt('2', '2')).toBe(false);
    expect(numeric.lte('2', '2')).toBe(true);
  });
});

// ---- 运行时上下文 ----

describe('NumericContext', () => {
  it('应用级上下文覆盖（字段级 scale 更小的场景）', () => {
    const money = createNumericRuntime({ maxScale: 2 });
    expect(money.divide('100', '3').round().serialize()).toBe('33.33');
    // maxScale=2 下 scale=3 越界
    expectCode('INVALID_ARGUMENT', () => money.of('1').round(3));
  });

  it('默认导出 numeric 即平台默认配置', () => {
    expect(numeric.ctx.precision).toBe(40);
    expect(numeric.ctx.maxScale).toBe(18);
    expect(numeric.ctx.roundingMode).toBe(RoundingMode.HALF_UP);
    expect(numeric.ctx.emptyValuePolicy).toBe(EmptyValuePolicy.NULL);
  });
});
