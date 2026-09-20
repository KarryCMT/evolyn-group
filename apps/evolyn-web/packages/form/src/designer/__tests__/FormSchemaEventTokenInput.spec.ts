import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { defineComponent, nextTick, ref } from 'vue';
import FormSchemaEventTokenInput from '../FormSchemaEventTokenInput.vue';
import type { FormEventFieldOption } from '../frontend-events';

afterEach(() => {
  document.body.innerHTML = '';
});

describe('FormSchemaEventTokenInput', () => {
  it('keeps browser-managed contenteditable nodes out of Vue patching and inserts a field token', async () => {
    const fields = ref<FormEventFieldOption[]>([
      { key: '_widget_name', label: '姓名', type: 'text' },
    ]);
    const value = ref('https://api.lingyanyun.com/lookup?name=');
    const Host = defineComponent({
      components: { FormSchemaEventTokenInput },
      setup: () => ({ fields, value }),
      template: '<FormSchemaEventTokenInput v-model="value" :fields="fields" :rows="3" />',
    });
    const wrapper = mount(Host, { attachTo: document.body });
    const control = wrapper.get('.event-token-input__control');

    // 模拟浏览器对 contenteditable 的原生改写，再触发父组件和字段选择器导致的更新。
    control.element.textContent = `${value.value}employee`;
    await control.trigger('input');
    fields.value = [{ key: '_widget_name', label: '员工姓名', type: 'text' }];
    await nextTick();

    await wrapper.get('.event-token-input__insert').trigger('mousedown');
    await wrapper.get('.event-token-input__insert').trigger('click');
    // 字段选择器会 teleport 到 body，避免被嵌套弹窗的滚动容器裁切。
    const pickerOption = document.body.querySelector<HTMLElement>('.event-field-picker__option');
    expect(pickerOption).not.toBeNull();
    pickerOption?.click();
    await nextTick();

    expect(value.value).toBe('https://api.lingyanyun.com/lookup?name=employee${_widget_name}');
    expect(control.element.querySelector('[data-token="_widget_name"]')?.textContent).toBe(
      '员工姓名',
    );
    wrapper.unmount();
  });

  it('keeps the native caret DOM intact when the parent echoes an input update', async () => {
    const value = ref('https://api.lingyanyun.com/lookup?year=2024');
    const Host = defineComponent({
      components: { FormSchemaEventTokenInput },
      setup: () => ({ value }),
      template: '<FormSchemaEventTokenInput v-model="value" :fields="[]" :rows="3" />',
    });
    const wrapper = mount(Host, { attachTo: document.body });
    const control = wrapper.get('.event-token-input__control');
    const originalText = control.element.firstElementChild;

    // 模拟连续退格：浏览器直接改写 contenteditable，随后父组件将 v-model 值回传。
    originalText!.textContent = 'https://api.lingyanyun.com/lookup?year=202';
    await control.trigger('input');
    await nextTick();
    originalText!.textContent = 'https://api.lingyanyun.com/lookup?year=20';
    await control.trigger('input');
    await nextTick();

    expect(value.value).toBe('https://api.lingyanyun.com/lookup?year=20');
    expect(control.element.firstElementChild).toBe(originalText);
    wrapper.unmount();
  });
});
