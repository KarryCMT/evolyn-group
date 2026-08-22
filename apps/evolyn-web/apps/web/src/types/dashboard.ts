import {
  isDashboardWidgetPresetInLayout,
  type DashboardSchema as BaseDashboardSchema,
  type DashboardWidget as BaseDashboardWidget,
  type DashboardWidgetContent as BaseDashboardWidgetContent,
  type DashboardWidgetPreset as BaseDashboardWidgetPreset,
} from '@evolyn.do/dashboard';

export type DashboardWidgetType =
  | 'onboarding'
  | 'greeting'
  | 'shortcut'
  | 'todo'
  | 'favorites'
  | 'apps'
  | 'charts';

const dashboardWidgetTypes = new Set<DashboardWidgetType>([
  'onboarding',
  'greeting',
  'shortcut',
  'todo',
  'favorites',
  'apps',
  'charts',
]);

/** 共享包只校验通用结构，业务应用在此收窄可渲染的卡片类型。 */
export function isDashboardWidgetType(type: string): type is DashboardWidgetType {
  return dashboardWidgetTypes.has(type as DashboardWidgetType);
}

/** 应用侧收窄共享 schema 的 type，保持业务组件注册的穷尽性校验。 */
export type DashboardWidgetPreset = BaseDashboardWidgetPreset<DashboardWidgetType>;

/** 快捷入口与图表看板可按需添加多个实例，其余页面组件在同一工作台中只能保留一个。 */
const repeatableDashboardWidgetPresetKeys = new Set(['shortcut', 'charts']);

export function isDashboardWidgetPresetRepeatable(preset: Pick<DashboardWidgetPreset, 'key'>) {
  return repeatableDashboardWidgetPresetKeys.has(preset.key);
}

/** 卡片内容所需的最小业务数据，不包含网格坐标。 */
export type DashboardWidgetContent = BaseDashboardWidgetContent<DashboardWidgetType>;

/** 工作台持久化卡片：只保存布局与业务配置，不包含 Vue 组件和运行时 props。 */
export type DashboardWidget = BaseDashboardWidget<DashboardWidgetType>;

/** 工作台的稳定根结构，后续接口直接保存该 JSON。 */
export type DashboardSchema = BaseDashboardSchema<DashboardWidgetType>;

/** 同类组件以业务类型和展示标题确定唯一性，避免配置项差异导致重复添加。 */
export { isDashboardWidgetPresetInLayout };
