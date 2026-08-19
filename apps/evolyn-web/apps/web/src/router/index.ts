import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '~/composables'

// 手动维护的路由表：不使用 vite-ssg / unplugin-vue-router 的文件式自动路由，
// 路由路径与 src/pages/ 目录结构一一对应，新增页面时在此登记
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
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
  ],
})

// 路由元信息类型收窄：public 标记认证域公开页面
declare module 'vue-router' {
  interface RouteMeta {
    /** 公开页面：无需登录即可访问（登录/注册/找回密码） */
    public?: boolean
  }
}

// 全局守卫：未登录访问受保护页面时跳转登录页并携带回跳地址；
// 已登录再访问登录页则直接回首页，避免会话内重复登录
router.beforeEach((to) => {
  const { isAuthenticated } = useAuth()

  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.name === 'login' && isAuthenticated.value) {
    return { path: '/' }
  }
})

export default router
