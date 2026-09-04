import type { ValidationDiagnostic, ValidationResult } from './types.js';

/**
 * 以不可变结果统一领域校验器的成功/失败出口。引擎只收敛诊断与结果形状，
 * 不解释字段 DSL、不操作 UI，也不承担服务端最终授权。
 */
export function createValidationResult<Value>(
  issues: readonly ValidationDiagnostic[],
  value: Value | null,
): ValidationResult<Value> {
  const diagnostics = Object.freeze([...issues]);
  return diagnostics.length === 0
    ? { valid: true, value, issues: diagnostics }
    : { valid: false, value: null, issues: diagnostics };
}
