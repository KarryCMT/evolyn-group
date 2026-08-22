<script setup lang="ts">
import { ElScrollbar } from 'element-plus';
import {
  RiApps2Fill,
  RiBarChartBoxFill,
  RiArticleFill,
  RiDashboardFill,
  RiFileTextFill,
  RiSendPlaneFill,
  RiSlideshowFill,
  RiStarFill,
  RiTimerFill,
  RiUserFill,
} from '@remixicon/vue';
import { nextTick, onMounted } from 'vue';
import { GridStack as GridStackCore } from 'gridstack';
import type { Component } from 'vue';
import type { DashboardWidgetPreset } from '~/types/dashboard';

defineOptions({ name: 'WidgetPalette' });
const emit = defineEmits<{
  add: [preset: DashboardWidgetPreset];
}>();

type PaletteItem = DashboardWidgetPreset & { icon: Component; label: string };

const palette: PaletteItem[] = [
  {
    key: 'todo',
    label: '流程中心',
    title: '流程中心',
    type: 'todo',
    icon: RiSendPlaneFill,
    w: 3,
    h: 4,
    minW: 3,
    minH: 3,
  },
  {
    key: 'apps',
    label: '我的应用',
    title: '我的应用',
    type: 'apps',
    icon: RiApps2Fill,
    w: 9,
    h: 3,
    minW: 4,
    minH: 3,
  },
  {
    key: 'shortcut',
    label: '快捷入口',
    title: '未命名快捷入口',
    type: 'shortcut',
    icon: RiFileTextFill,
    w: 12,
    h: 2,
    minW: 3,
    minH: 2,
  },
  {
    key: 'charts',
    label: '图表看板',
    title: '图表看板',
    type: 'charts',
    icon: RiDashboardFill,
    w: 9,
    h: 2,
    minW: 4,
    minH: 2,
  },
  {
    key: 'rich-text',
    label: '富文本',
    title: '富文本',
    type: 'onboarding',
    icon: RiArticleFill,
    w: 12,
    h: 2,
    minW: 4,
    minH: 2,
    config: { variant: 'rich-text' },
  },
  {
    key: 'carousel',
    label: '轮播图',
    title: '轮播图',
    type: 'onboarding',
    icon: RiSlideshowFill,
    w: 12,
    h: 2,
    minW: 4,
    minH: 2,
    config: { variant: 'carousel' },
  },
  {
    key: 'recent',
    label: '最近使用',
    title: '最近使用',
    type: 'favorites',
    icon: RiTimerFill,
    w: 9,
    h: 2,
    minW: 4,
    minH: 2,
    config: { variant: 'recent' },
  },
  {
    key: 'favorites',
    label: '我的收藏',
    title: '我的收藏',
    type: 'favorites',
    icon: RiStarFill,
    w: 9,
    h: 2,
    minW: 4,
    minH: 2,
  },
  {
    key: 'my-charts',
    label: '我的图表',
    title: '我的图表',
    type: 'charts',
    icon: RiBarChartBoxFill,
    w: 9,
    h: 2,
    minW: 4,
    minH: 2,
    config: { variant: 'my-charts' },
  },
  {
    key: 'greeting',
    label: '问候语',
    title: '问候语',
    type: 'greeting',
    icon: RiUserFill,
    w: 3,
    h: 1,
    minW: 3,
    minH: 1,
  },
];

async function setupDragSources() {
  await nextTick();
  const presets = palette.map(({ icon: _icon, ...preset }) => preset);
  // 与官方 side-panel 示例一致：DOM 挂载后一次性注册来源元素及对应组件定义。
  const widgets = presets.map((preset) => ({
    id: `palette-${preset.key}`,
    x: 0,
    y: 0,
    w: preset.w,
    h: preset.h,
    minW: preset.minW,
    minH: preset.minH,
    component: 'WorkbenchEditorWidgetHost',
    props: {
      widget: {
        id: `palette-${preset.key}`,
        type: preset.type,
        title: preset.title,
        config: preset.config,
      },
    },
  }));
  GridStackCore.setupDragIn(
    '.widget-palette__drag-source',
    { appendTo: 'body', helper: 'clone' },
    widgets,
  );
}

onMounted(setupDragSources);
</script>

<template>
  <aside class="widget-palette">
    <strong class="widget-palette__title">页面组件</strong>
    <el-scrollbar class="widget-palette__scrollbar">
      <div class="widget-palette__list">
        <div
          v-for="item in palette"
          :key="item.key"
          class="widget-palette__item widget-palette__drag-source"
          :data-widget-key="item.key"
          role="button"
          tabindex="0"
          @click="emit('add', item)"
          @keydown.enter="emit('add', item)"
          @keydown.space.prevent="emit('add', item)"
        >
          <component :is="item.icon" class="widget-palette__icon" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </div>
      </div>
    </el-scrollbar>
  </aside>
</template>

<style scoped lang="scss">
.widget-palette {
  flex: 0 0 168px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 14px 12px;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);

  &__title {
    display: block;
    margin-bottom: 8px;
    font-size: var(--el-font-size-base);
  }
  &__scrollbar {
    flex: 1;
    min-height: 0;
  }
  &__list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  &__item {
    box-sizing: border-box;
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    height: var(--el-component-size);
    margin: 0;
    padding: 0 15px;
    cursor: grab;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      outline: none;
    }

    &:active {
      cursor: grabbing;
    }
  }
  &__icon {
    flex: none;
    width: 18px;
    height: 18px;
  }
}
</style>
