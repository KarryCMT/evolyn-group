<script setup lang="ts">
import { CollectionTag } from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'FavoritesWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
const apps = ['合同管理', '简道云高级功能介绍', 'IT项目管理', '任务管理'];
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template #actions><el-button text type="primary">添加</el-button></template>
    <el-empty v-if="widget.title === '最近使用'" class="favorites-widget__empty" description="无需统一配置，系统自动展示成员最近访问的表单/仪表盘" :image-size="0" />
    <div v-else class="favorites-widget">
      <el-button v-for="app in apps" :key="app" text :icon="CollectionTag">{{ app }}</el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.favorites-widget { display: flex; align-items: center; height: 100%; gap: 16px; }
.favorites-widget :deep(.el-button + .el-button) { margin-left: 0; }
.favorites-widget__empty { height: 100%; padding: 0; }
</style>
