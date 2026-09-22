import { Numeric, type NumericInput, type NumericRuntime, numeric } from '@evolyn.do/numeric';

import { FormulaError } from './errors';
import { type RuntimeValue, numericElements, tryAsNumeric } from './values';

/**
 * 函数注册表（设计 §21）：数值函数全部以 @evolyn.do/numeric 为底层实现，
 * 本层只负责参数展开/条件与数组语义（SUMIF/SUMPRODUCT）。
 * 逻辑函数（IF/AND/OR/NOT）服务于公式完整性；文本/日期函数不在数值
 * 运行时范围（Phase 4 按字段能力接入时扩展）。
 */
export type FormulaFunctionImpl = (
  args: readonly RuntimeValue[],
  ctx: FunctionContext,
) => RuntimeValue;

export interface FunctionContext {
  runtime: NumericRuntime;
}

/** 二元数值函数的参数解包：两侧数值化后委托 NumericRuntime */
function numericBinary(fn: (a: Numeric, b: NumericInput) => Numeric): FormulaFunctionImpl {
  return (args, ctx) => {
    assertArity(args, 2, 2);
    const a = ctx.runtime.of(requireNumeric(args[0], ctx));
    return fn(a, requireNumeric(args[1], ctx));
  };
}

function requireNumeric(value: RuntimeValue | undefined, ctx: FunctionContext): NumericInput {
  if (value === undefined) {
    return null;
  }
  const n = tryAsNumeric(value, ctx.runtime);
  if (n === null) {
    return null; // 空值交由 NumericRuntime 按策略传播（平台默认 NULL）
  }
  return n;
}

function assertArity(args: readonly unknown[], min: number, max: number, name = 'function'): void {
  if (args.length < min || args.length > max) {
    throw new FormulaError('FUNCTION_ARG_MISMATCH', `${name} 参数数量不匹配（${args.length}）`);
  }
}

/** 数组参数解包：RuntimeValue[] 直接用；标量视为单元素（AVERAGE($a) 兼容） */
function flatten(args: readonly RuntimeValue[]): (RuntimeValue | undefined)[] {
  const out: (RuntimeValue | undefined)[] = [];
  for (const arg of args) {
    if (Array.isArray(arg)) {
      out.push(...arg);
    } else {
      out.push(arg);
    }
  }
  return out;
}

/** SUMIF 条件解析：">10"/"<=5"/"10"/"文本" → 谓词（数值比较走 NumericRuntime） */
function sumifPredicate(
  criteria: RuntimeValue,
  ctx: FunctionContext,
): (v: RuntimeValue | undefined) => boolean {
  if (typeof criteria === 'string') {
    const match = criteria.match(/^(>=|<=|!=|>|<|==)\s*(.+)$/);
    if (match?.[1] && match[2] !== undefined) {
      const op = match[1];
      const operand = tryAsNumeric(match[2], ctx.runtime);
      if (operand !== null) {
        return (v: RuntimeValue | undefined) => {
          const n = v === undefined ? null : tryAsNumeric(v, ctx.runtime);
          if (n === null) {
            return false;
          }
          switch (op) {
            case '>':
              return n.gt(operand);
            case '<':
              return n.lt(operand);
            case '>=':
              return n.gte(operand);
            case '<=':
              return n.lte(operand);
            case '!=':
              return !n.eq(operand);
            default:
              return n.eq(operand);
          }
        };
      }
    }
    return (v) => typeof v === 'string' && v === criteria;
  }
  const numCriteria = tryAsNumeric(criteria, ctx.runtime);
  if (numCriteria !== null) {
    return (v: RuntimeValue | undefined) => {
      const n = v === undefined ? null : tryAsNumeric(v, ctx.runtime);
      return n !== null && n.eq(numCriteria);
    };
  }
  return () => false;
}

export const FORMULA_RUNTIME_FUNCTIONS: Readonly<Record<string, FormulaFunctionImpl>> = {
  // ---- 数学函数：NumericRuntime 底层（设计 §21 映射表） ----
  ABS: (args, ctx) => {
    assertArity(args, 1, 1, 'ABS');
    return ctx.runtime.abs(requireNumeric(args[0], ctx));
  },
  AVERAGE: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'AVERAGE');
    return ctx.runtime.average(numericElements(flatten(args), ctx.runtime));
  },
  CEILING: (args, ctx) => {
    assertArity(args, 1, 2, 'CEILING');
    const a = ctx.runtime.of(requireNumeric(args[0], ctx));
    const b = requireNumeric(args[1], ctx);
    return b === null ? a.ceil() : a.divide(b).ceil().multiply(b);
  },
  FLOOR: (args, ctx) => {
    assertArity(args, 1, 1, 'FLOOR');
    return ctx.runtime.floor(requireNumeric(args[0], ctx));
  },
  INT: (args, ctx) => {
    assertArity(args, 1, 1, 'INT');
    return ctx.runtime.floor(requireNumeric(args[0], ctx));
  },
  MAX: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'MAX');
    return ctx.runtime.max(numericElements(flatten(args), ctx.runtime));
  },
  MIN: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'MIN');
    return ctx.runtime.min(numericElements(flatten(args), ctx.runtime));
  },
  MOD: numericBinary((a: Numeric, b: NumericInput) => a.mod(b)),
  POWER: numericBinary((a: Numeric, b: NumericInput) => a.pow(b)),
  PRODUCT: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'PRODUCT');
    return ctx.runtime.product(numericElements(flatten(args), ctx.runtime));
  },
  ROUND: (args, ctx) => {
    assertArity(args, 1, 2, 'ROUND');
    const scale = args.length === 2 ? Number(tryAsNumeric(args[1], ctx.runtime)?.toFixed(0)) : 0;
    return ctx.runtime.round(requireNumeric(args[0], ctx), Number.isFinite(scale) ? scale : 0);
  },
  SQRT: (args, ctx) => {
    assertArity(args, 1, 1, 'SQRT');
    return ctx.runtime.sqrt(requireNumeric(args[0], ctx));
  },
  SUM: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'SUM');
    return ctx.runtime.sum(numericElements(flatten(args), ctx.runtime));
  },
  SUMIF: (args, ctx) => {
    assertArity(args, 2, 3, 'SUMIF');
    const range = Array.isArray(args[0]) ? args[0] : [args[0]];
    const predicate = sumifPredicate(args[1] ?? null, ctx);
    const sumRange = args.length === 3 && Array.isArray(args[2]) ? args[2] : range;
    const picked: NumericInput[] = [];
    range.forEach((v, i) => {
      if (predicate(v)) {
        const target = sumRange[i] ?? v;
        const n = tryAsNumeric(target, ctx.runtime);
        picked.push(n === null ? null : n);
      }
    });
    return ctx.runtime.sum(picked);
  },
  SUMPRODUCT: (args, ctx) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'SUMPRODUCT');
    // SUMPRODUCT([$q#, $p#]) 的实参是单个数组节点，其元素为各字段的数组：
    // 一层展开为操作数数组；多参形态（SUMPRODUCT(a, b)）同样支持
    const flat = args.length === 1 && Array.isArray(args[0]) ? args[0] : args;
    const arrays = flat.map((a) => (Array.isArray(a) ? a : [a]));
    const len = Math.max(...arrays.map((a) => a.length));
    const products: NumericInput[] = [];
    for (let i = 0; i < len; i++) {
      let acc: Numeric | null = null;
      for (const arr of arrays) {
        const n = tryAsNumeric(arr[i], ctx.runtime);
        if (n === null) {
          acc = null;
          break;
        }
        acc = acc === null ? n : acc.multiply(n);
      }
      products.push(acc === null ? null : acc);
    }
    return ctx.runtime.sum(products);
  },

  // ---- 逻辑函数（公式完整性） ----
  IF: (args) => {
    assertArity(args, 3, 3, 'IF');
    return truthy(args[0]) ? (args[1] ?? null) : (args[2] ?? null);
  },
  AND: (args) => args.every((v) => truthy(v)),
  OR: (args) => args.some((v) => truthy(v)),
  NOT: (args) => {
    assertArity(args, 1, 1, 'NOT');
    return !truthy(args[0]);
  },
  TRUE: () => true,
  FALSE: () => false,

  // ---- 文本函数（字段公式 V1） ----
  CONCATENATE: (args) => {
    assertArity(args, 1, Number.MAX_SAFE_INTEGER, 'CONCATENATE');
    return args.map(runtimeText).join('');
  },
  LEN: (args) => {
    assertArity(args, 1, 1, 'LEN');
    return numeric.of(String(runtimeText(args[0])).length);
  },
  LOWER: (args) => {
    assertArity(args, 1, 1, 'LOWER');
    return runtimeText(args[0]).toLocaleLowerCase();
  },
  UPPER: (args) => {
    assertArity(args, 1, 1, 'UPPER');
    return runtimeText(args[0]).toLocaleUpperCase();
  },
  TRIM: (args) => {
    assertArity(args, 1, 1, 'TRIM');
    return runtimeText(args[0]).trim().replace(/\s+/g, ' ');
  },
  ISBLANK: (args) => {
    assertArity(args, 1, 1, 'ISBLANK');
    const value = args[0];
    return value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0);
  },
  ISEMPTY: (args) => {
    assertArity(args, 1, 1, 'ISEMPTY');
    const value = args[0];
    return value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0);
  },
};

function runtimeText(value: RuntimeValue | undefined): string {
  if (value === undefined || value === null) return '';
  if (value instanceof Numeric) return value.serialize() ?? '';
  if (Array.isArray(value)) return value.map(runtimeText).join('、');
  return String(value);
}

function truthy(value: RuntimeValue | undefined): boolean {
  if (value === undefined) {
    return false;
  }
  if (value === null) {
    return false;
  }
  if (typeof value === 'boolean') {
    return value;
  }
  if (value instanceof Numeric) {
    return !value.isNull() && !value.eq('0');
  }
  if (typeof value === 'string') {
    return value !== '';
  }
  return value.length > 0;
}
