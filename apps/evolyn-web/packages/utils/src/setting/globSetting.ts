import type { GlobConfig } from '../types/config';

/**
 * 全局接口配置（apiUrl / urlPrefix 等）。
 *
 * utils 是纯 TS 基础包，不感知构建期环境变量（import.meta.env 属应用层），
 * 因此采用「应用启动时注入」的方式：主应用入口（如读取 Vite 环境变量后）
 * 调用 setupGlobSetting 完成配置，其余模块经 useGlobSetting 只读消费。
 */

// 默认全空：未注入时请求走同源相对路径，由应用的代理/网关兜底
const defaultGlobConfig: GlobConfig = {
  title: '',
  apiUrl: '',
  reportApiUrl: '',
  shortName: '',
  webSocketUrl: '',
  cipherKey: '',
  aMapJsKey: '',
  aMapWebKey: '',
  aMapSecurityJsCode: '',
  filePreviewServer: '',
  dataVUrl: '',
  reportServer: '',
  report: '',
};

let globConfig: GlobConfig = defaultGlobConfig;

/** 读取全局接口配置（只读视图） */
export function useGlobSetting(): Readonly<GlobConfig> {
  return globConfig;
}

/** 注入/覆盖全局接口配置，应用启动时调用一次 */
export function setupGlobSetting(config: Partial<GlobConfig>): void {
  globConfig = { ...globConfig, ...config };
}
