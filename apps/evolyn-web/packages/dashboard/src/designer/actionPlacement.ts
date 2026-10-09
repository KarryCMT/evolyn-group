export type DashboardWidgetActionPlacement = 'inside' | 'outside';

/** 首行必须把操作条放进卡片，避免越过设计画布顶部；其余行优先浮在卡片外。 */
export function resolveDashboardWidgetActionPlacement(row: number): DashboardWidgetActionPlacement {
  return row <= 0 ? 'inside' : 'outside';
}
