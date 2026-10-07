import type { DashboardDetail } from '~/types';
import { describe, expect, it, vi } from 'vitest';
import { precreateDashboardAsset } from '../precreateDashboardAsset';

const detail: DashboardDetail = {
  appId: 7,
  appCode: 'app_demo',
  code: 'dashboard_0123456789abcdef',
  name: '未命名仪表盘',
  icon: '',
  color: '',
  protocolVersion: 1,
  draftRevision: 1,
  draft: {
    version: 1,
    settings: { desktop: { columns: 12, rowHeight: 80 } },
    datasets: [],
    widgets: [],
    filters: [],
    interactions: [],
    publishScope: { type: 'all' },
  },
  publishedVersion: 0,
  published: { version: 0, publishedAt: null },
  createdAt: '2026-09-25 10:00:00',
  updatedAt: '2026-09-25 10:00:00',
};

describe('precreateDashboardAsset', () => {
  it('uses one request id and navigates with the returned public code', async () => {
    const create = vi.fn().mockResolvedValue(detail);
    const createRequestId = vi.fn().mockReturnValue('request-1');
    const navigate = vi.fn().mockResolvedValue(undefined);

    const outcome = await precreateDashboardAsset('app_demo', 'menu_group', navigate, {
      create,
      createRequestId,
    });

    expect(createRequestId).toHaveBeenCalledTimes(1);
    expect(create).toHaveBeenCalledOnce();
    expect(create).toHaveBeenCalledWith('app_demo', {
      requestId: 'request-1',
      name: '未命名仪表盘',
      parentMenuCode: 'menu_group',
    });
    expect(navigate).toHaveBeenCalledWith(detail);
    expect(outcome).toEqual({ detail, navigationFailed: false });
  });

  it('reports navigation failure without submitting another create request', async () => {
    const create = vi.fn().mockResolvedValue(detail);
    const navigate = vi.fn().mockRejectedValue(new Error('navigation aborted'));

    const outcome = await precreateDashboardAsset('app_demo', undefined, navigate, {
      create,
      createRequestId: () => 'request-2',
    });

    expect(create).toHaveBeenCalledOnce();
    expect(outcome).toEqual({ detail, navigationFailed: true });
  });
});
