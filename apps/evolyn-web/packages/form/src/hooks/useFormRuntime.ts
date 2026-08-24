import { PluginRuntime } from '../types';

export type PluginRuntimeLanguage = 'javascript' | 'python' | 'java';

export const formRuntimeOptions = [PluginRuntime.NodeJS, PluginRuntime.Python3];

/**
 * 将接口旧运行时版本和页面旧展示文案统一归一成当前运行时枚举值。
 */
export const normalizeFormRuntime = (runtime?: string): PluginRuntime => {
  const value = runtime?.trim();
  if (!value) return PluginRuntime.Python3;
  const lowerValue = value.toLowerCase();
  if (lowerValue.startsWith('nodejs') || lowerValue.includes('node.js'))
    return PluginRuntime.NodeJS;
  if (lowerValue.startsWith('python')) return PluginRuntime.Python3;
  return PluginRuntime.Python3;
};

export const isFormNodeRuntime = (runtime?: string) =>
  normalizeFormRuntime(runtime) === PluginRuntime.NodeJS;

export const isFormPythonRuntime = (runtime?: string) =>
  normalizeFormRuntime(runtime) === PluginRuntime.Python3;

export const getFormRuntimeLanguage = (runtime?: string): PluginRuntimeLanguage => {
  if (runtime?.trim().toLowerCase().startsWith('java')) return 'java';
  if (isFormNodeRuntime(runtime)) return 'javascript';
  return 'python';
};

export const getFormRuntimeIcon = (runtime?: string) => {
  if (getFormRuntimeLanguage(runtime) === 'java') return 'J';
  if (isFormNodeRuntime(runtime)) return 'JS';
  if (isFormPythonRuntime(runtime)) return 'Py';
  return '{}';
};
