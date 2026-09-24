// 智能助手是独立构建的共享包，宿主应用的自动按需导入不会扫描包内 SFC。
// 在公共入口显式声明设计器实际使用的主题片，保证包可被任意 Vue 3 宿主直接消费。
import 'element-plus/theme-chalk/src/base.scss';
import 'element-plus/theme-chalk/src/button.scss';
import 'element-plus/theme-chalk/src/popper.scss';
import 'element-plus/theme-chalk/src/tooltip.scss';
import '@logicflow/core/dist/index.css';
import './assets/base.css';
import './index.scss';

export * from './schema';
export * from './context/IntelligentContext';
export * from './context/IntelligentEventEmitter';
export * from './context/IntelligentHistory';
export * from './mock/nodeTemplates';
export { default as LogicPanel } from './components/LogicPanel.vue';
export { default as IntelligentCanvas } from './designer/IntelligentCanvas.vue';
export { default as IntelligentDesigner } from './designer/IntelligentDesigner.vue';
