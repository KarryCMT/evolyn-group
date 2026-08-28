import type { Component } from 'vue';
import type { ApplicationMenuCapabilities, FormType } from '~/types';

/** 应用工作区顶部的上下文模式；具体内容由后续表单包接管。 */
export type ApplicationWorkspaceMode = 'fill' | 'design' | 'data';

/**
 * 应用侧栏的资产节点（树形）：folder 为分组导航容器并携带 children，
 * 其余为可打开资产；由 useApplicationMenu 从后端菜单（rootEntryIds +
 * entryMap）建树产出，组件不解析后端结构。code 为菜单节点编码 entryId。
 */
export interface ApplicationWorkspaceAsset {
  code: string;
  label: string;
  icon: Component;
  type: 'form' | 'dashboard' | 'page' | 'folder';
  /** 资产公开编码；表单节点用于跳转设计器，分组节点为空。 */
  targetCode: string | null;
  /** 表单资产类型；非表单节点为空。 */
  formType: FormType | null;
  /** 当前成员由菜单读取接口派生的操作能力，用于控制右侧更多入口。 */
  capabilities: ApplicationMenuCapabilities;
  children?: ApplicationWorkspaceAsset[];
}

/** 侧栏更多浮窗当前已具备的操作入口；实际写入接口按后续里程碑接入。 */
export type ApplicationWorkspaceAssetAction = 'edit' | 'move' | 'favorite' | 'delete';

/** 创建菜单支持的资产类型；父级为空时创建根节点，实际持久化由页面层在对应 API 就绪后接入。 */
export type ApplicationWorkspaceCreateAssetType = 'workflow-form' | 'form' | 'dashboard' | 'folder';
