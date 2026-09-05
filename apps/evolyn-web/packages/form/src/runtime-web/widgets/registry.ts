import { FormFieldRegistry } from '../../runtime/widgets/registry';
import SubformField from '../../runtime/widgets/base/SubformField.vue';
import WebBasicField from './WebBasicField.vue';

/** Web 端字段注册表：所有基础字段由 Element Plus 控件承担交互和视觉呈现。 */
export function createWebFieldRegistry(): FormFieldRegistry {
  const registry = new FormFieldRegistry();
  for (const type of [
    'text',
    'textarea',
    'number',
    'datetime',
    'radiogroup',
    'checkboxgroup',
    'combo',
    'combocheck',
    'separator',
  ]) {
    registry.register(type, { component: WebBasicField });
  }
  registry.register('subform', { component: SubformField });
  return registry;
}
