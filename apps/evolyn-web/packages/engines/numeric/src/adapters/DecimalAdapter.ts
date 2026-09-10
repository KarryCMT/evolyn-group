import Decimal from 'decimal.js'

import { NumericError } from '../errors/NumericError'
import { RoundingMode } from '../types/RoundingMode'
import { EmptyValuePolicy } from '../policies/EmptyValuePolicy'
import type { NumericContext } from '../core/NumericContext'
import type { NumericInput } from '../types/NumericInput'
import { Numeric } from '../core/Numeric'

/**
 * DecimalAdapter decimal.js 唯一触达点（设计 §5/§11/§14）：
 * 舍入常量映射、输入解析（空值策略/有限性/精度护栏）、规范化字符串。
 * 业务包禁止直接 import decimal.js——所有语义在此收口。
 */
export class DecimalAdapter {
  readonly ctx: NumericContext

  constructor(ctx: NumericContext) {
    this.ctx = ctx
    // 运算精度交由 decimal.js 全局配置（有效数字上限），超限即抛错
    Decimal.set({ precision: ctx.precision, defaults: true })
  }

  /** 平台舍入枚举 → decimal.js 舍入常量（禁止业务层直接引用后者） */
  roundingConstant(mode: RoundingMode): Decimal.Rounding {
    switch (mode) {
      case RoundingMode.UP:
        return Decimal.ROUND_UP
      case RoundingMode.DOWN:
        return Decimal.ROUND_DOWN
      case RoundingMode.CEIL:
        return Decimal.ROUND_CEIL
      case RoundingMode.FLOOR:
        return Decimal.ROUND_FLOOR
      case RoundingMode.HALF_UP:
        return Decimal.ROUND_HALF_UP
      case RoundingMode.HALF_DOWN:
        return Decimal.ROUND_HALF_DOWN
      case RoundingMode.HALF_EVEN:
        return Decimal.ROUND_HALF_EVEN
    }
  }

  /**
   * coerce 输入规约：Numeric 直取内部值；空值按策略处理（NULL 返回 null）；
   * number 校验有限性；string 走严格解析（禁 NaN/Infinity/垃圾文本）。
   * 解析结果超精度即抛 PRECISION_EXCEEDED。
   */
  coerce(input: NumericInput): Decimal | null {
    if (input instanceof Numeric) {
      return input.unwrap()
    }
    if (input === null || input === undefined || input === '') {
      switch (this.ctx.emptyValuePolicy) {
        case EmptyValuePolicy.ZERO:
          return new Decimal(0)
        case EmptyValuePolicy.NULL:
          return null
        case EmptyValuePolicy.ERROR:
          throw new NumericError('INVALID_NUMBER', 'empty value is not allowed in strict mode')
      }
    }
    if (typeof input === 'number') {
      if (!Number.isFinite(input)) {
        throw new NumericError('NON_FINITE_NUMBER', `number input is not finite: ${input}`)
      }
      return this.guard(new Decimal(input))
    }
    // string：显式拦截 NaN/Infinity（decimal.js 原生可解析，须先拒之，设计 §14）
    const text = input.trim()
    if (/^(nan|infinity|-infinity|\+?inf)$/i.test(text)) {
      throw new NumericError('NON_FINITE_NUMBER', `non-finite number literal: ${input}`)
    }
    if (!/^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$/.test(text)) {
      throw new NumericError('INVALID_NUMBER', `invalid decimal string: ${input}`)
    }
    return this.guard(new Decimal(text))
  }

  /** require 非空值断言：空值在 NULL 传播语义外的位置（如除数）不可为 null */
  require(input: NumericInput, what: string): Decimal {
    const value = this.coerce(input)
    if (value === null) {
      throw new NumericError('INVALID_ARGUMENT', `${what} must not be null`)
    }
    return value
  }

  /** guard 精度护栏：有效数字超出上下文上限即抛（设计 §10） */
  guard(value: Decimal): Decimal {
    if (value.isZero()) {
      return value
    }
    if (value.precision() > this.ctx.precision) {
      throw new NumericError('PRECISION_EXCEEDED', `significant digits exceed precision ${this.ctx.precision}`)
    }
    return value
  }

  /** guardScale 序列化前的 scale 护栏：超 maxScale 须显式 round 落位 */
  guardScale(value: Decimal): Decimal {
    if (value.dp() > this.ctx.maxScale) {
      throw new NumericError(
        'SCALE_EXCEEDED',
        `decimal places ${value.dp()} exceed maxScale ${this.ctx.maxScale}; round explicitly before serialize`,
      )
    }
    return value
  }

  /**
   * canonical 规范化十进制字符串（设计 §49）：
   * 展开科学计数法、输出最短精确表示（无意义尾零随 decimal.js 构造去除——
   * 定长尾零属 toFixed 展示语义与字段 Serialization Policy 存储语义），
   * 禁止 +001.23e2/NaN/Infinity 形态。
   */
  canonical(value: Decimal): string {
    const text = value.toString()
    if (!/[eE]/.test(text)) {
      return text === '-0' ? '0' : text
    }
    // 指数形态展开：整数幂用定点（1e+40 → 41 位整串）；小数幂按实际小数位
    return value.toFixed(Math.max(0, value.dp()))
  }
}
