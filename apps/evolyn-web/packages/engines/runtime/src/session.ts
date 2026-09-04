import type { RuntimeSessionState } from './types.js';

/**
 * 创建可由 Vue、原生容器或测试宿主接管的运行会话初始状态。
 * 引擎不引入响应式库；调用方可包装为 reactive/shallowReactive 或采用不可变状态管理。
 */
export function createRuntimeSessionState<
  Value,
  FieldState,
  Issue,
  Lifecycle extends string,
  Operation extends string,
>(lifecycle: Lifecycle): RuntimeSessionState<Value, FieldState, Issue, Lifecycle, Operation> {
  return {
    values: {},
    fieldStates: {},
    lifecycle,
    activeOperation: null,
    dirtyKeys: new Set<string>(),
    issues: [],
  };
}
