import type { UserInfoResult } from '~/types';
import { createWidgetItem } from '@evolyn.do/form/schema';
import { flushPromises, mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import DepartmentSelectionField from '../DepartmentSelectionField.vue';
import { useAuthStore } from '~/stores/auth';

const { loadDepartmentOptions } = vi.hoisted(() => ({ loadDepartmentOptions: vi.fn() }));

vi.mock('../departmentOptions', () => ({ loadDepartmentOptions }));

afterEach(() => {
  document.body.innerHTML = '';
  vi.clearAllMocks();
});

describe('DepartmentSelectionField', () => {
  it('登录聚合中的 departments 为 null 时仍可渲染预览', () => {
    const pinia = createPinia();
    const auth = useAuthStore(pinia);
    // 兼容历史租户的异常聚合数据：该字段只影响“当前用户所在部门”辅助页签，
    // 不能阻断设计器预览的字段树渲染。
    auth.userInfo = {
      tenant: { id: 1 },
      member: { departments: null },
    } as unknown as UserInfoResult;

    const wrapper = mount(DepartmentSelectionField, {
      props: {
        item: createWidgetItem('dept'),
        modelValue: null,
        disabled: false,
        readonly: false,
        errors: [],
      },
      global: { plugins: [pinia] },
    });

    expect(wrapper.get('.form-department-selection__placeholder').text()).toContain('选择部门');
    expect(wrapper.get('.form-department-selection__control').classes()).toContain(
      'form-department-selection__control',
    );
  });

  it('打开选择器时补齐登录资料并展示当前用户部门页签', async () => {
    loadDepartmentOptions.mockResolvedValue([
      { value: '3', label: '产品部', disabled: false },
      { value: '4', label: '研发部', disabled: false },
    ]);
    const pinia = createPinia();
    const auth = useAuthStore(pinia);
    auth.userInfo = { tenant: { id: 1 }, member: { departments: null } } as unknown as UserInfoResult;
    vi.spyOn(auth, 'loadUserInfo').mockImplementation(async () => {
      auth.userInfo = {
        tenant: { id: 1 },
        member: { departments: [{ id: 3, name: '产品部' }] },
      } as unknown as UserInfoResult;
      return auth.userInfo;
    });

    const wrapper = mount(DepartmentSelectionField, {
      attachTo: document.body,
      props: {
        item: createWidgetItem('dept'),
        modelValue: null,
        disabled: false,
        readonly: false,
        errors: [],
      },
      global: { plugins: [pinia] },
    });

    await wrapper.get('.form-department-selection__control').trigger('click');
    await flushPromises();
    const currentMemberTab = [...document.body.querySelectorAll<HTMLButtonElement>('[role="tab"]')].find(
      (tab) => tab.textContent?.trim() === '当前用户所在部门',
    );
    expect(auth.loadUserInfo).toHaveBeenCalledOnce();
    expect(currentMemberTab).toBeDefined();
    expect(loadDepartmentOptions).toHaveBeenCalledWith('1');
  });
});
