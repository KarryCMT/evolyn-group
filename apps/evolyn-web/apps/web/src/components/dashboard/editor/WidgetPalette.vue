<script setup lang="ts">
import { DashboardWidgetPalette } from '@evolyn.do/dashboard';
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
import type { Component } from 'vue';
import type { DashboardWidgetPreset } from '~/types/dashboard';

defineOptions({ name: 'WidgetPalette' });
const emit = defineEmits<{
  add: [preset: DashboardWidgetPreset];
}>();
defineProps<{ disabledKeys: string[] }>();

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
    maxH: 1,
  },
];
</script>

<template>
  <DashboardWidgetPalette
    :presets="palette"
    :disabled-preset-keys="disabledKeys"
    @add="emit('add', $event)"
  >
    <template #item="{ preset }">
      <component :is="preset.icon" class="widget-palette__icon" aria-hidden="true" />
      <span>{{ preset.label }}</span>
    </template>
  </DashboardWidgetPalette>
</template>

<style scoped lang="scss">
.widget-palette__icon {
  flex: none;
  width: 18px;
  height: 18px;
}
</style>
