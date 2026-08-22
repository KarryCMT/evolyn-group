import { createPinia } from 'pinia';
import { createApp } from 'vue';
import { initGlobSetting, setupRequestMessage } from '@evolyn.do/hooks';
import App from './App.vue';
import router from './router';
import '~/styles/index.scss';
import '@evolyn.do/ui/style.css';
import 'element-plus/theme-chalk/src/message.scss';
import 'element-plus/theme-chalk/src/message-box.scss';
import 'element-plus/theme-chalk/src/overlay.scss'; // the modal class for message box

// 注入共享请求层运行环境（须先于任何 defHttp 调用）：
// - initGlobSetting：把 Vite 环境变量映射为全局接口配置（默认同源 /api/v1）；
// - setupRequestMessage：注册 Element Plus 错误提示实现，供 defHttp/checkStatus 消费。
initGlobSetting(import.meta.env);
setupRequestMessage();

const app = createApp(App);

// pinia 需先于 router 装配：初始导航触发路由守卫时即会实例化 auth store
app.use(createPinia());
app.use(router);

app.mount('#app');
