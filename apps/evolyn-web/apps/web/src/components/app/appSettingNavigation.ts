import {
  RiApps2Fill,
  RiBarChartBoxFill,
  RiBrainFill,
  RiCalculatorFill,
  RiDatabase2Fill,
  RiFileChartFill,
  RiGroupFill,
  RiSendPlaneFill,
  RiSettings3Fill,
  RiShareForwardFill,
} from '@remixicon/vue';
import type { Component } from 'vue';

/** 应用后台侧栏中一个功能入口；路由名称避免与动态 appCode 路径拼接耦合。 */
export interface AppSettingNavigationItem {
  key: string;
  label: string;
  routeName: string;
  icon: Component;
}

/** 应用后台侧栏中的功能分组。 */
export interface AppSettingNavigationGroup {
  label: string;
  items: AppSettingNavigationItem[];
}

/**
 * 应用后台菜单与 app.ts 子路由一一对应。
 * 新增设置模块时需同时补齐路由和此处入口，确保页面可发现且高亮状态准确。
 */
export const appSettingNavigationGroups: AppSettingNavigationGroup[] = [
  {
    label: '应用设置',
    items: [
      {
        key: 'permissions',
        label: '表单/仪表盘权限',
        routeName: 'app-setting-permissions',
        icon: RiShareForwardFill,
      },
      {
        key: 'cross-app',
        label: '跨应用',
        routeName: 'app-setting-cross-app',
        icon: RiApps2Fill,
      },
      {
        key: 'basic',
        label: '应用设置',
        routeName: 'app-setting-basic',
        icon: RiSettings3Fill,
      },
      {
        key: 'management-groups',
        label: '应用管理组',
        routeName: 'app-setting-management-groups',
        icon: RiGroupFill,
      },
    ],
  },
  {
    label: '高级功能',
    items: [
      {
        key: 'aggregate-tables',
        label: '聚合表',
        routeName: 'app-setting-aggregate-tables',
        icon: RiDatabase2Fill,
      },
      {
        key: 'calculations',
        label: '计算',
        routeName: 'app-setting-calculations',
        icon: RiCalculatorFill,
      },
      {
        key: 'ai-assistant',
        label: '智能助手',
        routeName: 'app-setting-ai-assistant',
        icon: RiBrainFill,
      },
      {
        key: 'data-factory',
        label: '数据工厂',
        routeName: 'app-setting-data-factory',
        icon: RiDatabase2Fill,
      },
      {
        key: 'data-push',
        label: '数据推送',
        routeName: 'app-setting-data-push',
        icon: RiSendPlaneFill,
      },
      {
        key: 'process-analysis',
        label: '流程分析',
        routeName: 'app-setting-process-analysis',
        icon: RiFileChartFill,
      },
    ],
  },
];
