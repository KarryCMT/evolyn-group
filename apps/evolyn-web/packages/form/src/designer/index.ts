// 设计器入口：仅供管理端按需引入，允许依赖拖拽、属性面板等设计态能力；
// 最终用户填写入口必须使用 @evolyn.do/form/runtime，二者不得交叉引用内部状态。
// 设计器状态即目标保存协议文档（content.items），不再存在画布模型→另一份
// 发布 Schema 的转换路径（ADR-010）。

// 共享组件不能依赖消费应用的按需自动导入；在设计器入口统一输出其实际使用的 Element Plus 主题样式。
import 'element-plus/theme-chalk/src/button.scss';
import 'element-plus/theme-chalk/src/checkbox.scss';
import 'element-plus/theme-chalk/src/divider.scss';
import 'element-plus/theme-chalk/src/empty.scss';
import 'element-plus/theme-chalk/src/form.scss';
import 'element-plus/theme-chalk/src/icon.scss';
import 'element-plus/theme-chalk/src/input-number.scss';
import 'element-plus/theme-chalk/src/input.scss';
import 'element-plus/theme-chalk/src/popconfirm.scss';
import 'element-plus/theme-chalk/src/popover.scss';
import 'element-plus/theme-chalk/src/radio-button.scss';
import 'element-plus/theme-chalk/src/scrollbar.scss';
import 'element-plus/theme-chalk/src/select.scss';
import 'element-plus/theme-chalk/src/segmented.scss';
import 'element-plus/theme-chalk/src/switch.scss';
import 'element-plus/theme-chalk/src/tabs.scss';
import 'element-plus/theme-chalk/src/tooltip.scss';

export { default as FormSchemaPalette } from './FormSchemaPalette.vue';
export type { FormSchemaPaletteGroup } from './FormSchemaPalette.vue';
export { default as FormSchemaCanvas } from './FormSchemaLayoutCanvas.vue';
export { default as FormSchemaItemPreview } from './FormSchemaItemPreview.vue';
export { default as FormSchemaPropertyPanel } from './FormSchemaPropertyPanel.vue';
export { createEmptyFormSchemaDocument, useFormSchemaEditor } from './useFormSchemaEditor';
export { FORM_SCHEMA_DRAG_GROUP } from './palette';
export type { FormSchemaPaletteEntry, FormSchemaPaletteDrag } from './palette';
export * from '../schema';
