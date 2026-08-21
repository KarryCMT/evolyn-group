// // For this project development
// import 'vue';

/**
 * 用作给全局引入的UI组件类型提示：
 * tsconfig.json 需要添加配置："types": ["@evolyn.do/ui/global.d.ts"]
 *
 * 或者
 * 一个全局的类型声明文件.d.ts写入：/// <reference types="@evolyn.do/ui/global.d.ts" />
 * 类似于：/// <reference types="vite/client" /> 具体可参考playground下的env.d.ts
 */
declare module 'vue' {
  // 给 Volar 提供全局组件类型提示，避免全局注册后模板中丢失类型。
  export interface GlobalComponents {
    VButton: (typeof import('@evolyn.do/ui'))['VButton'];
    VDialog: (typeof import('@evolyn.do/ui'))['VDialog'];
    EvolynGrid: (typeof import('@evolyn.do/ui'))['EvolynGrid'];
    EvolynTable: (typeof import('@evolyn.do/ui'))['EvolynTable'];
  }
}

export {};
