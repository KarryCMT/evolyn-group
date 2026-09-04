/** 可映射到 JSON Path、字段键或 DSL 位置的结构化校验问题。 */
export interface ValidationDiagnostic {
  path: string;
  message: string;
  code?: string;
}

export interface ValidationResult<Value> {
  valid: boolean;
  value: Value | null;
  issues: readonly ValidationDiagnostic[];
}
