import type { InjectionKey, ShallowRef } from 'vue';
import type { FormDetail } from '~/types';
import { inject } from 'vue';

/** 表单工作区详情上下文：外壳负责单次加载，子工作区共享同一响应。 */
export interface FormWorkspaceContext {
  detail: Readonly<ShallowRef<FormDetail | null>>;
  loading: Readonly<ShallowRef<boolean>>;
  loadFailed: Readonly<ShallowRef<boolean>>;
  renaming: Readonly<ShallowRef<boolean>>;
  setDetail: (detail: FormDetail) => void;
  patchDetail: (patch: Partial<FormDetail>) => void;
  rename: (name: string) => Promise<boolean>;
  reload: () => Promise<FormDetail | null>;
}

export const formWorkspaceContextKey: InjectionKey<FormWorkspaceContext> =
  Symbol('form-workspace-context');

/** 子路由必须位于 FormWorkspaceShell 下；缺少上下文属于路由装配错误。 */
export function useFormWorkspaceContext(): FormWorkspaceContext {
  const context = inject(formWorkspaceContextKey);
  if (!context) throw new Error('Form workspace context is not available');
  return context;
}
