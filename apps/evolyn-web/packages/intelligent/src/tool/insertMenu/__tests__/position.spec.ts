import { describe, expect, it } from 'vitest';
import { clampInsertMenuPosition } from '../position';

const bounds = {
  viewportWidth: 1_200,
  viewportHeight: 800,
  menuWidth: 420,
  menuHeight: 300,
};

describe('clampInsertMenuPosition', () => {
  it('保留可完整展示的锚点位置', () => {
    expect(clampInsertMenuPosition({ x: 300, y: 240 }, bounds)).toEqual({ x: 320, y: 212 });
  });

  it('避免菜单超出画布右侧和底部', () => {
    expect(clampInsertMenuPosition({ x: 1_150, y: 780 }, bounds)).toEqual({
      x: 768,
      y: 488,
    });
  });

  it('避免菜单超出画布左侧和顶部', () => {
    expect(clampInsertMenuPosition({ x: -100, y: -100 }, bounds)).toEqual({ x: 12, y: 12 });
  });

  it('画布小于菜单时仍固定在安全边距内', () => {
    expect(
      clampInsertMenuPosition(
        { x: 200, y: 200 },
        { viewportWidth: 300, viewportHeight: 200, menuWidth: 420, menuHeight: 300 },
      ),
    ).toEqual({ x: 12, y: 12 });
  });
});
