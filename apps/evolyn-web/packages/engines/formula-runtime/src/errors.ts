import { NumericError } from '@evolyn.do/numeric'

/**
 * FormulaError 公式求值错误（设计 §13）：
 * cause 为 NumericError 稳定码，alias 为电子表格习惯展示位（#DIV/0! 等），
 * 页面层可另映射中文文案。
 */
export type FormulaErrorCode =
  | 'FIELD_NOT_FOUND'
  | 'FIELD_NOT_RESOLVABLE'
  | 'FUNCTION_NOT_IMPLEMENTED'
  | 'FUNCTION_ARG_MISMATCH'
  | 'UNSUPPORTED_OPERATOR'
  | 'INVALID_LITERAL'
  | NumericErrorCodeAlias

type NumericErrorCodeAlias =
  | 'INVALID_NUMBER'
  | 'DIVISION_BY_ZERO'
  | 'NON_FINITE_NUMBER'
  | 'PRECISION_EXCEEDED'
  | 'SCALE_EXCEEDED'
  | 'INVALID_ARGUMENT'

const SPREADSHEET_ALIAS: Partial<Record<FormulaErrorCode, string>> = {
  DIVISION_BY_ZERO: '#DIV/0!',
  INVALID_NUMBER: '#VALUE!',
  NON_FINITE_NUMBER: '#VALUE!',
  SCALE_EXCEEDED: '#NUM!',
  PRECISION_EXCEEDED: '#NUM!',
  INVALID_ARGUMENT: '#NUM!',
}

export class FormulaError extends Error {
  readonly code: FormulaErrorCode
  /** 电子表格风格展示位（#DIV/0!/#VALUE!/#NUM!）；无对应位用 #ERROR! */
  readonly alias: string

  constructor(code: FormulaErrorCode, message: string) {
    super(message)
    this.name = 'FormulaError'
    this.code = code
    this.alias = SPREADSHEET_ALIAS[code] ?? '#ERROR!'
  }

  /** fromNumeric 数值层错误转公式层（保持稳定码透传） */
  static fromNumeric(err: NumericError): FormulaError {
    return new FormulaError(err.code as FormulaErrorCode, err.message)
  }
}
