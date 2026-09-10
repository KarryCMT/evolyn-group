/**
 * @evolyn.do/numeric 高精度数值运行时（设计：docs/低代码平台/表单设计器/
 * 表单数值计算运行时功能设计-V1.0.md，Phase 1）。
 *
 * 平台数值铁律：
 * - decimal 全链路 string 序列化（canonical decimal），禁 number 直传；
 * - 业务包禁止 import decimal.js——Decimal 是本包私有实现；
 * - 空值经 EmptyValuePolicy 收口，运算精度（40 位有效数字）全程 Decimal，
 *   业务 scale 仅在落库/展示前显式 round。
 */
export { Numeric } from './core/Numeric'
export { createNumericRuntime, type NumericRuntime } from './core/NumericRuntime'
export { DEFAULT_NUMERIC_CONTEXT, normalizeContext, type NumericContext } from './core/NumericContext'
export { DecimalAdapter } from './adapters/DecimalAdapter'

export { RoundingMode } from './types/RoundingMode'
export { EmptyValuePolicy } from './policies/EmptyValuePolicy'
export { NumericError } from './errors/NumericError'
export type { NumericErrorCode } from './errors/NumericErrorCode'
export type { NumericInput } from './types/NumericInput'

export { serialize, format } from './format/serialize'
export { compare } from './compare/compare'

import { createNumericRuntime } from './core/NumericRuntime'

/** numeric 平台默认运行时（V1 默认配置：precision 40 / maxScale 18 / HALF_UP / NULL） */
export const numeric = createNumericRuntime()
