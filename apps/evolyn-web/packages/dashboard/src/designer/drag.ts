import { GridStack as GridStackCore } from 'gridstack';
import { type Component, nextTick } from 'vue';
import type { DashboardWidgetContent, DashboardWidgetPreset } from '../schema/types.js';

export interface SetupDashboardWidgetDragSourcesOptions<TType extends string> {
  selector?: string;
  presets: readonly DashboardWidgetPreset<TType>[];
  widgetComponent?: string;
  widgetRegistry?: Partial<Record<TType, Component>>;
  getWidgetProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
}

/**
 * 注册组件库到 GridStack 的外部拖放源。调用方只声明受控预设与渲染适配，
 * GridStack 节点结构和克隆策略由 dashboard 包统一维护。
 */
export async function setupDashboardWidgetDragSources<TType extends string>(
  options: SetupDashboardWidgetDragSourcesOptions<TType>,
) {
  await nextTick();
  const widgetComponent = options.widgetComponent ?? 'DashboardDesignWidgetHost';
  const widgets = options.presets.map((preset) => {
    const widget = toWidgetContent(preset);
    return {
      id: widget.id,
      x: 0,
      y: 0,
      w: preset.w,
      h: preset.h,
      minW: preset.minW,
      minH: preset.minH,
      maxW: preset.maxW,
      maxH: preset.maxH,
      component: widgetComponent,
      props: options.getWidgetProps?.(widget) ?? {
        widget,
        ...(options.widgetRegistry ? { widgetRegistry: options.widgetRegistry } : {}),
      },
    };
  });

  GridStackCore.setupDragIn(
    options.selector ?? '.dashboard-widget-palette__drag-source',
    { appendTo: 'body', helper: 'clone' },
    widgets,
  );
}

function toWidgetContent<TType extends string>(
  preset: DashboardWidgetPreset<TType>,
): DashboardWidgetContent<TType> {
  return {
    id: `palette-${preset.key}`,
    type: preset.type,
    title: preset.title,
    config: preset.config,
  };
}
