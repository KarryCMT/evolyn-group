<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { ref } from 'vue';
import type { DashboardWidget, DashboardWidgetType } from '~/types/dashboard';
import WidgetPalette from './WidgetPalette.vue';
import WorkbenchDesignCanvas from './WorkbenchDesignCanvas.vue';

defineOptions({ name: 'WorkbenchEditorShell' });

defineProps<{ device: 'desktop' | 'mobile' }>();

const widgets = ref<DashboardWidget[]>(createInitialWidgets());

function addWidget(type: DashboardWidgetType) {
  const id = `${type}-${Date.now()}`;
  const titleMap: Record<DashboardWidgetType, string> = {
    onboarding: '富文本', greeting: '问候语', shortcut: '未命名快捷入口', todo: '流程中心', favorites: '我的收藏', apps: '我的应用', charts: '我的图表',
  };
  widgets.value.push({ id, type, title: titleMap[type], x: 0, y: 99, w: 4, h: 2, minW: 3, minH: 1, component: 'WorkbenchEditorWidgetHost' });
}

function notify(action: string) { ElMessage.success(`${action}功能已就绪`); }

function createInitialWidgets(): DashboardWidget[] {
  return [
    widget('greeting', '问候语', 0, 0, 3, 1),
    widget('favorites', '最近使用', 3, 0, 9, 2),
    widget('shortcut', '未命名快捷入口', 0, 2, 12, 2),
    widget('todo', '流程中心', 0, 4, 3, 4),
    widget('shortcut', '未命名快捷入口', 0, 8, 3, 2),
    widget('favorites', '我的收藏', 3, 4, 9, 2),
    widget('apps', '我的应用', 3, 6, 9, 3),
    widget('charts', '我的图表', 3, 9, 9, 2),
  ];
}

function widget(type: DashboardWidgetType, title: string, x: number, y: number, w: number, h: number): DashboardWidget {
  return { id: `${type}-${x}-${y}`, type, title, x, y, w, h, minW: 3, minH: 1, component: 'WorkbenchEditorWidgetHost' };
}
</script>

<template>
  <div class="workbench-editor-shell">
    <WidgetPalette @add="addWidget" />
    <WorkbenchDesignCanvas v-model="widgets" :device="device" />
  </div>
</template>

<style scoped lang="scss">
.workbench-editor-shell { display: flex; min-height: calc(100vh - 92px); overflow: hidden; }
</style>
