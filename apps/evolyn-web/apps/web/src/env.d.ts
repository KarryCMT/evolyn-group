/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 后端 API 基础地址：缺省走同源 /api/v1（开发期由 Vite 代理转发到 evolyn-core） */
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent<object, object, any>
  export default component
}
