import { shallowRef } from 'vue';
import { normalizeDashboardSchema, type DashboardPersistenceAdapter } from '@evolyn.do/dashboard';
import { getWorkbench, saveWorkbench } from '~/api/workbench';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import {
  isDashboardWidgetType,
  type DashboardSchema,
  type DashboardWidgetType,
} from '~/types/dashboard';

/**
 * 工作台数据适配层：对接后端 /workbench（000078，企业级配置 + revision
 * 乐观锁）。工作台由企业管理员配置、全员共用：成员端首页读取渲染，设计
 * 页（仅企业管理员可达）读写保存。revision 口令随适配器闭包流转：load 时
 * 记录服务端版本，save 时携带并接受新值；保存冲突（409）原样抛出，由页面
 * 决定提示方式。共享 dashboard 包及设计器均不感知成员、租户或请求细节。
 */
let workbenchRevision = 0;

export const dashboardWorkspaceAdapter: DashboardPersistenceAdapter<DashboardWidgetType> = {
  async load() {
    const view = await getWorkbench();
    if (!view) return null; // 行缺失：持久化层回退默认布局
    workbenchRevision = view.revision;
    return view.document;
  },
  async save(document) {
    const view = await saveWorkbench({ revision: workbenchRevision, document });
    workbenchRevision = view.revision;
    return view.document;
  },
};

/**
 * 成员端读取企业工作台文档：先以本地默认布局立即渲染（租户开通即种子，
 * 读取通常命中），异步拉取成功后整档替换；行缺失或数据无效时保持默认布局。
 */
export function useDashboardWorkspace() {
  const schema = shallowRef<DashboardSchema>(createDefaultWorkbenchSchema());

  // 失败静默回落默认布局，不打断首页渲染（设计页保存时会再次尝试）
  dashboardWorkspaceAdapter
    .load()
    .then((input) => {
      schema.value = resolveDashboardSchema(input);
    })
    .catch(() => undefined);

  return { schema };
}

/** 服务端 JSON 或未知结构统一归一化；未知类型或损坏数据回退默认布局。 */
function resolveDashboardSchema(input: unknown): DashboardSchema {
  return (
    normalizeDashboardSchema(input, { isWidgetType: isDashboardWidgetType }) ??
    createDefaultWorkbenchSchema()
  );
}
