/**
 * Web 端运行时：对外暴露终端语义，而非 Element Plus 实现细节。
 * 当前内部使用 Element Plus；替换 Web UI 库不影响 Schema 或宿主调用方。
 */
import 'element-plus/theme-chalk/src/button.scss';
import 'element-plus/theme-chalk/src/checkbox.scss';
import 'element-plus/theme-chalk/src/date-picker-panel.scss';
import 'element-plus/theme-chalk/src/divider.scss';
import 'element-plus/theme-chalk/src/dropdown.scss';
import 'element-plus/theme-chalk/src/input-number.scss';
import 'element-plus/theme-chalk/src/input.scss';
import 'element-plus/theme-chalk/src/option.scss';
import 'element-plus/theme-chalk/src/popper.scss';
import 'element-plus/theme-chalk/src/radio.scss';
import 'element-plus/theme-chalk/src/scrollbar.scss';
import 'element-plus/theme-chalk/src/select.scss';
import 'element-plus/theme-chalk/src/tag.scss';
import 'element-plus/theme-chalk/src/tabs.scss';
import 'element-plus/theme-chalk/src/time-picker.scss';

export * from '../runtime';
export { default as FormWebRuntimeSurface } from '../runtime/surface/FormRuntimeSurface.vue';
export { default as FormWebMultitabRenderer } from '../runtime/renderer/FormMultitabRenderer.vue';
export { createWebFieldRegistry } from './widgets/registry';
