import { ref } from 'vue';
import type { DashboardWidget } from '~/types/dashboard';

const initialWidgets: DashboardWidget[] = [
  { id: 'onboarding', type: 'onboarding', title: '新手引导', x: 0, y: 0, w: 12, h: 2, noResize: true, component: 'DashboardWidgetHost' },
  { id: 'greeting', type: 'greeting', title: '问候', x: 0, y: 2, w: 3, h: 1, minW: 3, minH: 1, component: 'DashboardWidgetHost' },
  { id: 'shortcut', type: 'shortcut', title: '快捷入口', x: 0, y: 3, w: 12, h: 2, minH: 2, component: 'DashboardWidgetHost' },
  { id: 'todo', type: 'todo', title: '我的待办', x: 0, y: 5, w: 3, h: 4, minW: 3, minH: 3, component: 'DashboardWidgetHost' },
  { id: 'favorites', type: 'favorites', title: '我的收藏', x: 3, y: 5, w: 9, h: 2, minW: 4, minH: 2, component: 'DashboardWidgetHost' },
  { id: 'apps', type: 'apps', title: '我的应用', x: 3, y: 7, w: 9, h: 3, minW: 4, minH: 3, component: 'DashboardWidgetHost' },
  { id: 'charts', type: 'charts', title: '我的图表', x: 3, y: 10, w: 9, h: 2, minW: 4, minH: 2, component: 'DashboardWidgetHost' },
];

/**
 * 当前先使用前端默认布局；接入接口后仅替换 load/save，不影响页面和网格组件边界。
 */
export function useDashboardWorkspace() {
  const isEditing = ref(false);
  const widgets = ref<DashboardWidget[]>(toGridItems(initialWidgets));

  function updateLayout(layout: DashboardWidget[]) {
    widgets.value = toGridItems(layout);
  }

  function resetLayout() {
    updateLayout(initialWidgets);
  }

  return { isEditing, resetLayout, updateLayout, widgets };
}

/** 将持久化数据转换为动态组件输入，避免把整个布局对象递归塞进 props。 */
function toGridItems(items: DashboardWidget[]): DashboardWidget[] {
  return items.map(item => ({
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
