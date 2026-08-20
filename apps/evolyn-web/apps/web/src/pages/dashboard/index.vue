<script setup lang="ts">
import DashboardCanvas from '~/components/dashboard/DashboardCanvas.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useDashboardWorkspace } from '~/composables/useDashboardWorkspace';

const { isEditing, resetLayout, updateLayout, widgets } = useDashboardWorkspace();
</script>

<template>
  <div class="dashboard-page">
    <TopNavigation />
    <DashboardCanvas
      v-model:editable="isEditing"
      :widgets="widgets"
      @update:widgets="updateLayout"
      @reset="resetLayout"
    />
  </div>
</template>

<style scoped lang="scss">
.dashboard-page { min-height: 100vh; background: var(--el-fill-color-light); }
</style>
