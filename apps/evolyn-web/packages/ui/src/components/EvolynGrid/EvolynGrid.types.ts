import type { Component } from 'vue';
import type { GridStackNode } from 'gridstack';
import type { GridStackOptions } from 'gridstack/dist/vue';

/**
 * EvolynGrid 的可持久化布局项。data 由业务侧定义，网格只维护位置和尺寸。
 */
export interface EvolynGridItem {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
  maxW?: number;
  maxH?: number;
  noMove?: boolean;
  noResize?: boolean;
  component: string;
  props?: Record<string, unknown>;
  data?: unknown;
}

/** GridStack 组件名到 Vue 组件的映射，由使用方提供。 */
export type EvolynGridComponents = Record<string, Component>;

/** 不含 children 的通用网格配置，布局项由 v-model 统一提供。 */
export type EvolynGridOptions = Omit<GridStackOptions, 'children' | 'staticGrid'>;

export interface EvolynGridEmits {
  (event: 'update:modelValue', value: EvolynGridItem[]): void;
  (event: 'layout-change', value: EvolynGridItem[]): void;
  (event: 'dropped', previous: GridStackNode | undefined, current: GridStackNode): void;
  (event: 'ready'): void;
}
