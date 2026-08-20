

/**
 * 注册登录路由集合
 */
export default [
    {
        path: '/',
        component: () => import('~/pages/index.vue'),
    },
    {
        path: '/auth/login',
        name: 'login',
        component: () => import('~/pages/auth/login.vue'),
        meta: { public: true },
    },
    {
        path: '/auth/register',
        name: 'register',
        component: () => import('~/pages/auth/register.vue'),
        meta: { public: true },
    },
    {
        path: '/auth/forgot-password',
        name: 'forgotPassword',
        component: () => import('~/pages/auth/forgot-password.vue'),
        meta: { public: true },
    },
]
