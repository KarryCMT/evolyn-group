import { createPinia } from 'pinia';
import { createApp } from 'vue';
import { initGlobSetting, setupRequestMessage } from '@evolyn.do/hooks';
import { removeToken, setUnauthorizedHandler } from '@evolyn.do/utils';
import App from './App.vue';
import router from './router';
import '~/styles/index.scss';
import '@evolyn.do/ui/style.css';
// dashboard 包将 SFC 样式发布为独立入口，应用启动时加载以保证设计画布取得完整尺寸。
import '@evolyn.do/dashboard/style.css';
import '@evolyn.do/data/style.css';
import '@evolyn.do/form/style.css';
import '@evolyn.do/workflow/style.css';
import 'element-plus/theme-chalk/src/message.scss';
import 'element-plus/theme-chalk/src/message-box.scss';
import 'element-plus/theme-chalk/src/overlay.scss'; // the modal class for message box

// 注入共享请求层运行环境（须先于任何 http 请求）：
// - initGlobSetting：把 Vite 环境变量映射为全局接口配置（默认同源 /api/v1）；
// - setupRequestMessage：注册 Element Plus 错误提示实现，供 defHttp/checkStatus 消费；
// - setUnauthorizedHandler：会话过期（携带令牌收到 401）时清令牌并带 redirect 回登录页。
initGlobSetting(import.meta.env);
setupRequestMessage();
setUnauthorizedHandler(() => {
  removeToken();
  const redirect = encodeURIComponent(window.location.pathname + window.location.search);
  window.location.href = `/auth/login?redirect=${redirect}`;
});

const app = createApp(App);

// pinia 需先于 router 装配：初始导航触发路由守卫时即会实例化 auth store
app.use(createPinia());
app.use(router);

app.mount('#app');
