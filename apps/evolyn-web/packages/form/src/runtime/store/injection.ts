import { inject, provide, type InjectionKey, type ShallowRef } from 'vue';
import type { FormRuntime } from './createFormRuntime';
import type { FormFieldRegistry } from '../widgets/registry';

/**
 * FormRenderer 向子组件注入的会话上下文（文档 §5.2）。
 * runtime 用 shallowRef 承载：Schema 引用变更时整体替换会话，
 * 子组件经 computed 消费，值输入只触发自身 key 的更新。
 */
export interface FormRendererContext {
  runtime: ShallowRef<FormRuntime | null>;
  registry: FormFieldRegistry;
  /** 字段聚焦注册表：错误定位时 FormRenderer 聚焦第一个出错字段（文档 §7.4）。 */
  registerFieldFocus(key: string, focus: () => void): void;
  unregisterFieldFocus(key: string): void;
  focusField(key: string): boolean;
  /** 诊断回传：未知/未启用字段类型不可静默丢弃，上报宿主处理（文档 §7.2）。 */
  reportUnsupportedField(info: { fieldKey: string; type: string }): void;
}

export const FormRendererContextKey: InjectionKey<FormRendererContext> = Symbol(
  'evolyn-form-renderer-context',
);

/** 仅供运行时内部组件（Section/Host/SubmitBar）使用；宿主页面不应注入伪造上下文。 */
export function useFormRendererContext(): FormRendererContext {
  const context = inject(FormRendererContextKey);
  if (!context) {
    throw new Error('表单运行时上下文缺失：该组件只能渲染在 FormRenderer 内部。');
  }
  return context;
}

/** FormRenderer 装配上下文；focusHandlers 由渲染器私有持有，这里仅完成注入。 */
export function provideFormRendererContext(context: FormRendererContext): void {
  provide(FormRendererContextKey, context);
}

/** 创建聚焦注册表的可复用实现，FormRenderer 与测试均可使用。 */
export function createFocusRegistry() {
  const handlers = new Map<string, () => void>();
  return {
    registerFieldFocus(key: string, focus: () => void): void {
      handlers.set(key, focus);
    },
    unregisterFieldFocus(key: string): void {
      handlers.delete(key);
    },
    focusField(key: string): boolean {
      const focus = handlers.get(key);
      if (!focus) return false;
      focus();
      return true;
    },
  };
}
