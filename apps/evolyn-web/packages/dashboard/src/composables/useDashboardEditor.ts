import {
  computed,
  ref,
  shallowRef,
  type ComputedRef,
  type Ref,
  type WritableComputedRef,
} from 'vue';
import {
  isDashboardWidgetPresetInLayout,
  type DashboardSchema,
  type DashboardWidget,
  type DashboardWidgetPreset,
} from '../schema';

interface WidgetPosition {
  x: number;
  y: number;
}

interface WidgetSize {
  w: number;
  h: number;
}

export interface UseDashboardEditorOptions<
  TWidget extends DashboardWidget<TType>,
  TType extends string,
> {
  initialSchema: DashboardSchema<TType>;
  isPresetRepeatable: (preset: Pick<DashboardWidgetPreset<TType>, 'key'>) => boolean;
  getWidgetSize: (preset: DashboardWidgetPreset<TType>) => WidgetSize;
  createWidget: (preset: DashboardWidgetPreset<TType>, position: WidgetPosition) => TWidget;
  getColumnCount?: () => number;
}

export interface DashboardEditor<TWidget extends DashboardWidget<string>> {
  schema: Ref<DashboardSchema<TWidget['type']>>;
  widgets: WritableComputedRef<TWidget[]>;
  selectedWidgetId: Readonly<Ref<string | null>>;
  selectedWidget: ComputedRef<TWidget | null>;
  disabledPresetKeys: ComputedRef<string[]>;
  addWidget: (preset: DashboardWidgetPreset<TWidget['type']>) => void;
  removeWidget: (id: string) => void;
  selectWidget: (id: string) => void;
  clearSelection: () => void;
  updateWidget: (id: string, patch: DashboardWidgetPatch<TWidget['type']>) => void;
}

/** 业务配置只允许修改持久化内容，布局字段仍由设计器网格引擎统一维护。 */
export type DashboardWidgetPatch<TType extends string> = Partial<
  Pick<DashboardWidget<TType>, 'title' | 'config'>
>;

/**
 * 编辑器只管理当前布局及其增删规则；默认布局、业务组件和保存接口均由应用侧提供。
 */
export function useDashboardEditor<TWidget extends DashboardWidget<TType>, TType extends string>(
  options: UseDashboardEditorOptions<TWidget, TType>,
): DashboardEditor<TWidget> {
  const schema = ref<DashboardSchema<TType>>(cloneSchema(options.initialSchema));
  const widgets = computed<TWidget[]>({
    get: () => schema.value.widgets as TWidget[],
    set: (value) => {
      schema.value = { ...schema.value, widgets: value };
    },
  });
  const selectedWidgetId = shallowRef<string | null>(null);
  const selectedWidget = computed<TWidget | null>(
    () => widgets.value.find((widget) => widget.id === selectedWidgetId.value) ?? null,
  );
  const disabledPresetKeys = computed(() =>
    widgets.value
      .filter(
        (widget): widget is TWidget & { presetKey: string } =>
          Boolean(widget.presetKey) && !options.isPresetRepeatable({ key: widget.presetKey! }),
      )
      .map((widget) => widget.presetKey),
  );

  function addWidget(preset: DashboardWidgetPreset<TType>) {
    if (
      !options.isPresetRepeatable(preset) &&
      isDashboardWidgetPresetInLayout(preset, widgets.value)
    ) {
      return;
    }

    const size = options.getWidgetSize(preset);
    const position = findAvailablePosition(widgets.value, size, options.getColumnCount?.() ?? 12);
    const widget = options.createWidget(preset, position);
    widgets.value = [...widgets.value, widget];
    selectedWidgetId.value = widget.id;
  }

  function removeWidget(id: string) {
    widgets.value = widgets.value.filter((widget) => widget.id !== id);
    if (selectedWidgetId.value === id) clearSelection();
  }

  function selectWidget(id: string) {
    selectedWidgetId.value = widgets.value.some((widget) => widget.id === id) ? id : null;
  }

  function clearSelection() {
    selectedWidgetId.value = null;
  }

  function updateWidget(id: string, patch: DashboardWidgetPatch<TType>) {
    widgets.value = widgets.value.map((widget) => {
      if (widget.id !== id) return widget;

      return {
        ...widget,
        ...patch,
        ...(Object.hasOwn(patch, 'config')
          ? { config: patch.config ? { ...patch.config } : undefined }
          : {}),
      } as TWidget;
    });
  }

  return {
    schema: schema as Ref<DashboardSchema<TWidget['type']>>,
    widgets,
    selectedWidgetId,
    selectedWidget,
    disabledPresetKeys,
    addWidget,
    removeWidget,
    selectWidget,
    clearSelection,
    updateWidget,
  };
}

/** 在当前列数内寻找首个没有与既有卡片重叠的位置。 */
function findAvailablePosition<TType extends string>(
  widgets: DashboardWidget<TType>[],
  size: WidgetSize,
  columns: number,
): WidgetPosition {
  const width = Math.min(Math.max(size.w, 1), columns);
  const lastRow = widgets.reduce((bottom, item) => Math.max(bottom, item.y + item.h), 0);
  const canPlace = (x: number, y: number) =>
    x >= 0 &&
    x + width <= columns &&
    !widgets.some(
      (item) =>
        x < item.x + item.w && x + width > item.x && y < item.y + item.h && y + size.h > item.y,
    );

  for (let y = 0; y <= lastRow; y += 1) {
    for (let x = 0; x <= columns - width; x += 1) {
      if (canPlace(x, y)) return { x, y };
    }
  }

  return { x: 0, y: lastRow };
}

/** 初始 schema 必须脱离默认常量，避免编辑操作污染调用方数据。 */
function cloneSchema<TType extends string>(schema: DashboardSchema<TType>): DashboardSchema<TType> {
  return {
    ...schema,
    widgets: schema.widgets.map((widget) => ({
      ...widget,
      config: widget.config ? { ...widget.config } : undefined,
    })),
  };
}
