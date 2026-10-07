/**
 * 通用布局层：只描述网格、组件外壳与文档持久化，不包含业务仪表盘的
 * Dataset、发布范围或数据查询语义。企业工作台只能依赖本层导出。
 */
export { default as LayoutDesignCanvas } from '../designer/DashboardDesignCanvas.vue';
export { default as LayoutRenderer } from '../renderer/DashboardRenderer.vue';
export { default as LayoutWidgetFrame } from '../components/DashboardWidgetFrame.vue';
export { default as LayoutWidgetPalette } from '../designer/DashboardWidgetPalette.vue';
export { useDashboardEditor as useLayoutEditor } from '../composables/useDashboardEditor.js';
export { useDashboardPersistence as useLayoutPersistence } from '../composables/useDashboardPersistence.js';
export {
  normalizeDashboardSchema as normalizeLayoutDocument,
  validateDashboardSchema as validateLayoutDocument,
} from '../schema/lifecycle.js';
export type {
  DashboardEditor as LayoutEditor,
  DashboardWidgetPatch as LayoutWidgetPatch,
  UseDashboardEditorOptions as UseLayoutEditorOptions,
} from '../composables/useDashboardEditor.js';
export type {
  DashboardPersistence as LayoutPersistence,
  DashboardPersistenceAdapter as LayoutPersistenceAdapter,
  UseDashboardPersistenceOptions as UseLayoutPersistenceOptions,
} from '../composables/useDashboardPersistence.js';
export type {
  DashboardDocument as LayoutDocument,
  DashboardGridItem as LayoutGridItem,
  DashboardSchema as LayoutSchema,
  DashboardWidget as LayoutWidget,
  DashboardWidgetContent as LayoutWidgetContent,
  DashboardWidgetLayout as LayoutWidgetPosition,
  DashboardWidgetPreset as LayoutWidgetPreset,
} from '../schema/types.js';
export type {
  DashboardSchemaNormalizationOptions as LayoutNormalizationOptions,
  DashboardSchemaValidationIssue as LayoutValidationIssue,
  DashboardSchemaValidationResult as LayoutValidationResult,
} from '../schema/lifecycle.js';
export {
  createDashboardGridItems as createLayoutGridItems,
  isDashboardWidgetPresetInLayout as isLayoutPresetInDocument,
  mergeDashboardWidgetLayout as mergeLayoutWidgetPosition,
  toDashboardWidgetContent as toLayoutWidgetContent,
} from '../schema/widgets.js';
