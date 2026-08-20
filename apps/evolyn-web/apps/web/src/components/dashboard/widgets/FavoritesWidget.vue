<script setup lang="ts">
import { CollectionTag } from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'FavoritesWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
const apps = ['合同管理', '简道云高级功能介绍', 'IT项目管理', '任务管理'];
</script>

<template>
  <DashboardWidgetFrame title="我的收藏">
    <template #actions><el-button text type="primary">添加</el-button></template>
    <div class="favorites-widget">
      <el-button v-for="app in apps" :key="app" text :icon="CollectionTag">{{ app }}</el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.favorites-widget { display: flex; align-items: center; height: 100%; gap: 16px; }
.favorites-widget :deep(.el-button + .el-button) { margin-left: 0; }
</style>
