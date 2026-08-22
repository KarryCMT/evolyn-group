/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 共享请求层（defHttp）的全局配置变量，缺省时 initGlobSetting 走同源 /api/v1 */
  readonly VITE_GLOB_API_URL?: string;
  readonly VITE_GLOB_API_URL_PREFIX?: string;
  readonly VITE_GLOB_APP_TITLE?: string;
  readonly VITE_GLOB_APP_SHORT_NAME?: string;
  readonly VITE_GLOB_REPORT_API_URL?: string;
  readonly VITE_GLOB_WEBSOCKET_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent<object, object, any>;
  export default component;
}
