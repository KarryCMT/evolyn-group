import { createRouter, createWebHistory } from 'vue-router';
import { useAuth } from '~/composables';
import routes from './modules/index';
// 手动维护的路由表：不使用 vite-ssg / unplugin-vue-router 的文件式自动路由，
// 路由路径与 src/pages/ 目录结构一一对应，新增页面时在此登记
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

// 路由元信息类型收窄：public 标记认证域公开页面
declare module 'vue-router' {
  interface RouteMeta {
    /** 公开页面：无需登录即可访问（登录/注册/找回密码） */
    public?: boolean;
  }
}

// 全局守卫：未登录访问受保护页面时跳转登录页并携带回跳地址；
// 已登录再访问登录页则直接回首页，避免会话内重复登录
router.beforeEach((to) => {
  const { isAuthenticated, userInfo, loadUserInfo } = useAuth();

  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }

  if (to.name === 'login' && isAuthenticated.value) {
    return { path: '/' };
  }

  // 聚合信息是内存态，刷新页面即丢失：已登录访问受保护页时兜底重拉
  // （不阻塞导航，页面消费方对空值自行降级）
  if (!to.meta.public && isAuthenticated.value && !userInfo.value) {
    void loadUserInfo();
  }
});

export default router;
