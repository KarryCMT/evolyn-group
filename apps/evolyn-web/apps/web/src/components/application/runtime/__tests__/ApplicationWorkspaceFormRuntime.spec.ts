import type { ApplicationWorkspaceAsset } from '../../workspace/applicationWorkspace.types';
import type { FormRuntimeBootstrap } from '~/types';
import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import ApplicationWorkspaceFormRuntime from '../ApplicationWorkspaceFormRuntime.vue';

const api = vi.hoisted(() => ({
  getFormRuntime: vi.fn(),
  submitFormRecord: vi.fn(),
}));

vi.mock('~/api/form', () => api);

const SurfaceStub = defineComponent({
  name: 'FormRuntimeSurface',
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
    type: 'form',
    targetCode,
    formType: 'standard',
    capabilities: { view: true, manage: true, move: true, delete: true, favorite: true },
  };
}

function bootstrap(formCode: string): FormRuntimeBootstrap {
  return {
    formCode,
    name: formCode,
    publishedVersion: 1,
    schemaRevision: '1',
    content: { content: { type: 'form', items: [] } },
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
        directives: { loading: () => undefined },
        stubs: {
          ElButton: true,
          ElResult: StateStub,
          FormRuntimeSurface: SurfaceStub,
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
        directives: { loading: () => undefined },
        stubs: {
          ElButton: true,
          ElResult: StateStub,
          FormRuntimeSurface: SurfaceStub,
        },
      },
    });
    await flushPromises();

    const actions = wrapper.findComponent(SurfaceStub).props('actions') as Array<{
      behavior: string;
    }>;
    expect(actions.map((action) => action.behavior)).toEqual(['submit']);
  });
});
