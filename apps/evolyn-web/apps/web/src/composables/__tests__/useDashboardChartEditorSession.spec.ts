import { businessDashboardWidgetDescriptors } from '@evolyn.do/dashboard';
import { describe, expect, it, vi } from 'vitest';
import { useDashboardChartEditorSession } from '~/composables/useDashboardChartEditorSession';

describe('dashboard chart editor session', () => {
  it('keeps a new chart isolated until the editor commits it', () => {
    const commit = vi.fn();
    const session = useDashboardChartEditorSession({
      commit,
      createID: vi.fn().mockReturnValueOnce('chart').mockReturnValueOnce('source'),
    });
    const descriptor = businessDashboardWidgetDescriptors.find((item) => item.type === 'chart')!;
    const draft = session.beginCreate({
      descriptor,
      layout: { x: 0, y: 0, ...descriptor.defaultLayout },
      source: {
        code: 'form_employee',
        name: '员工档案',
        publishedVersion: 1,
        schemaRevision: 'schema-1',
      },
    });

    expect(commit).not.toHaveBeenCalled();
    expect(draft.widget.datasetId).toBe('dataset_source');
    expect(session.isDirty.value).toBe(true);

    session.updateWidget((widget) => ({ ...widget, title: '员工统计' }));
    expect(session.isDirty.value).toBe(true);

    session.commit();
    expect(commit).toHaveBeenCalledOnce();
    expect(commit.mock.calls[0]?.[0]).toMatchObject({ title: '员工统计' });
    expect(session.draft.value).toBeNull();
  });

  it('clears incompatible bindings when the data source changes', () => {
    const session = useDashboardChartEditorSession({ commit: vi.fn() });
    const descriptor = businessDashboardWidgetDescriptors.find((item) => item.type === 'chart')!;
    session.beginCreate({
      descriptor,
      layout: { x: 0, y: 0, ...descriptor.defaultLayout },
      source: {
        code: 'form_employee',
        name: '员工档案',
        publishedVersion: 1,
        schemaRevision: 'schema-1',
      },
    });
    session.updateWidget((widget) => ({
      ...widget,
      settings: {
        ...widget.settings,
        encoding: { dimensions: [{ field: { fieldId: 'name' } }], metrics: [] },
      },
    }));

    session.replaceSource({
      code: 'form_order',
      name: '订单管理',
      publishedVersion: 2,
      schemaRevision: 'schema-2',
    });

    expect(session.draft.value?.dataset.source.formCode).toBe('form_order');
    expect(session.draft.value?.widget.settings.encoding).toEqual({ dimensions: [], metrics: [] });
  });
});
