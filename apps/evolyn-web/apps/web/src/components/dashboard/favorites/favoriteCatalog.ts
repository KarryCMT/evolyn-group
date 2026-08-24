import type { Component } from 'vue';
import {
  Collection,
  Document,
  Files,
  Goods,
  Histogram,
  Management,
  Memo,
  PieChart,
  ShoppingCart,
  Tickets,
  TrendCharts,
  User,
} from '@element-plus/icons-vue';

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
    icon: Collection,
    tone: 'green',
    children: [
      { id: 'order-management', label: '订单管理', icon: Tickets, tone: 'orange' },
      { id: 'purchase-request', label: '采购申请', icon: ShoppingCart, tone: 'orange' },
      { id: 'office-request', label: '办公用品申请', icon: Goods, tone: 'orange' },
      { id: 'employee-file', label: '员工档案', icon: Document, tone: 'cyan' },
      { id: 'product-management', label: '产品管理', icon: Management, tone: 'cyan' },
      { id: 'customer-info', label: '客户信息', icon: User, tone: 'cyan' },
      { id: 'employee-analysis', label: '员工信息分析', icon: Histogram, tone: 'purple' },
      { id: 'order-analysis', label: '订单分析', icon: PieChart, tone: 'purple' },
      { id: 'customer-analysis', label: '客户信息分析', icon: TrendCharts, tone: 'purple' },
    ],
  },
  { id: 'contract-management', label: '合同管理', icon: Files, tone: 'red' },
  { id: 'it-project-management', label: 'IT项目管理', icon: Memo, tone: 'blue' },
  { id: 'task-management', label: '任务管理', icon: Memo, tone: 'cyan' },
  { id: 'advanced-guide', label: '灵衍云高级功能介绍', icon: Collection, tone: 'purple' },
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
