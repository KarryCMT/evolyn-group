import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, nextTick } from 'vue';
import FormSchemaFrontendEventDebugDrawer from '../FormSchemaFrontendEventDebugDrawer.vue';
import { createFormEvent, type FormEventFieldOption } from '../frontend-events';

const DrawerStub = defineComponent({
  name: 'ElDrawer',
  props: {
    direction: String,
    size: String,
  },
  template: '<section><slot /></section>',
});

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
          ElDrawer: DrawerStub,
          ElAlert: true,
          ElButton: { template: '<button><slot /></button>' },
          ElEmpty: true,
          ElTag: { template: '<span><slot /></span>' },
        },
      },
    });

    expect(wrapper.text()).toContain('联系电话');
    expect(wrapper.text()).not.toContain('输入姓名');
    expect(wrapper.getComponent(DrawerStub).props()).toMatchObject({
      direction: 'btt',
      size: '90%',
    });
    await wrapper.get('input').setValue('18223456545');
    await nextTick();
    expect(wrapper.text()).toContain('https://api.lingyanyun.com/member?phone=18223456545');
    wrapper.unmount();
  });

  it('sends values through the injected backend executor and renders the latest result', async () => {
    const event = createFormEvent();
    event.request.url = 'https://api.lingyanyun.com/member?phone=${_widget_phone}';
    const execute = vi.fn().mockResolvedValue({
      sequence: 1,
      writes: { _widget_name: '测试员工' },
      requestSummary: {
        method: 'GET',
        url: 'https://api.lingyanyun.com/member?phone=18223456545',
        headerNames: [],
      },
      responseSummary: {
        statusCode: 200,
        durationMs: 18,
        format: 'json',
        body: '{"name":"测试员工"}',
      },
    });
    const wrapper = mount(FormSchemaFrontendEventDebugDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        event,
        fields: [
          { key: '_widget_phone', label: '联系电话', type: 'text' },
          { key: '_widget_name', label: '姓名', type: 'text' },
        ],
        execute,
      },
      global: {
        stubs: {
          ElDrawer: DrawerStub,
          ElAlert: true,
          ElButton: { template: '<button><slot /></button>' },
          ElEmpty: true,
        },
      },
    });
    await wrapper.get('input').setValue('18223456545');
    await wrapper.get('.form-event-debug-drawer__send-bar button').trigger('click');
    await flushPromises();
    expect(execute).toHaveBeenCalledWith(event.id, {
      values: { _widget_phone: '18223456545' },
      sequence: 1,
    });
    expect(wrapper.text()).toContain('200 · 18 ms');
    expect(wrapper.text()).toContain('测试员工');
    wrapper.unmount();
  });
});
