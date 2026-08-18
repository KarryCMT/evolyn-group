import { createRouter, createWebHistory } from 'vue-router'

// 手动维护的路由表：不使用 vite-ssg / unplugin-vue-router 的文件式自动路由，
// 路由路径与 src/pages/ 目录结构一一对应，新增页面时在此登记
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: () => import('~/pages/index.vue'),
    },
  ],
})

export default router
