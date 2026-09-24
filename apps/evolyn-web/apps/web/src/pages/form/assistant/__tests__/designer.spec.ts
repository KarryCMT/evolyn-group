import { shallowMount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { reactive } from 'vue';
import AssistantDesignerPage from '../designer.vue';

const api = vi.hoisted(() => ({
  getForm: vi.fn(),
  getFormRuntime: vi.fn(),
  listForms: vi.fn(),
  listMembers: vi.fn(),
  getDepartmentTree: vi.fn(),
}));

const router = vi.hoisted(() => ({
  push: vi.fn(),
}));
const route = reactive({
  params: {
    appCode: 'app_hr',
    formCode: 'form_employee',
    assistantId: 'assistant_1',
  },
  query: {
    name: '员工入职同步',
    triggerType: 'form',
    triggerFormCode: 'form_employee',
    triggerFormName: '员工档案',
  },
});

vi.mock('~/api/form', () => ({
  getForm: api.getForm,
  getFormRuntime: api.getFormRuntime,
  listForms: api.listForms,
}));
vi.mock('~/api/member', () => ({ listMembers: api.listMembers }));
vi.mock('~/api/department', () => ({ getDepartmentTree: api.getDepartmentTree }));

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => router,
}));

describe('assistant designer page', () => {
  beforeEach(() => {
    router.push.mockClear();
    route.params.assistantId = 'assistant_1';
    route.query.name = '员工入职同步';
    route.query.triggerType = 'form';
    route.query.triggerFormCode = 'form_employee';
    route.query.triggerFormName = '员工档案';
    const form = {
      appId: 1,
      code: 'form_employee',
      name: '员工档案',
      formType: 'standard',
      draft: {
        content: {
          items: [],
          fieldFormulas: [],
        },
      },
    };
    api.getForm.mockResolvedValue(form);
    api.getFormRuntime.mockRejectedValue(new Error('not published'));
    api.listForms.mockResolvedValue({ items: [], nextCursor: '', hasMore: false });
    api.listMembers.mockResolvedValue({ items: [], total: 0 });
    api.getDepartmentTree.mockResolvedValue([]);
  });

  it('将路由草稿转换为包内画布文档并可返回智能助手列表', async () => {
    const wrapper = shallowMount(AssistantDesignerPage);
    const designer = wrapper.findComponent({ name: 'IntelligentDesigner' });
    const document = designer.props('document');

    expect(document.name).toBe('员工入职同步');
    expect(document.trigger.formName).toBe('员工档案');
    expect(document.nodes.map((node: { type: string }) => node.type)).toEqual(['trigger', 'end']);

    designer.vm.$emit('back');
    await wrapper.vm.$nextTick();

    expect(router.push).toHaveBeenCalledWith({
      name: 'form-extension-ai',
      params: {
        appCode: 'app_hr',
        formCode: 'form_employee',
      },
    });
  });

  it('同一路由切换助手参数时重建画布文档', async () => {
    const wrapper = shallowMount(AssistantDesignerPage);

    route.params.assistantId = 'assistant_2';
    route.query.name = '产品数据同步';
    route.query.triggerFormName = '产品管理';
    await wrapper.vm.$nextTick();

    const document = wrapper.findComponent({ name: 'IntelligentDesigner' }).props('document');
    expect(document.id).toBe('assistant_2');
    expect(document.name).toBe('产品数据同步');
    expect(document.trigger.formName).toBe('产品管理');
  });
});
