import type { Component } from 'vue';

/** 应用工作区顶部的上下文模式；具体内容由后续表单包接管。 */
export type ApplicationWorkspaceMode = 'fill' | 'design' | 'data';

/** 应用侧栏的资产节点：后续由应用运行时接口返回，不在组件内绑定表单实体。 */
export interface ApplicationWorkspaceAsset {
  code: string;
  label: string;
  icon: Component;
  type: 'form' | 'dashboard' | 'folder';
}
