import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import FormSchemaFrontendEventDebugDrawer from '../FormSchemaFrontendEventDebugDrawer.vue';
import { createFormEvent, type FormEventFieldOption } from '../frontend-events';

afterEach(() => {
  document.body.innerHTML = '';
});

describe('FormSchemaFrontendEventDebugDrawer', () => {
  it('only asks for referenced fields and previews the resolved request', async () => {
    const event = createFormEvent();
    event.name = '查询员工档案';
    event.request.url = 'https://api.lingyanyun.com/member?phone=${_widget_phone}';
    const fields: FormEventFieldOption[] = [
      { key: '_widget_phone', label: '联系电话', type: 'text' },
      { key: '_widget_name', label: '姓名', type: 'text' },
    ];
    const wrapper = mount(FormSchemaFrontendEventDebugDrawer, {
      attachTo: document.body,
      props: { modelValue: true, event, fields },
      global: {
        stubs: {
          ElDrawer: { template: '<section><slot /></section>' },
          ElAlert: true,
          ElButton: { template: '<button><slot /></button>' },
          ElEmpty: true,
          ElTag: { template: '<span><slot /></span>' },
        },
      },
    });

    expect(wrapper.text()).toContain('联系电话');
    expect(wrapper.text()).not.toContain('输入姓名');
    await wrapper.get('input').setValue('18223456545');
    await nextTick();
    expect(wrapper.text()).toContain('https://api.lingyanyun.com/member?phone=18223456545');
    wrapper.unmount();
  });
});
