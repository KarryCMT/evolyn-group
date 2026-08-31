import { defineAsyncComponent, type Component } from 'vue';
import CheckboxGroupField from './base/CheckboxGroupField.vue';
import DateTimeField from './base/DateTimeField.vue';
import DividerWidget from './base/DividerWidget.vue';
import MultiSelectField from './base/MultiSelectField.vue';
import NumberField from './base/NumberField.vue';
import RadioGroupField from './base/RadioGroupField.vue';
import SelectField from './base/SelectField.vue';
import TextAreaField from './base/TextAreaField.vue';
import TextField from './base/TextField.vue';

/**
 * 字段注册表：`widget.type` 表达业务能力，组件实现经注册表解析（P2 起 keys 即
 * widget.type）。基础字段静态注册进首屏包；重型字段（成员/附件/富文本等，P3 起）
 * 必须以 loader 动态注册，避免静态 import 把它们合并进主入口。
 */
export type FieldWidgetLoader = () => Promise<{ default: Component }>;

export interface FieldWidgetDefinition {
  /** 基础字段：直接随运行时入口打包。 */
  component?: Component;
  /** 重型字段：动态模块，仅在首屏出现或临近可视区时导入。 */
  loader?: FieldWidgetLoader;
}

export class FormFieldRegistry {
  private readonly widgets = new Map<string, FieldWidgetDefinition>();
  /** 异步组件缓存：同一类型多次渲染复用同一包装，避免重复请求与组件重建。 */
  private readonly asyncCache = new Map<string, Component>();

  register(type: string, definition: FieldWidgetDefinition): this {
    if (!definition.component && !definition.loader) {
      throw new Error(`字段 ${type} 的注册项必须提供 component 或 loader。`);
    }
    this.widgets.set(type, definition);
    return this;
  }

  has(type: string): boolean {
    return this.widgets.has(type);
  }

  /** 解析字段组件；未注册类型返回 null，由 FormFieldHost 渲染受控的「暂不支持」状态。 */
  resolve(type: string): Component | null {
    const definition = this.widgets.get(type);
    if (!definition) return null;
    if (definition.component) return definition.component;
    const cached = this.asyncCache.get(type);
    if (cached) return cached;
    const asyncComponent = defineAsyncComponent({
      loader: definition.loader as FieldWidgetLoader,
      // 重型模块加载失败可整页重试，不在字段级静默吞错。
      onError(error, retry, fail, attempts) {
        if (attempts <= 2) retry();
        else fail(error as Error);
      },
    });
    this.asyncCache.set(type, asyncComponent);
    return asyncComponent;
  }
}

/**
 * 移动端注册表：仅含 P2 基础字段（原生 HTML 控件 + 最小 Vue 包装），
 * 注册键与目标协议 widget.type 一一对应。后续阶段按白名单追加，例如：
 *   registry.register('user', () => import('./heavy/UserField.vue'))
 */
export function createMobileFieldRegistry(): FormFieldRegistry {
  const registry = new FormFieldRegistry();
  registry.register('text', { component: TextField });
  registry.register('textarea', { component: TextAreaField });
  registry.register('number', { component: NumberField });
  registry.register('datetime', { component: DateTimeField });
  registry.register('radiogroup', { component: RadioGroupField });
  registry.register('checkboxgroup', { component: CheckboxGroupField });
  registry.register('combo', { component: SelectField });
  registry.register('combocheck', { component: MultiSelectField });
  registry.register('separator', { component: DividerWidget });
  return registry;
}
