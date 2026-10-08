// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import ProfileDrawer from '../ProfileDrawer.vue';

describe('profile drawer', () => {
  it('从个人设置菜单进入账号设置页', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/account/settings', component: { template: '<div>个人设置页</div>' } },
      ],
    });
    await router.push('/');
    await router.isReady();

    const wrapper = mount(ProfileDrawer, {
      props: { modelValue: true },
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router],
        stubs: {
          VanPopup: { template: '<div><slot /></div>' },
        },
      },
    });
    const settingsButton = wrapper.findAll('button').find((button) => button.text().includes('个人设置'));
    expect(settingsButton).toBeDefined();

    await settingsButton?.trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/account/settings');
  });
});
