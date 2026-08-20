import { createPinia } from 'pinia';
import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import '~/styles/index.scss';
import '@evolyn.do/ui/style.css';
import 'element-plus/theme-chalk/src/message.scss';
import 'element-plus/theme-chalk/src/message-box.scss';
import 'element-plus/theme-chalk/src/overlay.scss'; // the modal class for message box

const app = createApp(App);

// pinia 需先于 router 装配：初始导航触发路由守卫时即会实例化 auth store
app.use(createPinia());
app.use(router);

app.mount('#app');
