import { createRouter, createWebHistory } from 'vue-router';
import MobileLayout from '~/layouts/MobileLayout.vue';

declare module 'vue-router' {
  interface RouteMeta {
    title: string;
    tab?: 'home' | 'tasks' | 'me';
    showBack?: boolean;
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: MobileLayout,
      children: [
        {
          path: '',
          name: 'home',
          component: () => import('~/pages/HomePage.vue'),
          meta: { title: '移动工作台', tab: 'home' },
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('~/pages/PlaceholderPage.vue'),
          meta: { title: '我的待办', tab: 'tasks' },
          props: { title: '我的待办', description: '审批与业务待办将在后续迭代接入。' },
        },
        {
          path: 'me',
          name: 'me',
          component: () => import('~/pages/PlaceholderPage.vue'),
          meta: { title: '我的', tab: 'me' },
          props: { title: '个人中心', description: '账号、企业切换与安全设置将在后续迭代接入。' },
        },
        {
          path: 'runtime-preview',
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

router.afterEach((to) => {
  const productName = import.meta.env.VITE_GLOB_APP_TITLE || '灵衍云';
  document.title = to.meta.title ? `${to.meta.title} - ${productName}` : productName;
});

export default router;
