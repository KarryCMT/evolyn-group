import { Numeric } from '../core/Numeric';
import type { DecimalAdapter } from '../adapters/DecimalAdapter';
import type { NumericInput } from '../types/NumericInput';

/**
 * 序列化与展示（设计 §48/§49）：
 * - serialize：canonical decimal string（不主动截断 scale，超限抛 SCALE_EXCEEDED）；
 * - format：定长展示（保留尾零）。存储值/展示值/存储 scale 三语义分离。
 */
export function serialize(value: NumericInput, adapter: DecimalAdapter): string | null {
  return new Numeric(adapter.coerce(value), adapter).serialize();
}

export function format(value: NumericInput, scale: number, adapter: DecimalAdapter): string | null {
  return new Numeric(adapter.coerce(value), adapter).toFixed(scale);
}
