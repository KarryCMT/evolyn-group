<script setup lang="ts">
import { ElScrollbar } from 'element-plus';
import { CollectionTag, DataAnalysis, Document, Grid, Histogram, Promotion, Timer, UserFilled } from '@element-plus/icons-vue';
import type { Component } from 'vue';
import type { DashboardWidgetType } from '~/types/dashboard';

defineOptions({ name: 'WidgetPalette' });
const emit = defineEmits<{ add: [type: DashboardWidgetType] }>();

const palette: Array<{ label: string; type: DashboardWidgetType; icon: Component }> = [
  { label: '流程中心', type: 'todo', icon: Promotion },
  { label: '我的应用', type: 'apps', icon: Grid },
  { label: '快捷入口', type: 'shortcut', icon: Document },
  { label: '图表看板', type: 'charts', icon: Histogram },
  { label: '富文本', type: 'onboarding', icon: CollectionTag },
  { label: '轮播图', type: 'onboarding', icon: DataAnalysis },
  { label: '最近使用', type: 'favorites', icon: Timer },
  { label: '我的收藏', type: 'favorites', icon: CollectionTag },
  { label: '我的图表', type: 'charts', icon: Histogram },
  { label: '问候语', type: 'greeting', icon: UserFilled },
];
</script>

<template>
  <aside class="widget-palette">
    <strong class="widget-palette__title">页面组件</strong>
    <el-scrollbar class="widget-palette__scrollbar">
      <div class="widget-palette__list">
        <el-button v-for="item in palette" :key="item.label" class="widget-palette__item" text :icon="item.icon" @click="emit('add', item.type)">
          {{ item.label }}
        </el-button>
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

  &__title { display: block; margin-bottom: 8px; font-size: var(--el-font-size-base); }
  &__scrollbar { flex: 1; min-height: 0; }
  &__list { display: flex; flex-direction: column; gap: 6px; }
  &__item { justify-content: flex-start; width: 100%; margin: 0; color: var(--el-text-color-regular); background: var(--el-fill-color-light); }
}
</style>
