import type { RouteRecordRaw } from 'vue-router';

/**
 * 工作台路由集合
 */
const dashboardRoutes: RouteRecordRaw[] = [
  {
    path: '/app/:appCode/dashboard/:dashboardCode/design',
    name: 'dashboard-design',
    component: () => import('~/pages/dashboard/design.vue'),
    meta: { public: false, title: '仪表盘设计' },
  },
  {
    path: '/app/:appCode/dashboard/:dashboardCode/preview',
    name: 'dashboard-preview',
    component: () => import('~/pages/dashboard/preview.vue'),
    meta: { public: false, title: '仪表盘预览' },
  },
  {
    path: '/app/:appCode/dashboard/:dashboardCode/extensions',
    name: 'dashboard-extensions',
    component: () => import('~/pages/dashboard/extensions.vue'),
    meta: { public: false, title: '仪表盘扩展功能' },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('~/pages/dashboard/index.vue'),
    meta: { public: false, title: '工作台' },
  },
  {
    path: '/dashboard/custom_workbench',
    name: 'custom_workbench',
    component: () => import('~/pages/dashboard/custom_workbench.vue'),
    meta: { public: false, title: '工作台设置' },
  },
  {
    path: '/dashboard/account',
    name: 'account',
    component: () => import('~/pages/dashboard/account.vue'),
    meta: { public: false, title: '个人中心' },
  },
];
export default dashboardRoutes;
