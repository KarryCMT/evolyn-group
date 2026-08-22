<script setup lang="ts">
import DashboardCanvas from '~/components/dashboard/DashboardCanvas.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useDashboardWorkspace } from '~/composables/useDashboardWorkspace';

const { schema } = useDashboardWorkspace();
</script>

<template>
  <div class="dashboard-page">
    <TopNavigation />
    <DashboardCanvas :schema="schema" />
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
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-9: #e8f1ff;
}
</style>
