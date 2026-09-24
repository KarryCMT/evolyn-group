import {
  type FormFieldRegistry,
  createNativeFieldRegistry,
} from '../../runtime/widgets/registry';
import MobileCheckboxGroupField from './MobileCheckboxGroupField.vue';
import MobileMultiSelectField from './MobileMultiSelectField.vue';
import MobileNumberField from './MobileNumberField.vue';
import MobileRadioGroupField from './MobileRadioGroupField.vue';
import MobileSelectField from './MobileSelectField.vue';
import MobileTextField from './MobileTextField.vue';

/**
 * Vant 移动字段注册表。先继承 Core 的完整原生回退，再覆盖已具备成熟移动交互的字段，
 * 保证新控件可以按白名单渐进接入，同时不让 UI 框架反向进入 Runtime Core。
 */
export function createMobileFieldRegistry(): FormFieldRegistry {
  return createNativeFieldRegistry()
    .register('text', { component: MobileTextField })
    .register('textarea', { component: MobileTextField })
    .register('number', { component: MobileNumberField })
    .register('radiogroup', { component: MobileRadioGroupField })
    .register('checkboxgroup', { component: MobileCheckboxGroupField })
    .register('combo', { component: MobileSelectField })
    .register('combocheck', { component: MobileMultiSelectField });
}
