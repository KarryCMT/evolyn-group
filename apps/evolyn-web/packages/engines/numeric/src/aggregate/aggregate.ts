import { Numeric } from '../core/Numeric'
import type { DecimalAdapter } from '../adapters/DecimalAdapter'
import type { NumericInput } from '../types/NumericInput'

/**
 * 聚合函数（设计 §8/§51）：
 * NULL 策略下空值元素按 SQL 语义跳过（SUM/AVG/MIN/MAX 均忽略 NULL），
 * 全空/空数组 → 空值结果；ZERO 策略下空值已在 coerce 处归零参与计算。
 */

/** 有效（非空）元素展开 */
function effective(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric[] {
  return values.map((v) => new Numeric(adapter.coerce(v), adapter))
}

function nonNull(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric[] {
  return effective(values, adapter).filter((n) => !n.isNull())
}

export function sum(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric {
  const items = nonNull(values, adapter)
  if (items.length === 0) {
    return new Numeric(null, adapter)
  }
  return items.reduce((acc, cur) => acc.add(cur))
}

export function average(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric {
  const items = nonNull(values, adapter)
  if (items.length === 0) {
    return new Numeric(null, adapter)
  }
  return sum(items, adapter).divide(String(items.length))
}

export function min(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric | null {
  const items = nonNull(values, adapter)
  if (items.length === 0) {
    return null
  }
  return items.reduce((acc, cur) => (cur.lt(acc) ? cur : acc))
}

export function max(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric | null {
  const items = nonNull(values, adapter)
  if (items.length === 0) {
    return null
  }
  return items.reduce((acc, cur) => (cur.gt(acc) ? cur : acc))
}

export function product(values: readonly NumericInput[], adapter: DecimalAdapter): Numeric {
  const items = nonNull(values, adapter)
  if (items.length === 0) {
    return new Numeric(null, adapter)
  }
  return items.reduce((acc, cur) => acc.multiply(cur))
}
