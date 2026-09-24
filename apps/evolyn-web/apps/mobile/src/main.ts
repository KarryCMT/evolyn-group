import type { RequestMessageContent } from '@evolyn.do/utils';
import { setRequestMessage, setupGlobSetting } from '@evolyn.do/utils';
import { createPinia } from 'pinia';
import { showDialog, showFailToast } from 'vant';
import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
// Vant 作为宿主 UI 框架由应用显式加载，表单包只发布自身的 Core/Mobile 样式。
import 'vant/lib/index.css';
// 品牌变量必须位于 Vant 默认值之后，才能稳定覆盖组件库主题。
import '~/styles/index.scss';
import '@evolyn.do/form/runtime-mobile/style.css';

function requestMessageText(message: RequestMessageContent): string {
  return typeof message === 'string' ? message : message.content || '请求失败';
}

// 移动宿主直接注入 Vant 反馈组件，避免经过依赖 Element Plus 的桌面 hooks 包。
setRequestMessage({
  createMessage: {
    error: (message) => showFailToast(requestMessageText(message)),
  },
  createErrorModal: (options) => {
    void showDialog({
      title: options.title || '错误提示',
      message: options.content || '',
      confirmButtonText: '知道了',
    });
  },
});

setupGlobSetting({
  title: import.meta.env.VITE_GLOB_APP_TITLE || '灵衍云',
  apiUrl: import.meta.env.VITE_GLOB_API_URL || '/api/v1',
  reportApiUrl: '',
  shortName: 'lingyanyun-mobile',
  webSocketUrl: '',
});

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.mount('#app');
