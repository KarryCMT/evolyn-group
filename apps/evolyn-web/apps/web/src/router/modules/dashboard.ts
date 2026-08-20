
/**
 * 工作台路由集合
 */
export default [
    {
        path: '/dashboard',
        name: 'dashboard',
        component: () => import('~/pages/dashboard/index.vue'),
        meta: { public: false },
    },
    {
        path: '/dashboard/custom_workbench',
        name: 'custom_workbench',
        component: () => import('~/pages/dashboard/custom_workbench.vue'),
        meta: { public: false },
    },
]
