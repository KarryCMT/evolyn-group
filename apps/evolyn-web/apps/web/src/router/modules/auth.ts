import type { RouteRecordRaw } from 'vue-router';

/**
 * 注册登录路由集合
 */
const authRoutes: RouteRecordRaw[] = [
  {
    // 根路径无独立页面：注册/登录完成后的默认去向统一重定向到工作台
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/auth/login',
    name: 'login',
    component: () => import('~/pages/auth/login.vue'),
    meta: { public: true, title: '账号登录' },
  },
  {
    path: '/auth/register',
    name: 'register',
    component: () => import('~/pages/auth/register.vue'),
    meta: { public: true, title: '注册账号' },
  },
  {
    path: '/auth/forgot-password',
    name: 'forgotPassword',
    component: () => import('~/pages/auth/forgot-password.vue'),
    meta: { public: true, title: '找回密码' },
  },
];

export default authRoutes;
