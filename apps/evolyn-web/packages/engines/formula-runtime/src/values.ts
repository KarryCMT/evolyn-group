import { Numeric, NumericError, numeric } from '@evolyn.do/numeric'
import type { NumericInput, NumericRuntime } from '@evolyn.do/numeric'

import { FormulaError } from './errors'

/**
 * 运行时值模型（设计 §19）：数值一律以 Numeric 载体在求值器内流动，
 * 禁止 Number()/parseFloat() 中转；null/布尔/文本/数组为非数值语义原样透传。
 */
export type RuntimeValue = Numeric | string | boolean | null | RuntimeValue[]

/** 数值化判定：Numeric 或可解析 decimal string/number（字段原始输入） */
export function asNumeric(value: RuntimeValue, rt: NumericRuntime = numeric): Numeric {
  if (value instanceof Numeric) {
    return value
  }
  if (typeof value === 'string' || typeof value === 'number') {
    try {
      return rt.of(value as NumericInput)
    } catch (err) {
      if (err instanceof NumericError) {
        throw FormulaError.fromNumeric(err)
      }
      throw err
    }
  }
  throw new FormulaError('INVALID_NUMBER', `操作数不是数值: ${describe(value)}`)
}

/** 数值化（宽松）：null/布尔/数组返回 null（不参与数值运算，交由上层空值语义） */
export function tryAsNumeric(
  value: RuntimeValue | undefined,
  rt: NumericRuntime = numeric,
): Numeric | null {
  if (value === undefined || value === null || typeof value === 'boolean' || Array.isArray(value)) {
    return null
  }
  return asNumeric(value, rt)
}

export function describe(value: RuntimeValue): string {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  if (value instanceof Numeric) return value.serialize() ?? 'null'
  if (typeof value === 'string') return `"${value}"`
  return String(value)
}

/**
 * 数组元素数值化：字段数组（如子表行取列）进入聚合函数前统一转换；
 * 空值元素保留 null（聚合按 SQL 语义跳过，见 numeric.aggregate）
 */
export function numericElements(
  values: readonly (RuntimeValue | undefined)[],
  rt: NumericRuntime = numeric,
): NumericInput[] {
  return values.map((v) => {
    const n = tryAsNumeric(v, rt)
    return n === null ? null : n
  })
}
