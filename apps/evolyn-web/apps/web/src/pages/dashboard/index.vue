<script setup lang="ts">
import DashboardCanvas from '~/components/dashboard/DashboardCanvas.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useDashboardWorkspace } from '~/composables/useDashboardWorkspace';

const { widgets } = useDashboardWorkspace();
</script>

<template>
  <div class="dashboard-page">
    <TopNavigation />
    <DashboardCanvas :widgets="widgets" />
  </div>
</template>

<style scoped lang="scss">
.dashboard-page {
  display: flex;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;

  /* 页面整体背景：设计稿统一的浅灰底（不在 Element 色板内，直接写字面值） */
  background: #f3f3f8;

  /* 成员端与设计器共享品牌主色，保证引导卡片和导航入口一致。 */
  --el-color-primary: #00b8a9;
  --el-color-primary-light-3: #4dcdc2;
  --el-color-primary-light-7: #b2e9e4;
  --el-color-primary-light-9: #e6f8f6;
}
</style>
