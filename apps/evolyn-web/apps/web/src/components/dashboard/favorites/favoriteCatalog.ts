import type { Component } from 'vue';
import {
  RiApps2Fill,
  RiBarChartFill,
  RiFileCopyFill,
  RiFileList3Fill,
  RiFileTextFill,
  RiLineChartFill,
  RiPieChartFill,
  RiSettings3Fill,
  RiShoppingBagFill,
  RiShoppingCartFill,
  RiTicketFill,
  RiUserFill,
} from '@remixicon/vue';

export type FavoriteApplicationTone = 'blue' | 'cyan' | 'green' | 'orange' | 'purple' | 'red';

export interface FavoriteApplication {
  id: string;
  label: string;
  icon: Component;
  tone: FavoriteApplicationTone;
  children?: FavoriteApplication[];
}

/**
 * 应用目录临时由前端维护，后续接入应用中心接口时仅替换此数据源。
 * 每个节点既可作为应用收藏，也可展开浏览其下的业务页面。
 */
export const favoriteApplicationCatalog: FavoriteApplication[] = [
  {
    id: 'sample-app',
    label: '灵衍云示例应用',
    icon: RiApps2Fill,
    tone: 'green',
    children: [
      { id: 'order-management', label: '订单管理', icon: RiTicketFill, tone: 'orange' },
      { id: 'purchase-request', label: '采购申请', icon: RiShoppingCartFill, tone: 'orange' },
      { id: 'office-request', label: '办公用品申请', icon: RiShoppingBagFill, tone: 'orange' },
      { id: 'employee-file', label: '员工档案', icon: RiFileTextFill, tone: 'cyan' },
      { id: 'product-management', label: '产品管理', icon: RiSettings3Fill, tone: 'cyan' },
      { id: 'customer-info', label: '客户信息', icon: RiUserFill, tone: 'cyan' },
      { id: 'employee-analysis', label: '员工信息分析', icon: RiBarChartFill, tone: 'purple' },
      { id: 'order-analysis', label: '订单分析', icon: RiPieChartFill, tone: 'purple' },
      { id: 'customer-analysis', label: '客户信息分析', icon: RiLineChartFill, tone: 'purple' },
    ],
  },
  { id: 'contract-management', label: '合同管理', icon: RiFileCopyFill, tone: 'red' },
  { id: 'it-project-management', label: 'IT项目管理', icon: RiFileList3Fill, tone: 'blue' },
  { id: 'task-management', label: '任务管理', icon: RiFileList3Fill, tone: 'cyan' },
  { id: 'advanced-guide', label: '灵衍云高级功能介绍', icon: RiApps2Fill, tone: 'purple' },
];

/** 收藏顺序保留用户在面板中选择的顺序，便于在工作台呈现稳定的位置。 */
export const defaultFavoriteApplicationIds = [
  'contract-management',
  'advanced-guide',
  'it-project-management',
  'task-management',
  'sample-app',
];

/** 将树状目录拍平，供收藏列表和搜索结果按 id 快速定位。 */
export function flattenFavoriteApplications(
  applications = favoriteApplicationCatalog,
): FavoriteApplication[] {
  return applications.flatMap((application) => [
    application,
    ...flattenFavoriteApplications(application.children ?? []),
  ]);
}
