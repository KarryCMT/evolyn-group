import { ref } from 'vue';
import { createDefaultWorkbenchWidgets } from '~/dashboard/defaultWorkbench';
import type { DashboardWidget } from '~/types/dashboard';

/**
 * 当前先使用前端默认布局；接入接口后仅替换加载逻辑，不影响成员端渲染边界。
 */
export function useDashboardWorkspace() {
  const widgets = ref<DashboardWidget[]>(toGridItems(createDefaultWorkbenchWidgets()));

  return { widgets };
}

/** 将持久化数据转换为动态组件输入，避免把整个布局对象递归塞进 props。 */
function toGridItems(items: DashboardWidget[]): DashboardWidget[] {
  return items.map((item) => ({
    ...item,
    props: {
      widget: {
        id: item.id,
        type: item.type,
        title: item.title,
        config: item.config,
      },
    },
  }));
}
