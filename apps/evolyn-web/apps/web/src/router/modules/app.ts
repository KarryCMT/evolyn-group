import type { RouteRecordRaw } from 'vue-router';

/**
 * 应用路由集合
 */
const appRoutes: RouteRecordRaw[] = [
    {
        path: '/app/:appCode',
        name: 'App',
        component: () => import('~/pages/app/index.vue'),
        meta: { public: false },
    },
];
export default appRoutes
