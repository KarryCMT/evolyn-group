import { describe, expect, it } from 'vitest';
import {
  defaultMenuIconKey,
  menuIconOptions,
  resolveMenuIcon,
  resolveMenuIconTone,
} from '../menuIcon';

describe('menu icon appearance', () => {
  it('按菜单及表单类型派生唯一展示色', () => {
    expect(resolveMenuIconTone('dashboard', null)).toBe('dashboard');
    expect(resolveMenuIconTone('form', 'standard')).toBe('form');
    expect(resolveMenuIconTone('form', 'workflow')).toBe('workflow');
    expect(resolveMenuIconTone('group', null)).toBe('group');
    expect(resolveMenuIconTone('page', null)).toBe('page');
  });

  it('为未配置图标的菜单提供与类型一致的回退图标', () => {
    expect(defaultMenuIconKey('dashboard', null)).toBe('dashboard');
    expect(defaultMenuIconKey('form', 'standard')).toBe('file-list');
    expect(defaultMenuIconKey('form', 'workflow')).toBe('route');
    expect(defaultMenuIconKey('group', null)).toBe('folder');
    expect(defaultMenuIconKey('page', null)).toBe('article');
  });

  it('选择器中的每个图标键都能被菜单展示层解析', () => {
    expect(menuIconOptions.length).toBeGreaterThanOrEqual(30);
    expect(new Set(menuIconOptions.map((option) => option.key)).size).toBe(menuIconOptions.length);
    for (const option of menuIconOptions) {
      expect(resolveMenuIcon('form', option.key)).toBe(option.icon);
    }
  });
});
