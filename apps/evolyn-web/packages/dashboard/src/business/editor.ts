import { type Ref, computed, shallowRef } from 'vue';
import {
  BUSINESS_DASHBOARD_COLUMNS,
  type BusinessDashboardDocument,
  type BusinessDashboardLayout,
  type BusinessDashboardWidget,
  type BusinessDashboardWidgetDescriptor,
} from './types.js';

export interface UseBusinessDashboardEditorOptions {
  document: Ref<BusinessDashboardDocument>;
  createID?: () => string;
}

/** 业务编辑动作集中在此，页面和展示组件都不直接修改文档内部引用。 */
export function useBusinessDashboardEditor(options: UseBusinessDashboardEditorOptions) {
  const selectedWidgetId = shallowRef<string | null>(null);
  const selectedWidget = computed(
    () => options.document.value.widgets.find((item) => item.id === selectedWidgetId.value) ?? null,
  );

  function addWidget(descriptor: BusinessDashboardWidgetDescriptor) {
    const layout = findAvailablePosition(options.document.value.widgets, descriptor.defaultLayout);
    const widget: BusinessDashboardWidget = {
      id: options.createID?.() ?? createWidgetID(),
      type: descriptor.type,
      title: descriptor.defaultTitle,
      layout,
      settings: { ...descriptor.defaultSettings },
    };
    replaceWidgets([...options.document.value.widgets, widget]);
    selectedWidgetId.value = widget.id;
  }

  function selectWidget(id: string | null) {
    selectedWidgetId.value =
      id && options.document.value.widgets.some((item) => item.id === id) ? id : null;
  }

  function updateWidget(
    id: string,
    patch: Partial<Pick<BusinessDashboardWidget, 'title' | 'settings'>>,
  ) {
    replaceWidgets(
      options.document.value.widgets.map((widget) =>
        widget.id === id
          ? {
              ...widget,
              ...patch,
              ...(patch.settings ? { settings: { ...patch.settings } } : {}),
            }
          : widget,
      ),
    );
  }

  function updateLayout(id: string, layout: BusinessDashboardLayout) {
    replaceWidgets(
      options.document.value.widgets.map((widget) =>
        widget.id === id ? { ...widget, layout: { ...layout } } : widget,
      ),
    );
  }

  function replaceLayouts(layouts: Array<{ id: string; layout: BusinessDashboardLayout }>) {
    const byID = new Map(layouts.map((item) => [item.id, item.layout]));
    replaceWidgets(
      options.document.value.widgets.map((widget) => {
        const layout = byID.get(widget.id);
        return layout ? { ...widget, layout: { ...layout } } : widget;
      }),
    );
  }

  function removeWidget(id: string) {
    replaceWidgets(options.document.value.widgets.filter((widget) => widget.id !== id));
    if (selectedWidgetId.value === id) selectedWidgetId.value = null;
  }

  function replaceWidgets(widgets: BusinessDashboardWidget[]) {
    options.document.value = { ...options.document.value, widgets };
  }

  return {
    selectedWidgetId,
    selectedWidget,
    addWidget,
    selectWidget,
    updateWidget,
    updateLayout,
    replaceLayouts,
    removeWidget,
  };
}

function findAvailablePosition(
  widgets: BusinessDashboardWidget[],
  size: Pick<BusinessDashboardLayout, 'w' | 'h'>,
): BusinessDashboardLayout {
  const width = Math.min(Math.max(size.w, 1), BUSINESS_DASHBOARD_COLUMNS);
  const bottom = widgets.reduce((value, item) => Math.max(value, item.layout.y + item.layout.h), 0);
  for (let y = 0; y <= bottom; y += 1) {
    for (let x = 0; x <= BUSINESS_DASHBOARD_COLUMNS - width; x += 1) {
      const overlaps = widgets.some(
        (item) =>
          x < item.layout.x + item.layout.w &&
          x + width > item.layout.x &&
          y < item.layout.y + item.layout.h &&
          y + size.h > item.layout.y,
      );
      if (!overlaps) return { x, y, w: width, h: size.h };
    }
  }
  return { x: 0, y: bottom, w: width, h: size.h };
}

function createWidgetID(): string {
  const value =
    globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `widget_${value.replaceAll('-', '')}`;
}
