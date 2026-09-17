import type { Component } from 'vue';
import type { AppMenuCapabilities, FormType } from '~/types';

/** 应用工作区顶部的上下文模式；具体内容由后续表单包接管。 */
export type AppWorkspaceMode = 'fill' | 'design' | 'data';

/**
 * 应用侧栏的资产节点（树形）：folder 为分组导航容器并携带 children，
 * 其余为可打开资产；由 useAppMenu 从后端菜单（rootMenuIds +
 * nodeMap）建树产出，组件不解析后端结构。code 为菜单节点编码 menuId。
 */
export interface AppWorkspaceAsset {
  code: string;
  label: string;
  icon: Component;
  /** 后端菜单节点的稳定图标键，供「修改名称和图标」回填当前选择。 */
  iconKey: string | null;
  type: 'form' | 'dashboard' | 'page' | 'folder';
  /** 资产公开编码；表单节点用于跳转设计器，分组节点为空。 */
  targetCode: string | null;
  /** 表单资产类型；非表单节点为空。 */
  formType: FormType | null;
  /** 当前成员的收藏状态（ADR-011 个人状态，随菜单快照透传；右键收藏/
   * 取消后由页面层以接口返回值本地覆写，重取菜单时回归服务端事实源） */
  favorited: boolean;
  /** 当前成员由菜单读取接口派生的操作能力，用于控制右侧更多入口。 */
  capabilities: AppMenuCapabilities;
  children?: AppWorkspaceAsset[];
}

/**
 * 侧栏更多浮窗的操作入口（ADR-011 动作码，与后端注册表一致）：
 * rename=修改名称（表单节点含图标/颜色）、switch-type=切换表单类型、
 * reference-view=查看引用视图、copy-in-app/copy-cross-app=复制（当前/其他
 * 应用）、hide=对成员隐藏；实际写入接口按后续里程碑接入。
 */
export type AppWorkspaceAssetAction =
  | 'edit'
  | 'rename'
  | 'switch-type'
  | 'reference-view'
  | 'copy-in-app'
  | 'copy-cross-app'
  | 'move'
  | 'favorite'
  | 'hide'
  | 'delete';

/** 创建菜单支持的资产类型；父级为空时创建根节点，实际持久化由页面层在对应 API 就绪后接入。 */
export type AppWorkspaceCreateAssetType = 'workflow-form' | 'form' | 'dashboard' | 'folder';
