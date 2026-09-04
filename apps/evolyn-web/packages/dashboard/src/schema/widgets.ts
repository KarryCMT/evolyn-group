import type { EvolynGridItem } from '@evolyn.do/ui';
import type {
  DashboardGridItem,
  DashboardWidget,
  DashboardWidgetContent,
  DashboardWidgetPreset,
} from './types';

/** 从布局项提取业务卡片信息，防止把 GridStack 运行时字段透传给业务组件。 */
export function toDashboardWidgetContent<TType extends string>(
  widget: DashboardWidget<TType>,
): DashboardWidgetContent<TType> {
  return {
    id: widget.id,
    type: widget.type,
    title: widget.title,
    config: widget.config,
  };
}

/** 将持久化卡片转换为网格所需的运行时输入。 */
export function createDashboardGridItems<TType extends string>(
  widgets: DashboardWidget<TType>[],
  options: {
    component: string;
    createProps?: (widget: DashboardWidget<TType>) => Record<string, unknown>;
  },
): DashboardGridItem<TType>[] {
  return widgets.map((widget) => ({
    ...widget,
    component: options.component,
    props: options.createProps?.(widget) ?? { widget: toDashboardWidgetContent(widget) },
  }));
}

/** 应用网格引擎返回的新坐标时，仅保留可持久化的布局字段。 */
export function mergeDashboardWidgetLayout<TType extends string>(
  widget: DashboardWidget<TType>,
  layout: Partial<EvolynGridItem>,
): DashboardWidget<TType> {
  return {
    ...widget,
    x: layout.x ?? widget.x,
    y: layout.y ?? widget.y,
    w: layout.w ?? widget.w,
    h: layout.h ?? widget.h,
    minW: layout.minW ?? widget.minW,
    minH: layout.minH ?? widget.minH,
    maxW: layout.maxW ?? widget.maxW,
    maxH: layout.maxH ?? widget.maxH,
    noMove: layout.noMove ?? widget.noMove,
    noResize: layout.noResize ?? widget.noResize,
  };
}

/** 同类组件以业务类型和展示标题确定唯一性，避免配置差异造成重复添加。 */
export function isDashboardWidgetPresetInLayout<TType extends string>(
  preset: Pick<DashboardWidgetPreset<TType>, 'type' | 'title'>,
  widgets: DashboardWidget<TType>[],
) {
  return widgets.some((widget) => widget.type === preset.type && widget.title === preset.title);
}
