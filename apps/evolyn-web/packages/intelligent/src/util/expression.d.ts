/** 旧表达式工具的类型边界，供迁移后的 Vue 3 组件安全消费。 */
export interface ComparisonOperator {
  name: string;
  value: string;
}

export const comparisonOperators: ComparisonOperator[];
