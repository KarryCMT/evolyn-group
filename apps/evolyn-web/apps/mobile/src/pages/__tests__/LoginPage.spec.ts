// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import LoginPage from '../LoginPage.vue';

async function mountLoginPage() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/auth/login', component: LoginPage }],
  });
  await router.push('/auth/login');
  await router.isReady();

  return mount(LoginPage, {
    global: { plugins: [createPinia(), router] },
  });
}

describe('login page', () => {
  it('仅提供验证码和密码登录，不展示注册入口', async () => {
    const wrapper = await mountLoginPage();

    expect(wrapper.text()).toContain('获取验证码');
    expect(wrapper.text()).toContain('密码登录');
    expect(wrapper.text()).not.toContain('注册');

    await wrapper.get('.login-form__switch').trigger('click');

    expect(wrapper.get('input[name="password"]').attributes('name')).toBe('password');
    expect(wrapper.text()).toContain('验证码登录');
    expect(wrapper.text()).toContain('忘记密码？');
  });
});
