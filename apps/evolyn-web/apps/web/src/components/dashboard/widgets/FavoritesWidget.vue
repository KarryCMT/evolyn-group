<script setup lang="ts">
import { CollectionTag, DataAnalysis, Document, Files } from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'FavoritesWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
const apps = [
  { label: '合同管理', icon: Files, tone: 'danger' },
  { label: '简道云高级功能介绍', icon: CollectionTag, tone: 'primary' },
  { label: 'IT项目管理', icon: Document, tone: 'info' },
  { label: '任务管理', icon: DataAnalysis, tone: 'success' },
];
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template #actions><el-button text type="primary">添加</el-button></template>
    <div v-if="widget.title === '最近使用'" class="favorites-widget favorites-widget--recent">
      <el-button text class="favorites-widget__recent-item" :icon="DataAnalysis"
        >合同统计看板</el-button
      >
    </div>
    <div v-else class="favorites-widget">
      <el-button v-for="app in apps" :key="app.label" text class="favorites-widget__item">
        <span class="favorites-widget__icon" :class="`favorites-widget__icon--${app.tone}`">
          <el-icon><component :is="app.icon" /></el-icon>
        </span>
        {{ app.label }}
      </el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.favorites-widget {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 20px;

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  &--recent {
    padding-left: 4px;
  }
  &__recent-item {
    margin: 0;
    color: var(--el-text-color-primary);
  }
  &__item {
    display: inline-flex;
    margin: 0;
    color: var(--el-text-color-primary);
  }
  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    margin-right: 8px;
    color: var(--el-color-white);
    border-radius: var(--el-border-radius-small);

    &--danger {
      background: var(--el-color-danger);
    }
    &--primary {
      background: var(--el-color-primary);
    }
    &--info {
      background: var(--el-color-info);
    }
    &--success {
      background: var(--el-color-success);
    }
  }
}
</style>
