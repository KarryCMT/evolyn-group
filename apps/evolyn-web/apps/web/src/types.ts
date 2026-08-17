import type { App } from 'vue'

// 模块安装函数：不依赖 vite-ssg 上下文，直接接收 Vue 应用实例
export type UserModule = (app: App) => void
