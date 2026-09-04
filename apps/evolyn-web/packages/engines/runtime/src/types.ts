/** 框架无关的运行会话状态；领域运行时自行决定如何使它响应式化。 */
export interface RuntimeSessionState<
  Value,
  FieldState,
  Issue,
  Lifecycle extends string,
  Operation extends string,
> {
  values: Record<string, Value>;
  fieldStates: Record<string, FieldState>;
  lifecycle: Lifecycle;
  activeOperation: Operation | null;
  dirtyKeys: Set<string>;
  issues: Issue[];
}
