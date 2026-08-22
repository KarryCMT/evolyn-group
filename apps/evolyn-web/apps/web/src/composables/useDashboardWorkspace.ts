import { ref } from 'vue';
import { normalizeDashboardSchema } from '@evolyn.do/dashboard';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import { isDashboardWidgetType, type DashboardSchema } from '~/types/dashboard';

/**
 * 当前先使用前端默认布局；接入接口后仅替换加载逻辑，不影响成员端渲染边界。
 */
export function useDashboardWorkspace() {
  const schema = ref<DashboardSchema>(resolveDashboardSchema(createDefaultWorkbenchSchema()));

  return { schema };
}

/** 接口接入后将服务端 JSON 传入此处；未知类型或损坏数据统一回退默认布局。 */
function resolveDashboardSchema(input: unknown): DashboardSchema {
  return (
    normalizeDashboardSchema(input, { isWidgetType: isDashboardWidgetType }) ??
    createDefaultWorkbenchSchema()
  );
}
