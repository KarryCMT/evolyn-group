import type { DashboardDetail } from '~/types';
import {
  businessDashboardWidgetDescriptors,
  createEmptyBusinessDashboardDocument,
} from '@evolyn.do/dashboard';
import { describe, expect, it, vi } from 'vitest';
import { prepareDashboardPreview, useDashboardDesigner } from '~/composables/useDashboardDesigner';

describe('dashboard design flow e2e', () => {
  it('loads an empty asset, edits, saves and hands the confirmed revision to preview', async () => {
    const empty = createEmptyBusinessDashboardDocument();
    const detail: DashboardDetail = {
      appId: 1,
      appCode: 'app_demo',
      code: 'dashboard_demo',
      name: '经营驾驶舱',
      icon: '',
      color: '',
      protocolVersion: 1,
      draftRevision: 1,
      draft: empty,
      publishedVersion: 0,
      published: { version: 0, publishedAt: null },
      createdAt: '2026-10-07 10:00:00',
      updatedAt: '2026-10-07 10:00:00',
    };
    const save = vi.fn(async (_code, revision, document) => ({
      draftRevision: revision + 1,
      document,
    }));
    const designer = useDashboardDesigner({ get: vi.fn().mockResolvedValue(detail), save });

    expect(await designer.load(detail.code)).toBe(true);
    expect(designer.editingDocument.value.widgets).toEqual([]);
    designer.addWidget(businessDashboardWidgetDescriptors[0]);
    designer.updateWidget(designer.selectedWidgetId.value!, { title: '月度经营趋势' });

    const revision = await prepareDashboardPreview({
      dirty: designer.isDirty.value,
      revision: designer.draftRevision.value,
      save: () => designer.save(detail.code),
    });
    expect(revision).toBe(2);
    expect(save).toHaveBeenCalledOnce();
    expect(designer.serverSnapshot.value?.widgets[0].title).toBe('月度经营趋势');
    expect(designer.isDirty.value).toBe(false);
  });
});
