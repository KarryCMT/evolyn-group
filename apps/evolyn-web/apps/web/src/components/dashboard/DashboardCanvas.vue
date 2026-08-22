<script setup lang="ts">
import { EvolynGrid } from '@evolyn.do/ui';
import { computed, markRaw } from 'vue';
import type { DashboardWidget } from '~/types/dashboard';
import DashboardWidgetHost from './DashboardWidgetHost.vue';

const props = defineProps<{
  widgets: DashboardWidget[];
}>();

const components = { DashboardWidgetHost: markRaw(DashboardWidgetHost) };
const gridOptions = computed(() => ({
  column: 12,
  cellHeight: 72,
  // 相邻卡片的上下边距各 8px，视觉间距为 16px。
  margin: '8px 12px',
  float: true,
}));
</script>

<template>
  <main class="dashboard-canvas">
    <EvolynGrid
      :model-value="widgets"
      :options="gridOptions"
      :components="components"
      :editable="false"
    />
  </main>
</template>

<style scoped lang="scss">
.dashboard-canvas {
  box-sizing: border-box;
  flex: 1;
  width: 100%;
  max-width: 1680px;
  min-height: 0;
  overflow: hidden;
  padding: 0px 40px 24px;
  margin: 0 auto;

  /* 工作台卡片使用独立的浮层规格，避免继承 Element Plus 过于克制的默认圆角和阴影。 */
  --el-border-radius-base: 10px;
  --el-border-color-lighter: rgba(31, 35, 41, 0.06);
  --el-box-shadow-lighter: 0 0 2px 0 rgba(19, 29, 46, 0.02), 0 1px 4px 0 rgba(19, 29, 46, 0.06);
}

.dashboard-canvas :deep(.evolyn-grid .grid-stack-item-content) {
  /* 网格内容盒不能裁掉卡片向外扩散的阴影；卡片自身已负责内部内容裁剪。 */
  overflow: visible !important;
}

@media (max-width: 768px) {
  .dashboard-canvas {
    padding-inline: 16px;
  }
}
</style>
