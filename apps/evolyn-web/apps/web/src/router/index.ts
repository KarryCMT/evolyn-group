import { createRouter, createWebHistory } from 'vue-router';
import { useAuth } from '~/composables';
import { useGlobSetting } from '@evolyn.do/hooks';
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
    /** 浏览器标签标题；未声明时回退至产品名 */
    title?: string;
  }
}

// 全局守卫：未登录访问受保护页面时跳转登录页并携带回跳地址；
// 已登录再访问登录页则直接回首页，避免会话内重复登录
router.beforeEach(async (to) => {
  const { isAuthenticated, userInfo, loadUserInfo } = useAuth();

  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }

  // 仅在已恢复聚合信息时才让已登录用户离开登录页。后端不可用时，
  // userInfo 拉取会失败但本地令牌仍在；若此处仅以 token 判断，会与下方
  // 「受保护页拉取失败后跳登录」形成 dashboard → login → dashboard 的请求循环。
  if (to.name === 'login' && isAuthenticated.value && userInfo.value) {
    return { path: '/' };
  }

  // 聚合信息是内存态，刷新页面即丢失：先向后端确认令牌仍对应有效设备会话。
  // 被挤下线的旧令牌不能只因仍存于浏览器而进入受保护界面。
  if (!to.meta.public && isAuthenticated.value && !userInfo.value) {
    const info = await loadUserInfo();
    if (!info) {
      return { name: 'login', query: { redirect: to.fullPath } };
    }
  }
});

// 仅在导航确认后更新标题，避免重定向或中断导航提前改变浏览器标签。
router.afterEach((to) => {
  // 未声明路由标题时使用环境配置的产品名称，保持与入口 HTML 一致。
  document.title = to.meta.title || useGlobSetting().title || '灵衍云';
});

export default router;
