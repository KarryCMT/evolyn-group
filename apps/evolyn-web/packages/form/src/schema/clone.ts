/**
 * 目标协议深拷贝器（P1）。文档只含 JSON 安全值（经校验器保证），JSON 往返即语义等价
 * 深拷贝，同时天然切断与设计器 reactive 状态的引用关系；循环引用在 stringify 抛错，
 * 由调用方（校验前置）保证不会出现。泛型不加 FormJsonValue 约束：协议接口类型
 * （FormSchemaDocument 等）不具备索引签名，约束会使克隆结果类型塌缩。
 */

/** 深拷贝 JSON 安全值；仅允许传入已通过协议校验或天然 JSON 安全的数据。 */
export function cloneFormSchema<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}
