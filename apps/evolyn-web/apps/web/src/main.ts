import { createPinia } from 'pinia';
import { createApp } from 'vue';
import { initGlobSetting, setupRequestMessage } from '@evolyn.do/hooks';
import { ERROR_CODES, setUnauthorizedHandler } from '@evolyn.do/utils';
import App from './App.vue';
import router from './router';
import { useAuthStore } from './stores/auth';
import '~/styles/index.scss';
import '@evolyn.do/ui/style.css';
// dashboard 包将 SFC 样式发布为独立入口，应用启动时加载以保证设计画布取得完整尺寸。
import '@evolyn.do/dashboard/style.css';
import '@evolyn.do/data-workspace/style.css';
import '@evolyn.do/form/style.css';
import '@evolyn.do/workflow/style.css';
import 'element-plus/theme-chalk/src/message.scss';
import 'element-plus/theme-chalk/src/message-box.scss';
import 'element-plus/theme-chalk/src/overlay.scss'; // the modal class for message box

const app = createApp(App);
const pinia = createPinia();
app.use(pinia);

// 注入共享请求层运行环境（须先于任何 http 请求）：
// - initGlobSetting：把 Vite 环境变量映射为全局接口配置（默认同源 /api/v1）；
// - setupRequestMessage：注册 Element Plus 错误提示实现，供 defHttp/checkStatus 消费；
// - setUnauthorizedHandler：会话过期（携带令牌收到 401）时清令牌并带 redirect 回登录页。
initGlobSetting(import.meta.env);
setupRequestMessage();
setUnauthorizedHandler((error) => {
  // 令牌存储和 Pinia 内存态必须同时清理，避免已失效会话继续渲染受保护页面。
  useAuthStore(pinia).clearSession();
  // 登录页自身的公开请求也可能返回 401；此时仅清理残留令牌，避免重复拼接登录回跳地址。
  if (window.location.pathname === '/auth/login') return;

  const redirect = encodeURIComponent(window.location.pathname + window.location.search);
  const reason =
    error.errCode === ERROR_CODES.AUTH_SESSION_REPLACED ? '&reason=session-replaced' : '';
  window.location.href = `/auth/login?redirect=${redirect}${reason}`;
});

// pinia 需先于 router 装配：初始导航触发路由守卫时即会实例化 auth store
app.use(router);

app.mount('#app');
