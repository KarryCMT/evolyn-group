import {
  RiLayoutGridFill,
  RiNotification3Fill,
  RiPlayCircleFill,
  RiSendPlaneFill,
  RiTaskFill,
} from '@remixicon/vue';
import { markRaw } from 'vue';

/** 应用运行态侧栏固定的个人入口编码；资产编码不应与其混用。 */
export type ApplicationPersonalNavigationCode =
  | 'todo'
  | 'started'
  | 'handled'
  | 'copied'
  | 'dashboard';

/**
 * 应用工作区侧栏的个人导航入口（我的待办/我发起的等）：不属于应用资产
 * 树（应用菜单接口不返回，见应用菜单接口功能设计方案 §2「homeList 不随
 * 菜单返回」），属于个人首页域，后续接入真实待办数据时替换。资产区
 * 数据源已切换为应用菜单接口（useApplicationMenu）。
 */
export const applicationPersonalNavigation = [
  { code: 'todo', label: '我的待办', icon: markRaw(RiNotification3Fill) },
  { code: 'started', label: '我发起的', icon: markRaw(RiPlayCircleFill) },
  { code: 'handled', label: '我处理的', icon: markRaw(RiTaskFill) },
  { code: 'copied', label: '抄送我的', icon: markRaw(RiSendPlaneFill) },
  { code: 'dashboard', label: '我的仪表盘', icon: markRaw(RiLayoutGridFill) },
] as const satisfies ReadonlyArray<{
  code: ApplicationPersonalNavigationCode;
  label: string;
  icon: ReturnType<typeof markRaw>;
}>;
