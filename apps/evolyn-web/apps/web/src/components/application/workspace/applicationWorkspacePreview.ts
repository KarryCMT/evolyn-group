import {
  RiBarChartBoxFill,
  RiFileList3Fill,
  RiFolder3Fill,
  RiLayoutGridFill,
  RiNotification3Fill,
  RiPlayCircleFill,
  RiSendPlaneFill,
  RiTaskFill,
} from '@remixicon/vue';
import { markRaw } from 'vue';
import type { ApplicationWorkspaceAsset } from './applicationWorkspace.types';

/**
 * 表单/仪表盘接口尚未落地时，用于串联「新建表单 → 返回应用」的临时导航样例。
 * 该文件是唯一的过渡数据出口；接入应用运行时接口后替换其数据源即可。
 */
export const applicationWorkspacePreviewAssets: ApplicationWorkspaceAsset[] = [
  { code: 'project-profile', label: '项目档案', icon: markRaw(RiFileList3Fill), type: 'form' },
  {
    code: 'project-progress',
    label: '项目进度看板',
    icon: markRaw(RiBarChartBoxFill),
    type: 'dashboard',
  },
  { code: 'plan-management', label: '计划管理', icon: markRaw(RiFolder3Fill), type: 'folder' },
];

export const applicationPersonalNavigation = [
  { code: 'todo', label: '我的待办', icon: markRaw(RiNotification3Fill) },
  { code: 'started', label: '我发起的', icon: markRaw(RiPlayCircleFill) },
  { code: 'handled', label: '我处理的', icon: markRaw(RiTaskFill) },
  { code: 'copied', label: '抄送我的', icon: markRaw(RiSendPlaneFill) },
  { code: 'dashboard', label: '我的仪表盘', icon: markRaw(RiLayoutGridFill) },
];
