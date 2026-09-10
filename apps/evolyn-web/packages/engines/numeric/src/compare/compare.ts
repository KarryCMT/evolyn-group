import { Numeric } from '../core/Numeric'
import type { DecimalAdapter } from '../adapters/DecimalAdapter'
import type { NumericInput } from '../types/NumericInput'

/**
 * compare 三态比较（设计 §8）：返回 -1/0/1。
 * 任一侧空值抛 INVALID_ARGUMENT——布尔判定（eq/gt 系列）才是空值的
 * SQL 假值语义入口。
 */
export function compare(a: NumericInput, b: NumericInput, adapter: DecimalAdapter): number {
  return new Numeric(adapter.coerce(a), adapter).compare(b)
}
