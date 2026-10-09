import type { BusinessDashboardDocument } from '@evolyn.do/dashboard';
import type {DashboardDesignerGateway} from '../useDashboardDesigner';
import type { DashboardDetail } from '~/types';
import {
  businessDashboardWidgetDescriptors,
  createEmptyBusinessDashboardDocument,
} from '@evolyn.do/dashboard';
import { ApiError } from '@evolyn.do/utils';
import { describe, expect, it, vi } from 'vitest';
import {
  
  prepareDashboardPreview,
  useDashboardDesigner
} from '../useDashboardDesigner';

function detail(
  document: BusinessDashboardDocument = createEmptyBusinessDashboardDocument(),
): DashboardDetail {
  return {
    appId: 1,
    appCode: 'app_demo',
    code: 'dashboard_demo',
    name: '经营驾驶舱',
    icon: '',
    color: '',
    protocolVersion: 1,
    draftRevision: 1,
    draft: document,
    publishedVersion: 0,
    published: { version: 0, publishedAt: null },
    createdAt: '2026-10-07 10:00:00',
    updatedAt: '2026-10-07 10:00:00',
  };
}

function gateway(save: DashboardDesignerGateway['save']): DashboardDesignerGateway {
  return { get: vi.fn().mockResolvedValue(detail()), save };
}

describe('useDashboardDesigner', () => {
  it('advances the server snapshot only after save succeeds', async () => {
    const save = vi.fn(async (_code, revision, document) => ({
      draftRevision: revision + 1,
      document,
    }));
    const designer = useDashboardDesigner(gateway(save));
    await designer.load('dashboard_demo');
    designer.addWidget(businessDashboardWidgetDescriptors[0]);
    expect(designer.isDirty.value).toBe(true);
    expect(designer.serverSnapshot.value?.widgets).toEqual([]);

    await designer.save('dashboard_demo');
    expect(designer.draftRevision.value).toBe(2);
    expect(designer.isDirty.value).toBe(false);
    expect(designer.isSaved.value).toBe(true);
    expect(designer.serverSnapshot.value?.widgets).toHaveLength(1);
  });

  it('keeps local content on 409 and reloads only after an explicit action', async () => {
    const latest = detail();
    latest.draftRevision = 2;
    const get = vi.fn().mockResolvedValueOnce(detail()).mockResolvedValueOnce(latest);
    const designer = useDashboardDesigner({
      get,
      save: vi
        .fn()
        .mockRejectedValue(new ApiError('仪表盘已被他人更新', 409, 'DASHBOARD_DRAFT_CONFLICT')),
    });
    await designer.load('dashboard_demo');
    designer.addWidget(businessDashboardWidgetDescriptors[0]);
    const local = JSON.stringify(designer.editingDocument.value);

    expect(await designer.save('dashboard_demo')).toBeNull();
    expect(designer.saveStatus.value).toBe('conflict');
    expect(JSON.stringify(designer.editingDocument.value)).toBe(local);
    expect(designer.isDirty.value).toBe(true);

    await designer.reloadServerVersion('dashboard_demo');
    expect(designer.draftRevision.value).toBe(2);
    expect(designer.editingDocument.value.widgets).toEqual([]);
    expect(designer.isDirty.value).toBe(false);
  });

  it('focuses a widget from a server path issue without discarding edits', async () => {
    const designer = useDashboardDesigner(
      gateway(
        vi.fn().mockRejectedValue(
          new ApiError('协议错误', 400, 'DASHBOARD_SCHEMA_INVALID', {
            issues: [
              { path: 'widgets[0].layout', code: 'DASHBOARD_LAYOUT_INVALID', message: '布局无效' },
            ],
          }),
        ),
      ),
    );
    await designer.load('dashboard_demo');
    designer.addWidget(businessDashboardWidgetDescriptors[0]);
    const id = designer.editingDocument.value.widgets[0].id;

    await designer.save('dashboard_demo');
    expect(designer.selectedWidgetId.value).toBe(id);
    expect(designer.focusedIssuePath.value).toBe('widgets[0].layout');
    expect(designer.editingDocument.value.widgets).toHaveLength(1);
  });

  it('patches asset metadata without replacing unsaved canvas content', async () => {
    const designer = useDashboardDesigner(gateway(vi.fn()));
    await designer.load('dashboard_demo');
    designer.addWidget(businessDashboardWidgetDescriptors[0]);
    const localDocument = designer.editingDocument.value;

    designer.patchDetail({ name: '销售分析看板', updatedAt: '2026-10-09 08:00:00' });

    expect(designer.detail.value?.name).toBe('销售分析看板');
    expect(designer.editingDocument.value).toBe(localDocument);
    expect(designer.editingDocument.value.widgets).toHaveLength(1);
    expect(designer.isDirty.value).toBe(true);
  });
});

describe('prepareDashboardPreview', () => {
  it('saves dirty content before returning the preview revision', async () => {
    const save = vi.fn().mockResolvedValue({
      draftRevision: 5,
      document: createEmptyBusinessDashboardDocument(),
    });
    await expect(prepareDashboardPreview({ dirty: true, revision: 4, save })).resolves.toBe(5);
    expect(save).toHaveBeenCalledOnce();
  });

  it('stays on the design page when pre-preview save fails', async () => {
    await expect(
      prepareDashboardPreview({
        dirty: true,
        revision: 4,
        save: vi.fn().mockResolvedValue(null),
      }),
    ).resolves.toBeNull();
  });
});
