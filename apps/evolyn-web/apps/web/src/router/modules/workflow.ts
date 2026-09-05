import type { RouteRecordRaw } from 'vue-router';

/** 审批中心属于租户级个人工作区，不绑定单个应用资产菜单。 */
const workflowRoutes: RouteRecordRaw[] = [
  {
    path: '/workflow',
    name: 'workflow-center',
    component: () => import('~/pages/workflow/index.vue'),
    meta: { title: '审批中心' },
  },
];

export default workflowRoutes;
