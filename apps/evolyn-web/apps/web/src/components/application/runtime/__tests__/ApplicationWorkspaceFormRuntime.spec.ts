import type { ApplicationWorkspaceAsset } from '../../workspace/applicationWorkspace.types';
import type { FormRuntimeBootstrap } from '~/types';
import { flushPromises, mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import ApplicationWorkspaceFormRuntime from '../ApplicationWorkspaceFormRuntime.vue';

const api = vi.hoisted(() => ({
  createFormDataOperationId: vi.fn(() => '8f85ff13-a326-45a6-8d09-12b93cc789b0'),
  getFormRuntime: vi.fn(),
  submitFormRecord: vi.fn(),
}));

vi.mock('~/api/form', () => api);

const SurfaceStub = defineComponent({
  name: 'FormWebRuntimeSurface',
  props: {
    formId: String,
    actions: Array,
    adapter: Object,
  },
  template: '<div class="surface-stub" :data-form-id="formId" />',
});

const StateStub = defineComponent({
  name: 'ElResult',
  setup(_props, { slots }) {
    return () => h('div', { class: 'result-stub' }, slots.extra?.());
  },
});

function asset(targetCode: string): ApplicationWorkspaceAsset {
  return {
    code: `entry_${targetCode}`,
    label: targetCode,
    icon: defineComponent({ render: () => null }),
    iconKey: 'file-list',
    type: 'form',
    targetCode,
    formType: 'standard',
    capabilities: {
      view: true,
      favorite: true,
      actions: {
        edit: true,
        rename: true,
        switchType: false,
        referenceView: false,
        copyInApp: false,
        copyCrossApp: false,
        move: true,
        hide: false,
        delete: true,
      },
    },
  };
}

function bootstrap(formCode: string): FormRuntimeBootstrap {
  return {
    formCode,
    name: formCode,
    publishedVersion: 1,
    schemaRevision: '1',
    protocolVersion: 3,
    content: {
      content: {
        type: 'form',
        layout: 'normal',
        items: [],
        layout_fields: [],
        field_layout: [],
        fieldShowRules: [],
        submitRule: 2,
        widget_submit_rules: {},
      },
    },
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

describe('applicationWorkspaceFormRuntime', () => {
  it('资产切换取消旧请求，且迟到响应不会覆盖当前表单', async () => {
    const first = deferred<FormRuntimeBootstrap>();
    const second = deferred<FormRuntimeBootstrap>();
    const signals: AbortSignal[] = [];
    api.getFormRuntime.mockImplementation(
      (_appCode: string, _formCode: string, signal: AbortSignal) => {
        signals.push(signal);
        return signals.length === 1 ? first.promise : second.promise;
      },
    );

    const wrapper = mount(ApplicationWorkspaceFormRuntime, {
      props: { appCode: 'app_a', asset: asset('form_a') },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        directives: { loading: () => undefined },
        stubs: {
          ElButton: true,
          ElResult: StateStub,
          FormWebRuntimeSurface: SurfaceStub,
        },
      },
    });
    await flushPromises();

    await wrapper.setProps({ asset: asset('form_b') });
    await flushPromises();
    expect(signals[0].aborted).toBe(true);

    second.resolve(bootstrap('form_b'));
    await flushPromises();
    expect(wrapper.findComponent(SurfaceStub).props('formId')).toBe('form_b');

    first.resolve(bootstrap('form_a'));
    await flushPromises();
    expect(wrapper.findComponent(SurfaceStub).props('formId')).toBe('form_b');
  });

  it('应用运行页只展示后端已具备的提交动作', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap('form_a'));
    const wrapper = mount(ApplicationWorkspaceFormRuntime, {
      props: { appCode: 'app_a', asset: asset('form_a') },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        directives: { loading: () => undefined },
        stubs: {
          ElButton: true,
          ElResult: StateStub,
          FormWebRuntimeSurface: SurfaceStub,
        },
      },
    });
    await flushPromises();

    const actions = wrapper.findComponent(SurfaceStub).props('actions') as Array<{
      behavior: string;
    }>;
    expect(actions.map((action) => action.behavior)).toEqual(['submit']);
  });

  it('提交时补齐应用、菜单和幂等上下文，字段值保留 data/visible 包装', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap('form_a'));
    api.submitFormRecord.mockResolvedValue({ recordId: 1 });
    const wrapper = mount(ApplicationWorkspaceFormRuntime, {
      props: { appCode: 'app_a', asset: asset('form_a') },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        directives: { loading: () => undefined },
        stubs: {
          ElButton: true,
          ElResult: StateStub,
          FormWebRuntimeSurface: SurfaceStub,
        },
      },
    });
    await flushPromises();

    const adapter = wrapper.findComponent(SurfaceStub).props('adapter') as {
      submit: (payload: unknown, signal: AbortSignal) => Promise<{ accepted: boolean }>;
    };
    const result = await adapter.submit(
      {
        formId: 'form_a',
        publishedVersion: 1,
        schemaRevision: '1',
        values: { _widget_a: { data: '测试', visible: true } },
      },
      new AbortController().signal,
    );

    expect(result.accepted).toBe(true);
    expect(api.submitFormRecord).toHaveBeenCalledWith(
      {
        appCode: 'app_a',
        entryCode: 'entry_form_a',
        formCode: 'form_a',
        publishedVersion: 1,
        schemaRevision: '1',
        values: { _widget_a: { data: '测试', visible: true } },
        hasResult: true,
        dataOpId: '8f85ff13-a326-45a6-8d09-12b93cc789b0',
      },
      expect.any(AbortSignal),
    );
  });
});
