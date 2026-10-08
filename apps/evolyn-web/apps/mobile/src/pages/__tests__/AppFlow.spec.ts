// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import AppPage from '../AppPage.vue';
import FormRuntimePage from '../FormRuntimePage.vue';

function createAppRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/apps/:appCode', name: 'mobile-app', component: AppPage },
      {
        path: '/apps/:appCode/forms/:formCode',
        name: 'mobile-form',
        component: FormRuntimePage,
      },
    ],
  });
}

describe('mobile application flow', () => {
  it('从应用目录打开对应表单', async () => {
    const router = createAppRouter();
    await router.push('/apps/demo-simple?name=灵衍云示例应用');
    await router.isReady();
    const wrapper = mount(AppPage, {
      global: { plugins: [createPinia(), router] },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('订单管理');
    const orderButton = wrapper.findAll('.app-menu-list > button').find((item) => item.text().includes('订单管理'));
    expect(orderButton).toBeDefined();
    await orderButton?.trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.name).toBe('mobile-form');
    expect(router.currentRoute.value.params.formCode).toBe('form_order');
  });

  it('应用底部可以切换到流程中心', async () => {
    const router = createAppRouter();
    await router.push('/apps/demo-simple?name=灵衍云示例应用');
    await router.isReady();
    const wrapper = mount(AppPage, {
      global: { plugins: [createPinia(), router] },
    });
    await flushPromises();

    const workflowTab = wrapper.findAll('.app-tabs button').find((item) => item.text().includes('流程中心'));
    expect(workflowTab).toBeDefined();
    await workflowTab?.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('我的待办');
    expect(wrapper.text()).toContain('我处理的');
    expect(wrapper.text()).toContain('我发起的');
    expect(wrapper.text()).not.toContain('请求失败');
  });

  it('演示表单可以保存并恢复隔离草稿', async () => {
    localStorage.clear();
    const router = createAppRouter();
    await router.push(
      '/apps/demo-simple/forms/form_order?menuCode=menu_order&name=订单管理&formType=workflow',
    );
    await router.isReady();
    const wrapper = mount(FormRuntimePage, {
      attachTo: document.body,
      global: { plugins: [createPinia(), router] },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('发起流程');
    expect(wrapper.text()).toContain('保存草稿');
    await wrapper.get('[data-action-key="draft"]').trigger('click');
    await flushPromises();

    const key = Object.keys(localStorage).find((item) => item.includes('form-draft') && item.includes('form_order'));
    expect(key).toBeDefined();
  });
});
