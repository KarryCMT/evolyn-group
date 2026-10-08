import { createRouter, createWebHistory } from 'vue-router';
import MobileLayout from '~/layouts/MobileLayout.vue';
import { useAuthStore } from '~/stores/auth';

declare module 'vue-router' {
  interface RouteMeta {
    title?: string;
    public?: boolean;
    showBack?: boolean;
  }
}

function loginRedirect(redirect: string) {
  return { path: '/auth/login', query: { redirect } };
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth/login',
      name: 'login',
      component: () => import('~/pages/LoginPage.vue'),
      meta: { title: '账号登录', public: true },
    },
    {
      path: '/',
      name: 'home',
      component: () => import('~/pages/HomePage.vue'),
      meta: { title: '工作台' },
    },
    {
      path: '/account/settings',
      name: 'account-settings',
      component: () => import('~/pages/AccountSettingsPage.vue'),
      meta: { title: '个人设置' },
    },
    {
      path: '/apps/:appCode',
      name: 'mobile-app',
      component: () => import('~/pages/AppPage.vue'),
      meta: { title: '应用' },
    },
    {
      path: '/apps/:appCode/forms/:formCode',
      name: 'mobile-form',
      component: () => import('~/pages/FormRuntimePage.vue'),
      meta: { title: '填写表单' },
    },
    {
      path: '/runtime-preview',
      component: MobileLayout,
      children: [
        {
          path: '',
          name: 'runtime-preview',
          component: () => import('~/pages/RuntimePreviewPage.vue'),
          meta: { title: '移动表单运行时', showBack: true },
        },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior: () => ({ top: 0 }),
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (!auth.isAuthenticated && (await auth.restoreSession())) {
    // Cookie 会话恢复成功后继续执行目标路由判定。
  }

  if (!to.meta.public && !auth.isAuthenticated) return loginRedirect(to.fullPath);
  if (to.name === 'login' && auth.isAuthenticated && auth.userInfo) return { path: '/' };
});

router.afterEach((to) => {
  const productName = import.meta.env.VITE_GLOB_APP_TITLE || '灵衍云';
  document.title = to.meta.title ? `${to.meta.title} - ${productName}` : productName;
});

export default router;
