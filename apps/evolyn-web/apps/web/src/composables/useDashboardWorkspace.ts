import { shallowRef } from 'vue';
import { normalizeDashboardSchema, type DashboardPersistenceAdapter } from '@evolyn.do/dashboard';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import {
  isDashboardWidgetType,
  type DashboardSchema,
  type DashboardWidgetType,
} from '~/types/dashboard';

const dashboardWorkspaceStorageKey = 'evolyn.dashboard.workspace';

/**
 * 工作台数据适配层。后端接口就绪后，只需替换这里的 localStorage 实现，
 * 共享 dashboard 包及设计器均不需要感知成员、租户或请求细节。
 */
export const dashboardWorkspaceAdapter: DashboardPersistenceAdapter<DashboardWidgetType> = {
  async load() {
    return readStoredDashboardSchema();
  },
  async save(document) {
    if (typeof window === 'undefined') return document;

    try {
      window.localStorage.setItem(dashboardWorkspaceStorageKey, JSON.stringify(document));
      return document;
    } catch {
      // 浏览器禁用存储或空间耗尽时，让页面决定如何向成员展示保存失败。
      throw new Error('Dashboard workspace persistence failed.');
    }
  },
};

/**
 * 成员端读取最近一次已保存的工作台文档；不存在或无效时回退默认布局。
 */
export function useDashboardWorkspace() {
  const schema = shallowRef<DashboardSchema>(resolveDashboardSchema(readStoredDashboardSchema()));

  return { schema };
}

function readStoredDashboardSchema() {
  if (typeof window === 'undefined') return null;

  try {
    const value = window.localStorage.getItem(dashboardWorkspaceStorageKey);
    return value ? (JSON.parse(value) as unknown) : null;
  } catch {
    return null;
  }
}

/** 接口接入后将服务端 JSON 传入此处；未知类型或损坏数据统一回退默认布局。 */
function resolveDashboardSchema(input: unknown): DashboardSchema {
  return (
    normalizeDashboardSchema(input, { isWidgetType: isDashboardWidgetType }) ??
    createDefaultWorkbenchSchema()
  );
}
