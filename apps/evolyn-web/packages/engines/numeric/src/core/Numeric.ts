import type Decimal from 'decimal.js'

import { DecimalAdapter } from '../adapters/DecimalAdapter'
import { NumericError } from '../errors/NumericError'
import { RoundingMode } from '../types/RoundingMode'
import type { NumericInput } from '../types/NumericInput'

/**
 * Numeric 数值载体（设计 §7.2）：内部持有 decimal.js 实例（或 null），
 * 对外仅暴露链式运算与序列化。null 表示 SQL NULL 语义的空值并沿运算传播；
 * 真实 Decimal 不出网，上层不得触碰 decimal.js 类型。
 */
export class Numeric {
  readonly kind = 'numeric' as const

  #value: Decimal | null
  #adapter: DecimalAdapter

  constructor(value: Decimal | null, adapter: DecimalAdapter) {
    this.#value = value
    this.#adapter = adapter
  }

  /** unwrap 仅供适配层/同包内部使用，不对外导出类型 */
  unwrap(): Decimal | null {
    return this.#value
  }

  /** isNull 当前是否为空值（NULL 策略传播结果） */
  isNull(): boolean {
    return this.#value === null
  }

  private map(fn: (a: Decimal, b: Decimal) => Decimal, other: NumericInput): Numeric {
    const left = this.#value
    const right = this.#adapter.coerce(other)
    // NULL 传播：任一侧空值，结果为空值（SQL 语义）
    if (left === null || right === null) {
      return new Numeric(null, this.#adapter)
    }
    return new Numeric(this.#adapter.guard(fn(left, right)), this.#adapter)
  }

  private unary(fn: (a: Decimal) => Decimal): Numeric {
    if (this.#value === null) {
      return this
    }
    return new Numeric(this.#adapter.guard(fn(this.#value)), this.#adapter)
  }

  add(value: NumericInput): Numeric {
    return this.map((a, b) => a.plus(b), value)
  }

  subtract(value: NumericInput): Numeric {
    return this.map((a, b) => a.minus(b), value)
  }

  multiply(value: NumericInput): Numeric {
    return this.map((a, b) => a.times(b), value)
  }

  divide(value: NumericInput): Numeric {
    return this.map((a, b) => {
      if (b.isZero()) {
        throw new NumericError('DIVISION_BY_ZERO', 'division by zero')
      }
      return a.div(b)
    }, value)
  }

  mod(value: NumericInput): Numeric {
    return this.map((a, b) => {
      if (b.isZero()) {
        throw new NumericError('DIVISION_BY_ZERO', 'modulo by zero')
      }
      return a.mod(b)
    }, value)
  }

  pow(value: NumericInput): Numeric {
    return this.map((a, b) => a.pow(b), value)
  }

  abs(): Numeric {
    return this.unary((a) => a.abs())
  }

  negate(): Numeric {
    return this.unary((a) => a.negated())
  }

  sqrt(): Numeric {
    if (this.#value === null) {
      return this
    }
    if (this.#value.isNeg()) {
      throw new NumericError('INVALID_ARGUMENT', 'sqrt of negative number')
    }
    return this.unary((a) => a.sqrt())
  }

  /** round 按小数位舍入（默认 maxScale 与上下文舍入模式；scale 须在 [0, maxScale]） */
  round(scale?: number, mode?: RoundingMode): Numeric {
    const effectiveScale = scale ?? this.#adapter.ctx.maxScale
    if (effectiveScale < 0 || effectiveScale > this.#adapter.ctx.maxScale) {
      throw new NumericError('INVALID_ARGUMENT', `scale must be within [0, ${this.#adapter.ctx.maxScale}]`)
    }
    if (this.#value === null) {
      return this
    }
    const rounding = this.#adapter.roundingConstant(mode ?? this.#adapter.ctx.roundingMode)
    return new Numeric(this.#value.toDecimalPlaces(effectiveScale, rounding), this.#adapter)
  }

  ceil(scale?: number): Numeric {
    return this.roundToDirection(scale, RoundingMode.CEIL)
  }

  floor(scale?: number): Numeric {
    return this.roundToDirection(scale, RoundingMode.FLOOR)
  }

  private roundToDirection(scale: number | undefined, mode: RoundingMode): Numeric {
    const effectiveScale = scale ?? 0
    if (effectiveScale < 0 || effectiveScale > this.#adapter.ctx.maxScale) {
      throw new NumericError('INVALID_ARGUMENT', `scale must be within [0, ${this.#adapter.ctx.maxScale}]`)
    }
    if (this.#value === null) {
      return this
    }
    const rounding = this.#adapter.roundingConstant(mode)
    return new Numeric(this.#value.toDecimalPlaces(effectiveScale, rounding), this.#adapter)
  }

  /** compare 三态比较：-1/0/1；空值参与比较抛 INVALID_ARGUMENT（用 eq/gt 系列获取 SQL 假值语义） */
  compare(value: NumericInput): number {
    const left = this.#adapter.require(this, 'left operand')
    const right = this.#adapter.require(value, 'right operand')
    return left.comparedTo(right) ?? 0
  }

  eq(value: NumericInput): boolean {
    return this.nullSafeCompare(value) === 0
  }

  gt(value: NumericInput): boolean {
    return this.nullSafeCompare(value) > 0
  }

  gte(value: NumericInput): boolean {
    return this.nullSafeCompare(value) >= 0
  }

  lt(value: NumericInput): boolean {
    return this.nullSafeCompare(value) < 0
  }

  lte(value: NumericInput): boolean {
    return this.nullSafeCompare(value) <= 0
  }

  /** nullSafeCompare 空值按 SQL 语义返回 NaN → 各判定为 false */
  private nullSafeCompare(value: NumericInput): number {
    if (this.#value === null || this.#adapter.coerce(value) === null) {
      return Number.NaN
    }
    return this.compare(value)
  }

  /**
   * serialize 规范化十进制字符串（设计 §48/§49）：不主动截断 scale，
   * 超过 maxScale 须先显式 round（SCALE_EXCEEDED）；空值返回 null。
   */
  serialize(): string | null {
    if (this.#value === null) {
      return null
    }
    return this.#adapter.canonical(this.#adapter.guardScale(this.#value))
  }

  /** toFixed 定长输出（展示语义）：保留尾零，空值返回 null */
  toFixed(scale: number): string | null {
    if (scale < 0 || scale > this.#adapter.ctx.maxScale) {
      throw new NumericError('INVALID_ARGUMENT', `scale must be within [0, ${this.#adapter.ctx.maxScale}]`)
    }
    if (this.#value === null) {
      return null
    }
    return this.#value.toFixed(scale)
  }
}
