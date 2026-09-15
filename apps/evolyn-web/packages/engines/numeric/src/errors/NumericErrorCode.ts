/** NumericError 稳定错误码（设计 §13）：Formula Runtime 与 UI 各自映射展示 */
export type NumericErrorCode =
  | 'INVALID_NUMBER'
  | 'DIVISION_BY_ZERO'
  | 'NON_FINITE_NUMBER'
  | 'PRECISION_EXCEEDED'
  | 'SCALE_EXCEEDED'
  | 'INVALID_ARGUMENT';
