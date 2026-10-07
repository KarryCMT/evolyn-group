import type { DashboardDetail } from '~/types';
import {
  createDashboardRequestId,
  DEFAULT_DASHBOARD_NAME,
  precreateDashboard,
} from '~/api/dashboard';

export interface DashboardPrecreateDependencies {
  create: typeof precreateDashboard;
  createRequestId: typeof createDashboardRequestId;
  navigate: (detail: DashboardDetail) => Promise<unknown>;
}

export interface DashboardPrecreateOutcome {
  detail: DashboardDetail;
  navigationFailed: boolean;
}

/**
 * 把“创建成功”与“导航成功”分成两个结果阶段。导航异常不会触发创建重试，
 * 页面可据此明确提示资产已存在并引导从应用菜单重新进入。
 */
export async function precreateDashboardAsset(
  appCode: string,
  parentMenuCode: string | undefined,
  navigate: DashboardPrecreateDependencies['navigate'],
  dependencies: Omit<DashboardPrecreateDependencies, 'navigate'> = {
    create: precreateDashboard,
    createRequestId: createDashboardRequestId,
  },
): Promise<DashboardPrecreateOutcome> {
  const requestId = dependencies.createRequestId();
  const detail = await dependencies.create(appCode, {
    requestId,
    name: DEFAULT_DASHBOARD_NAME,
    ...(parentMenuCode ? { parentMenuCode } : {}),
  });

  try {
    await navigate(detail);
    return { detail, navigationFailed: false };
  } catch {
    return { detail, navigationFailed: true };
  }
}
