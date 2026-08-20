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
  margin: 12,
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
}

@media (max-width: 768px) {
  .dashboard-canvas {
    padding-inline: 16px;
  }
}
</style>
