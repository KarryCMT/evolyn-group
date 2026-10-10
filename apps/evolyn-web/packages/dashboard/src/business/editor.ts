import { type Ref, computed, shallowRef } from 'vue';
import {
  BUSINESS_DASHBOARD_COLUMNS,
  type BusinessDashboardDataset,
  type BusinessDashboardDatasetPatch,
  type BusinessDashboardDocument,
  type BusinessDashboardLayout,
  type BusinessDashboardWidget,
  type BusinessDashboardWidgetDescriptor,
  type BusinessDashboardWidgetPatch,
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
    const layout = findAvailableWidgetPosition(
      options.document.value.widgets,
      descriptor.defaultLayout,
    );
    return addWidgetAtLayout(descriptor, layout);
  }

  /** 外部拖入已由 GridStack 完成碰撞计算，此处只收敛边界并生成正式业务组件。 */
  function addWidgetAtLayout(
    descriptor: BusinessDashboardWidgetDescriptor,
    requestedLayout: BusinessDashboardLayout,
  ) {
    const width = Math.min(Math.max(requestedLayout.w, 1), BUSINESS_DASHBOARD_COLUMNS);
    const layout: BusinessDashboardLayout = {
      x: Math.min(Math.max(requestedLayout.x, 0), BUSINESS_DASHBOARD_COLUMNS - width),
      y: Math.max(requestedLayout.y, 0),
      w: width,
      h: Math.max(requestedLayout.h, 1),
    };
    const widget: BusinessDashboardWidget = {
      id: options.createID?.() ?? createWidgetID(),
      type: descriptor.type,
      title: descriptor.defaultTitle,
      layout,
      settings: structuredClone(descriptor.defaultSettings),
    } as BusinessDashboardWidget;
    replaceWidgets([...options.document.value.widgets, widget]);
    selectedWidgetId.value = widget.id;
    return widget.id;
  }

  function selectWidget(id: string | null) {
    selectedWidgetId.value =
      id && options.document.value.widgets.some((item) => item.id === id) ? id : null;
  }

  function updateWidget(id: string, patch: BusinessDashboardWidgetPatch) {
    replaceWidgets(
      options.document.value.widgets.map((widget) =>
        widget.id === id
          ? {
              ...widget,
              ...patch,
              ...(patch.settings ? { settings: structuredClone(patch.settings) } : {}),
            }
          : widget,
      ) as BusinessDashboardWidget[],
    );
  }

  /** 复制组件时重新计算空闲位置，避免新副本与原组件完全重叠。 */
  function duplicateWidget(id: string) {
    const source = options.document.value.widgets.find((widget) => widget.id === id);
    if (!source) return;
    const layout = findAvailableWidgetPosition(options.document.value.widgets, source.layout);
    const widget = structuredClone(source);
    widget.id = options.createID?.() ?? createWidgetID();
    widget.title = `${source.title || '未命名组件'} 副本`;
    widget.layout = layout;
    replaceWidgets([...options.document.value.widgets, widget]);
    selectedWidgetId.value = widget.id;
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

  /** 桌面画布参数属于业务文档的一部分，更新时保持不可变数据流以正确驱动 dirty 状态。 */
  function updateDesktopSettings(
    patch: Partial<BusinessDashboardDocument['settings']['desktop']>,
  ) {
    options.document.value = {
      ...options.document.value,
      settings: {
        ...options.document.value.settings,
        desktop: { ...options.document.value.settings.desktop, ...patch },
      },
    };
  }

  function addFormDataset(formCode: string, name: string) {
    const id = createDatasetID();
    const dataset: BusinessDashboardDataset = {
      id,
      name,
      source: { type: 'form', formCode },
      query: { version: 1, sorts: [], paging: { page: 1, pageSize: 20 }, projection: [] },
    };
    options.document.value = {
      ...options.document.value,
      datasets: [...options.document.value.datasets, dataset],
    };
    return id;
  }

  function updateDataset(id: string, patch: BusinessDashboardDatasetPatch) {
    options.document.value = {
      ...options.document.value,
      datasets: options.document.value.datasets.map((dataset) =>
        dataset.id === id
          ? {
              ...dataset,
              ...patch,
              ...(patch.query ? { query: structuredClone(patch.query) } : {}),
            }
          : dataset,
      ),
    };
  }

  function removeDataset(id: string) {
    options.document.value = {
      ...options.document.value,
      datasets: options.document.value.datasets.filter((dataset) => dataset.id !== id),
      widgets: options.document.value.widgets.map((widget) =>
        widget.datasetId === id ? { ...widget, datasetId: undefined } : widget,
      ) as BusinessDashboardWidget[],
    };
  }

  /**
   * 组件编辑页以隔离草稿同时编辑 Dataset 与组件；完成时在一次不可变替换中提交，
   * 避免画布观察到只写入其中一半的中间状态。
   */
  function upsertWidgetWithDataset(
    widget: BusinessDashboardWidget,
    dataset: BusinessDashboardDataset,
  ) {
    const hasWidget = options.document.value.widgets.some((item) => item.id === widget.id);
    const hasDataset = options.document.value.datasets.some((item) => item.id === dataset.id);
    options.document.value = {
      ...options.document.value,
      datasets: hasDataset
        ? options.document.value.datasets.map((item) =>
            item.id === dataset.id ? structuredClone(dataset) : item,
          )
        : [...options.document.value.datasets, structuredClone(dataset)],
      widgets: hasWidget
        ? options.document.value.widgets.map((item) =>
            item.id === widget.id ? structuredClone(widget) : item,
          )
        : [...options.document.value.widgets, structuredClone(widget)],
    };
    selectedWidgetId.value = widget.id;
  }

  function replaceWidgets(widgets: BusinessDashboardWidget[]) {
    options.document.value = { ...options.document.value, widgets };
  }

  return {
    selectedWidgetId,
    selectedWidget,
    addWidget,
    addWidgetAtLayout,
    selectWidget,
    updateWidget,
    duplicateWidget,
    updateLayout,
    replaceLayouts,
    removeWidget,
    updateDesktopSettings,
    addFormDataset,
    updateDataset,
    removeDataset,
    upsertWidgetWithDataset,
  };
}

function createDatasetID(): string {
  const value =
    globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `dataset_${value.replaceAll('-', '')}`;
}

export function findAvailableWidgetPosition(
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
