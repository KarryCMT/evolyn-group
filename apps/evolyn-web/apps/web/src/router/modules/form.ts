import type { RouteRecordRaw } from 'vue-router';

/**
 * 表单路由集合
 */
const formRoutes: RouteRecordRaw[] = [
  {
    path: '/app/:appCode/form/:formId',
    name: 'form',
    component: () => import('~/pages/form/index.vue'),
    meta: { public: false },
    // 表单设计是默认入口；其余子路由暂复用同一设计器骨架，待各业务视图落地后再挂载页面。
    redirect: { name: 'form-design' },
    children: [
      {
        path: 'design',
        name: 'form-design',
        component: () => import('~/pages/form/canvas-placeholder.vue'),
      },
      {
        path: 'workflow',
        name: 'form-workflow-design',
        component: () => import('~/pages/form/canvas-placeholder.vue'),
      },
      {
        path: 'extensions',
        name: 'form-extensions',
        component: () => import('~/pages/form/canvas-placeholder.vue'),
      },
      {
        path: 'publish',
        name: 'form-publish',
        component: () => import('~/pages/form/canvas-placeholder.vue'),
      },
      {
        path: 'data',
        name: 'form-data',
        component: () => import('~/pages/form/canvas-placeholder.vue'),
      },
    ],
  },
];
export default formRoutes;
