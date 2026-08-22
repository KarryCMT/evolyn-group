import type { GlobConfig } from '@evolyn.do/utils';
import { setupGlobSetting, useGlobSetting as readGlobSetting } from '@evolyn.do/utils';

/**
 * 全局接口配置的应用侧装配。
 *
 * utils 层不感知构建期环境变量（import.meta.env 属应用源码，库产物中不可靠），
 * 因此由应用启动时把 Vite 环境变量传入 initGlobSetting 完成映射注入：
 *
 *   // apps/web/src/main.ts
 *   import { initGlobSetting } from '@evolyn.do/hooks';
 *   initGlobSetting(import.meta.env);
 */

/** 约定的全局配置环境变量（VITE_GLOB_ 前缀，Vite 会注入 import.meta.env） */
export interface GlobSettingEnv {
  // 站点标题
  VITE_GLOB_APP_TITLE?: string;
  // 服务接口地址
  VITE_GLOB_API_URL?: string;
  // 报表服务接口地址
  VITE_GLOB_REPORT_API_URL?: string;
  // 服务接口 url 前缀
  VITE_GLOB_API_URL_PREFIX?: string;
  // 项目简称
  VITE_GLOB_APP_SHORT_NAME?: string;
  // WebSocket 地址
  VITE_GLOB_WEBSOCKET_URL?: string;
}

/**
 * 应用启动时调用：将 Vite 环境变量映射注入 utils 的全局配置。
 * env 由调用方传入而非库内直接读 import.meta.env，保证库产物与构建环境解耦。
 */
export function initGlobSetting(env: GlobSettingEnv): Readonly<GlobConfig> {
  setupGlobSetting({
    title: env.VITE_GLOB_APP_TITLE ?? '',
    // 未配置时走同源 /api/v1，开发期由 Vite 代理转发到 evolyn-core，生产由网关托管
    apiUrl: env.VITE_GLOB_API_URL || '/api/v1',
    reportApiUrl: env.VITE_GLOB_REPORT_API_URL ?? '',
    urlPrefix: env.VITE_GLOB_API_URL_PREFIX,
    shortName: env.VITE_GLOB_APP_SHORT_NAME ?? '',
    webSocketUrl: env.VITE_GLOB_WEBSOCKET_URL ?? '',
  });
  return readGlobSetting();
}

/** 读取全局接口配置（只读视图），等价于 utils 的 useGlobSetting，供应用侧统一从 hooks 引用 */
export function useGlobSetting(): Readonly<GlobConfig> {
  return readGlobSetting();
}
