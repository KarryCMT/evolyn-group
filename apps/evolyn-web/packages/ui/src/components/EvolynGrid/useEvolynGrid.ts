import type { GridStackWidget } from 'gridstack/dist/vue';
import type { EvolynGridItem } from './EvolynGrid.types';

/** 将业务布局转换为 GridStack Vue 包装层所需的数据结构。 */
export function toGridStackWidgets(items: EvolynGridItem[]): GridStackWidget[] {
  return items.map(({ data: _data, ...item }) => item);
}

/**
 * 将 GridStack 变更合并回原业务数据，避免拖拽后丢失卡片配置。
 */
export function mergeGridLayout(
  source: EvolynGridItem[],
  changed: Array<Partial<EvolynGridItem> & { id?: string }>,
): EvolynGridItem[] {
  const changedById = new Map(changed.filter((item): item is typeof item & { id: string } => Boolean(item.id)).map(item => [item.id, item]));

  return source.map((item) => {
    const next = changedById.get(item.id);
    return next
      ? {
          ...item,
          x: next.x ?? item.x,
          y: next.y ?? item.y,
          w: next.w ?? item.w,
          h: next.h ?? item.h,
        }
      : item;
  });
}
