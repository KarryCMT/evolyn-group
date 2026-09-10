import type { NumericErrorCode } from './NumericErrorCode'

/**
 * NumericError 数值运行时统一错误（设计 §13）。
 * code 为稳定标识：Formula 层映射 #DIV/0! / #VALUE!，页面层映射中文文案。
 */
export class NumericError extends Error {
  readonly code: NumericErrorCode

  constructor(code: NumericErrorCode, message: string) {
    super(message)
    this.name = 'NumericError'
    this.code = code
  }
}
