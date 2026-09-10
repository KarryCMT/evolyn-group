import { DecimalAdapter } from '../adapters/DecimalAdapter'
import { Numeric } from './Numeric'
import { normalizeContext, type NumericContext } from './NumericContext'
import { RoundingMode } from '../types/RoundingMode'
import { sum, average, min, max, product } from '../aggregate/aggregate'

import type { NumericInput } from '../types/NumericInput'

/**
 * NumericRuntime 数值运行时公共 API（设计 §8）：函数式入口，
 * 绑定一份 NumericContext（平台默认/应用级/字段级皆可建实例）。
 */
export interface NumericRuntime {
  readonly ctx: NumericContext

  of(value: NumericInput): Numeric
  parse(value: NumericInput): Numeric

  add(a: NumericInput, b: NumericInput): Numeric
  subtract(a: NumericInput, b: NumericInput): Numeric
  multiply(a: NumericInput, b: NumericInput): Numeric
  divide(a: NumericInput, b: NumericInput): Numeric
  mod(a: NumericInput, b: NumericInput): Numeric
  pow(a: NumericInput, b: NumericInput): Numeric

  abs(value: NumericInput): Numeric
  negate(value: NumericInput): Numeric
  sqrt(value: NumericInput): Numeric

  round(value: NumericInput, scale?: number, mode?: RoundingMode): Numeric
  ceil(value: NumericInput, scale?: number): Numeric
  floor(value: NumericInput, scale?: number): Numeric

  compare(a: NumericInput, b: NumericInput): number
  eq(a: NumericInput, b: NumericInput): boolean
  gt(a: NumericInput, b: NumericInput): boolean
  gte(a: NumericInput, b: NumericInput): boolean
  lt(a: NumericInput, b: NumericInput): boolean
  lte(a: NumericInput, b: NumericInput): boolean

  sum(values: readonly NumericInput[]): Numeric
  average(values: readonly NumericInput[]): Numeric
  min(values: readonly NumericInput[]): Numeric | null
  max(values: readonly NumericInput[]): Numeric | null
  product(values: readonly NumericInput[]): Numeric

  serialize(value: NumericInput): string | null
}

class NumericRuntimeImpl implements NumericRuntime {
  readonly ctx: NumericContext
  readonly #adapter: DecimalAdapter

  constructor(ctx: NumericContext) {
    this.ctx = ctx
    this.#adapter = new DecimalAdapter(ctx)
  }

  of(value: NumericInput): Numeric {
    return new Numeric(this.#adapter.coerce(value), this.#adapter)
  }

  parse(value: NumericInput): Numeric {
    return this.of(value)
  }

  private binary(
    fn: (a: Numeric, b: NumericInput) => Numeric,
    a: NumericInput,
    b: NumericInput,
  ): Numeric {
    return fn(this.of(a), b)
  }

  add(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.add(y), a, b)
  }

  subtract(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.subtract(y), a, b)
  }

  multiply(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.multiply(y), a, b)
  }

  divide(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.divide(y), a, b)
  }

  mod(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.mod(y), a, b)
  }

  pow(a: NumericInput, b: NumericInput): Numeric {
    return this.binary((x, y) => x.pow(y), a, b)
  }

  abs(value: NumericInput): Numeric {
    return this.of(value).abs()
  }

  negate(value: NumericInput): Numeric {
    return this.of(value).negate()
  }

  sqrt(value: NumericInput): Numeric {
    return this.of(value).sqrt()
  }

  round(value: NumericInput, scale?: number, mode?: RoundingMode): Numeric {
    return this.of(value).round(scale, mode)
  }

  ceil(value: NumericInput, scale?: number): Numeric {
    return this.of(value).ceil(scale)
  }

  floor(value: NumericInput, scale?: number): Numeric {
    return this.of(value).floor(scale)
  }

  compare(a: NumericInput, b: NumericInput): number {
    return this.of(a).compare(b)
  }

  eq(a: NumericInput, b: NumericInput): boolean {
    return this.of(a).eq(b)
  }

  gt(a: NumericInput, b: NumericInput): boolean {
    return this.of(a).gt(b)
  }

  gte(a: NumericInput, b: NumericInput): boolean {
    return this.of(a).gte(b)
  }

  lt(a: NumericInput, b: NumericInput): boolean {
    return this.of(a).lt(b)
  }

  lte(a: NumericInput, b: NumericInput): boolean {
    return this.of(a).lte(b)
  }

  sum(values: readonly NumericInput[]): Numeric {
    return sum(values, this.#adapter)
  }

  average(values: readonly NumericInput[]): Numeric {
    return average(values, this.#adapter)
  }

  min(values: readonly NumericInput[]): Numeric | null {
    return min(values, this.#adapter)
  }

  max(values: readonly NumericInput[]): Numeric | null {
    return max(values, this.#adapter)
  }

  product(values: readonly NumericInput[]): Numeric {
    return product(values, this.#adapter)
  }

  serialize(value: NumericInput): string | null {
    return this.of(value).serialize()
  }
}

/** createNumericRuntime 按上下文创建运行时（平台默认/应用级/字段级） */
export function createNumericRuntime(context?: Partial<NumericContext>): NumericRuntime {
  return new NumericRuntimeImpl(normalizeContext(context))
}
