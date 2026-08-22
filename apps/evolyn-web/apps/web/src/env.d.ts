/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 后端 API 基础地址：缺省走同源 /api/v1（开发期由 Vite 代理转发到 evolyn-core） */
  readonly VITE_API_BASE_URL?: string;
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
