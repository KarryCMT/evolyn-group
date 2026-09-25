import type { IntelligentPosition } from '../../schema';

const MENU_VIEWPORT_PADDING = 12;
const MENU_ANCHOR_OFFSET_X = 20;
const MENU_ANCHOR_OFFSET_Y = -28;

interface InsertMenuBounds {
  viewportWidth: number;
  viewportHeight: number;
  menuWidth: number;
  menuHeight: number;
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}

/**
 * 将节点菜单约束在画布可视区内；菜单尺寸变化（例如展开更多节点）后可复用同一计算。
 */
export function clampInsertMenuPosition(
  anchor: IntelligentPosition,
  bounds: InsertMenuBounds,
): IntelligentPosition {
  const maximumX = Math.max(
    MENU_VIEWPORT_PADDING,
    bounds.viewportWidth - bounds.menuWidth - MENU_VIEWPORT_PADDING,
  );
  const maximumY = Math.max(
    MENU_VIEWPORT_PADDING,
    bounds.viewportHeight - bounds.menuHeight - MENU_VIEWPORT_PADDING,
  );

  return {
    x: clamp(anchor.x + MENU_ANCHOR_OFFSET_X, MENU_VIEWPORT_PADDING, maximumX),
    y: clamp(anchor.y + MENU_ANCHOR_OFFSET_Y, MENU_VIEWPORT_PADDING, maximumY),
  };
}
