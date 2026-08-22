import type { EvolynGridItem } from '@evolyn.do/ui';

/** 当前包可读取、编辑和写回的最新 schema 版本。 */
export const DASHBOARD_SCHEMA_VERSION = 1 as const;
export type DashboardSchemaVersion = typeof DASHBOARD_SCHEMA_VERSION;

/** 可持久化的看板布局属性，不包含任何 Vue 组件或运行时 props。 */
export interface DashboardWidgetLayout {
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
}

/** 卡片业务内容；具体 type 的取值及配置由接入应用定义。 */
export interface DashboardWidgetContent<TType extends string = string> {
  id: string;
  type: TType;
  title: string;
  config?: Record<string, unknown>;
}

/** 看板持久化卡片定义。运行时组件映射由渲染器接收，不写入该模型。 */
export interface DashboardWidget<TType extends string = string>
  extends DashboardWidgetContent<TType>, DashboardWidgetLayout {
  presetKey?: string;
}

/** 后续接口持久化使用的稳定 JSON 根结构。 */
export interface DashboardSchema<TType extends string = string> {
  version: DashboardSchemaVersion;
  widgets: DashboardWidget<TType>[];
}

/**
 * 工作台持久化文档。当前 schema 已是稳定 JSON 根结构，保留该别名便于应用侧实现加载、保存适配器。
 */
export type DashboardDocument<TType extends string = string> = DashboardSchema<TType>;

/** 设计器组件面板的组件预设。 */
export interface DashboardWidgetPreset<TType extends string = string> {
  key: string;
  type: TType;
  title: string;
  w: number;
  h: number;
  minW: number;
  minH: number;
  maxW?: number;
  maxH?: number;
  config?: Record<string, unknown>;
}

/** 供 EvolynGrid 渲染的运行时适配项，component 和 props 不属于持久化 schema。 */
export type DashboardGridItem<TType extends string = string> = DashboardWidget<TType> &
  EvolynGridItem;
