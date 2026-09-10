import { RoundingMode } from '../types/RoundingMode'
import { EmptyValuePolicy } from '../policies/EmptyValuePolicy'

/**
 * NumericContext 数值运行时上下文（设计 §10.1/§50）。
 * 优先级：操作显式参数 > 字段 policy > 业务 Runtime policy > 平台默认。
 */
export interface NumericContext {
  /** 运算精度：有效数字位数上限（超限抛 PRECISION_EXCEEDED） */
  precision: number
  /** 最大小数位（序列化超限抛 SCALE_EXCEEDED，须显式 round 落位） */
  maxScale: number
  /** 默认舍入模式 */
  roundingMode: RoundingMode
  /** 空值策略 */
  emptyValuePolicy: EmptyValuePolicy
}

/** V1 平台默认配置（设计 §10.1） */
export const DEFAULT_NUMERIC_CONTEXT: NumericContext = {
  precision: 40,
  maxScale: 18,
  roundingMode: RoundingMode.HALF_UP,
  emptyValuePolicy: EmptyValuePolicy.NULL,
}

export function normalizeContext(partial?: Partial<NumericContext>): NumericContext {
  return { ...DEFAULT_NUMERIC_CONTEXT, ...partial }
}
