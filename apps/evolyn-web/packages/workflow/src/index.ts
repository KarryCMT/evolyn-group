// 工作流是独立构建的共享包，宿主应用的 Element Plus 自动按需导入不会扫描包内 SFC。
// 在公共入口显式声明设计器实际使用的主题片，确保属性面板与弹层始终带有完整组件样式。
import 'element-plus/theme-chalk/src/base.scss';
import 'element-plus/theme-chalk/src/alert.scss';
import 'element-plus/theme-chalk/src/button.scss';
import 'element-plus/theme-chalk/src/checkbox.scss';
import 'element-plus/theme-chalk/src/collapse.scss';
import 'element-plus/theme-chalk/src/collapse-item.scss';
import 'element-plus/theme-chalk/src/dialog.scss';
import 'element-plus/theme-chalk/src/empty.scss';
import 'element-plus/theme-chalk/src/form.scss';
import 'element-plus/theme-chalk/src/form-item.scss';
import 'element-plus/theme-chalk/src/input.scss';
import 'element-plus/theme-chalk/src/input-number.scss';
import 'element-plus/theme-chalk/src/option.scss';
import 'element-plus/theme-chalk/src/option-group.scss';
import 'element-plus/theme-chalk/src/overlay.scss';
import 'element-plus/theme-chalk/src/popper.scss';
import 'element-plus/theme-chalk/src/popover.scss';
import 'element-plus/theme-chalk/src/radio.scss';
import 'element-plus/theme-chalk/src/radio-button.scss';
import 'element-plus/theme-chalk/src/radio-group.scss';
import 'element-plus/theme-chalk/src/scrollbar.scss';
import 'element-plus/theme-chalk/src/select.scss';
import 'element-plus/theme-chalk/src/switch.scss';
import 'element-plus/theme-chalk/src/tag.scss';
import 'element-plus/theme-chalk/src/text.scss';
import 'element-plus/theme-chalk/src/tooltip.scss';
import 'element-plus/theme-chalk/src/tree.scss';
import 'element-plus/theme-chalk/src/tree-select.scss';

export * from './adapters/graph';
export * from './schema';
export { default as WorkflowCanvas } from './designer/WorkflowCanvas.vue';
export { default as WorkflowDesigner } from './designer/WorkflowDesigner.vue';
export { default as WorkflowInspector } from './designer/WorkflowInspector.vue';
