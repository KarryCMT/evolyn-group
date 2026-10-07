import { describe, expect, it } from 'vitest';
import appRoutes from '~/router/modules/app';
import {
  appWorkspaceRoute,
  findWorkspaceAssetByTargetCode,
  resolveWorkspaceAssetRoute,
} from '../appWorkspaceNavigation';

describe('application workspace asset state', () => {
  it('uses a generic asset code in the application route', () => {
    expect(appRoutes.find((route) => route.name === 'App')?.path).toBe(
      '/app/:appCode/:assetCode?',
    );
    expect(appWorkspaceRoute('app_demo', 'dashboard_demo')).toEqual({
      name: 'App',
      params: { appCode: 'app_demo', assetCode: 'dashboard_demo' },
    });
  });

  it('restores a nested dashboard by its stable public code', () => {
    const dashboard = {
      code: 'menu_dashboard',
      type: 'dashboard',
      targetCode: 'dashboard_demo',
    } as Parameters<typeof findWorkspaceAssetByTargetCode>[0][number];
    const assets = [
      {
        code: 'menu_group',
        type: 'folder',
        targetCode: null,
        children: [dashboard],
      },
    ] as Parameters<typeof findWorkspaceAssetByTargetCode>[0];

    expect(findWorkspaceAssetByTargetCode(assets, 'dashboard_demo')).toBe(dashboard);
  });
});

describe('resolveWorkspaceAssetRoute', () => {
  it('opens the selected dashboard design workspace from the edit mode', () => {
    expect(
      resolveWorkspaceAssetRoute(
        'app_demo',
        { type: 'dashboard', targetCode: 'dashboard_demo' },
        'design',
      ),
    ).toEqual({
      name: 'dashboard-design',
      params: { appCode: 'app_demo', dashboardCode: 'dashboard_demo' },
    });
  });

  it.each([
    ['design', 'form-design'],
    ['data', 'form-data'],
  ] as const)('keeps form %s navigation unchanged', (mode, routeName) => {
    expect(
      resolveWorkspaceAssetRoute('app_demo', { type: 'form', targetCode: 'form_demo' }, mode),
    ).toEqual({
      name: routeName,
      params: { appCode: 'app_demo', formCode: 'form_demo' },
    });
  });

  it('does not expose form data management for dashboards', () => {
    expect(
      resolveWorkspaceAssetRoute(
        'app_demo',
        { type: 'dashboard', targetCode: 'dashboard_demo' },
        'data',
      ),
    ).toBeNull();
  });

  it('rejects assets without a stable public code', () => {
    expect(
      resolveWorkspaceAssetRoute('app_demo', { type: 'dashboard', targetCode: null }, 'design'),
    ).toBeNull();
  });
});
