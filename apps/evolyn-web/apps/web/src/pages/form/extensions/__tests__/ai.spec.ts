import { flushPromises, shallowMount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, shallowRef } from 'vue';
import { formWorkspaceContextKey } from '../../workspace-context';
import FormAssistantPage from '../ai.vue';

const api = vi.hoisted(() => ({
  listForms: vi.fn(),
}));
const router = vi.hoisted(() => ({
  push: vi.fn(),
}));

vi.mock('~/api/form', () => api);
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { appCode: 'app_hr', formCode: 'form_employee' } }),
  useRouter: () => router,
}));

const ButtonStub = defineComponent({
  name: 'ElButton',
  inheritAttrs: false,
  setup(_props, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.());
  },
});

function mountPage() {
  const detail = shallowRef({
    appId: 1,
    code: 'form_employee',
    name: '员工档案',
    formType: 'standard',
  });
  return shallowMount(FormAssistantPage, {
    global: {
      provide: {
        [formWorkspaceContextKey as symbol]: {
          detail,
          loading: shallowRef(false),
          loadFailed: shallowRef(false),
          renaming: shallowRef(false),
          setDetail: vi.fn(),
          patchDetail: vi.fn(),
          rename: vi.fn(),
          reload: vi.fn(),
        },
      },
      stubs: { ElButton: ButtonStub },
    },
  });
}

describe('form assistant page', () => {
  beforeEach(() => {
    router.push.mockClear();
    api.listForms.mockResolvedValue({
      items: [
        {
          appId: 1,
          code: 'form_employee',
          name: '员工档案',
          formType: 'standard',
          publishedVersion: 1,
          updatedAt: '2026-09-24 10:00:00',
        },
        {
          appId: 1,
          code: 'form_product',
          name: '产品管理',
          formType: 'standard',
          publishedVersion: 1,
          updatedAt: '2026-09-24 10:00:00',
        },
      ],
      nextCursor: '',
      hasMore: false,
    });
  });

  it('确认新建配置后跳转到智能助手设计器', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(api.listForms).toHaveBeenCalledWith({ appId: 1, limit: 100 });

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('新建智能助手 Pro'));
    await createButton!.trigger('click');

    const createDialog = wrapper.findComponent({ name: 'AssistantCreateDialog' });
    expect(createDialog.props('modelValue')).toBe(true);
    createDialog.vm.$emit('create', {
      name: '员工入职同步',
      triggerType: 'form',
      triggerFormCode: 'form_employee',
      triggerFormName: '员工档案',
      tags: ['人事'],
    });
    await wrapper.vm.$nextTick();

    expect(router.push).toHaveBeenCalledWith({
      name: 'form-assistant-designer',
      params: {
        appCode: 'app_hr',
        formCode: 'form_employee',
        assistantId: expect.any(String),
      },
      query: {
        name: '员工入职同步',
        triggerType: 'form',
        triggerFormCode: 'form_employee',
        triggerFormName: '员工档案',
        tags: ['人事'],
      },
    });
  });
});
