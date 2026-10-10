import type { AppWorkspaceAsset } from '../appWorkspace.types';
import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { markRaw } from 'vue';
import AppWorkspaceAssetItem from '../AppWorkspaceAssetItem.vue';

function dashboardAsset(): AppWorkspaceAsset {
  return {
    code: 'menu_dashboard',
    label: '经营总览',
    icon: markRaw({ render: () => null }),
    iconKey: 'bar-chart-box',
    type: 'dashboard',
    targetCode: 'dashboard_demo',
    formType: null,
    favorited: false,
    capabilities: {
      view: true,
      favorite: true,
      actions: {
        edit: true,
        rename: true,
        switchType: false,
        referenceView: false,
        copyInApp: true,
        copyCrossApp: false,
        move: true,
        hide: false,
        delete: true,
      },
    },
  };
}

describe('app workspace dashboard actions', () => {
  it('exposes the complete authorized dashboard menu in product order', () => {
    const wrapper = shallowMount(AppWorkspaceAssetItem, {
      props: { asset: dashboardAsset(), activeAssetCode: '', depth: 0 },
    });

    const actions = (
      wrapper.vm as unknown as { actionItems: Array<{ action: string; label: string }> }
    ).actionItems;
    expect(actions.map(({ action }) => action)).toEqual([
      'edit',
      'rename',
      'copy-in-app',
      'move',
      'favorite',
      'delete',
    ]);
    expect(actions.map(({ label }) => label)).toEqual([
      '编辑',
      '修改名称和图标',
      '复制',
      '移动',
      '收藏',
      '删除',
    ]);
  });

  it('keeps server capability projection as the button source of truth', () => {
    const asset = dashboardAsset();
    asset.capabilities.actions.copyInApp = false;
    asset.capabilities.actions.move = false;
    const wrapper = shallowMount(AppWorkspaceAssetItem, {
      props: { asset, activeAssetCode: '', depth: 0 },
    });

    const actions = (wrapper.vm as unknown as { actionItems: Array<{ action: string }> })
      .actionItems;
    expect(actions.map(({ action }) => action)).toEqual(['edit', 'rename', 'favorite', 'delete']);
  });
});
