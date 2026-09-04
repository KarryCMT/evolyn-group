/**
 * 对已校验的 JSON 安全数据做语义深拷贝，切断设计器/运行时状态引用。
 * 循环引用、函数等非 JSON 值是调用方的协议校验责任，本函数不试图宽容处理。
 */
export function cloneJsonValue<Value>(value: Value): Value {
  return JSON.parse(JSON.stringify(value)) as Value;
}
