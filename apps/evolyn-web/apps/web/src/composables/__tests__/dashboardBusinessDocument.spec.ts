import type { BusinessDashboardDocument } from '@evolyn.do/dashboard';
import {
  businessDashboardWidgetDescriptors,
  createEmptyBusinessDashboardDocument,
  normalizeBusinessDashboardDocument,
  useBusinessDashboardEditor,
} from '@evolyn.do/dashboard';
import { describe, expect, it } from 'vitest';
import { shallowRef } from 'vue';

describe('business dashboard document', () => {
  it('starts with a real empty canvas and no sample widgets', () => {
    const document = createEmptyBusinessDashboardDocument();
    expect(document.widgets).toEqual([]);
    expect(document.datasets).toEqual([]);
    expect(document.settings.desktop).toEqual({ columns: 12, rowHeight: 80 });
    expect(normalizeBusinessDashboardDocument(document)).toEqual({ document, issues: [] });
  });

  it('rejects unknown widgets and invalid layouts with stable paths', () => {
    const input = createEmptyBusinessDashboardDocument() as unknown as {
      widgets: Array<Record<string, unknown>>;
    };
    input.widgets = [{ id: 'broken', type: 'script', layout: { x: 11, y: 0, w: 2, h: 1 } }];
    const result = normalizeBusinessDashboardDocument(input);
    expect(result.document).toBeNull();
    expect(result.issues.map((issue) => [issue.path, issue.code])).toEqual([
      ['widgets[0].type', 'DASHBOARD_WIDGET_TYPE_UNSUPPORTED'],
      ['widgets[0].layout', 'DASHBOARD_LAYOUT_INVALID'],
    ]);
  });

  it('supports add, select, move, resize, property update and delete', () => {
    const document = shallowRef<BusinessDashboardDocument>(createEmptyBusinessDashboardDocument());
    const editor = useBusinessDashboardEditor({ document, createID: () => 'widget_chart' });
    editor.addWidget(businessDashboardWidgetDescriptors[0]);
    expect(editor.selectedWidgetId.value).toBe('widget_chart');
    expect(document.value.widgets[0].layout).toEqual({ x: 0, y: 0, w: 6, h: 4 });

    editor.replaceLayouts([{ id: 'widget_chart', layout: { x: 3, y: 2, w: 8, h: 5 } }]);
    editor.updateWidget('widget_chart', { title: '经营趋势' });
    expect(document.value.widgets[0]).toMatchObject({
      title: '经营趋势',
      layout: { x: 3, y: 2, w: 8, h: 5 },
    });

    editor.removeWidget('widget_chart');
    expect(document.value.widgets).toEqual([]);
    expect(editor.selectedWidgetId.value).toBeNull();
  });
});
