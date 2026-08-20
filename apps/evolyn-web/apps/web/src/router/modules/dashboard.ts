
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
]
