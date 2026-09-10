/**
 * EmptyValuePolicy 表单空值（''/null/undefined）的统一处理策略（设计 §12）。
 * 策略在 NumericContext 中配置，绑定于运行时实例，业务层不得散落判空。
 */
export enum EmptyValuePolicy {
  /** null/'' 视为 0：公式兼容模式（沿用电子表格语义） */
  ZERO = 'ZERO',
  /** null/'' 保持 null 并沿运算传播（SQL NULL 语义）：平台默认 */
  NULL = 'NULL',
  /** 严格计算场景：空值直接抛 INVALID_NUMBER */
  ERROR = 'ERROR',
}
