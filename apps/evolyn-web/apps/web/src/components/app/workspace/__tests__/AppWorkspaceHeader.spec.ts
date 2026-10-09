import { shallowMount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';
import AppWorkspaceHeader from '../AppWorkspaceHeader.vue';

describe('app workspace header asset modes', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  function mountHeader(assetType: 'dashboard' | 'form', canEdit = true) {
    return shallowMount(AppWorkspaceHeader, {
      props: {
        mode: 'fill',
        assetType,
        canEdit,
        sidebarCollapsed: false,
        personalTitle: null,
      },
      global: {
        stubs: {
          MessageCenterDrawer: true,
          UserMenu: true,
        },
      },
    });
  }

  it('shows only the authorized edit action for a dashboard', () => {
    const wrapper = mountHeader('dashboard');
    const labels = wrapper.findAll('.app-workspace-header__mode').map((item) => item.text());

    expect(labels).toEqual(['编辑']);
  });

  it('does not expose an edit action when the dashboard capability is absent', () => {
    const wrapper = mountHeader('dashboard', false);

    expect(wrapper.findAll('.app-workspace-header__mode')).toHaveLength(0);
  });

  it('keeps form fill, edit and data modes', () => {
    const wrapper = mountHeader('form');
    const labels = wrapper.findAll('.app-workspace-header__mode').map((item) => item.text());

    expect(labels).toEqual(['仅添加数据', '编辑', '数据管理']);
  });
});
