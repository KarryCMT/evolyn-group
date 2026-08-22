<script setup lang="ts">
import {
  CollectionTag,
  DataAnalysis,
  Document,
  Files,
  Plus,
  Search,
} from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'AppsWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
const apps = [
  { label: '简道云示例应用', icon: CollectionTag, tone: 'success' },
  { label: '合同管理', icon: Files, tone: 'danger' },
  { label: 'IT项目管理', icon: Document, tone: 'primary' },
  { label: '任务管理', icon: DataAnalysis, tone: 'info' },
];
</script>

<template>
  <DashboardWidgetFrame title="我的应用">
    <template v-if="!props.editorMode" #actions>
      <div class="apps-widget__actions">
        <el-input placeholder="请输入名称搜索" :prefix-icon="Search" />
        <el-button type="primary" :icon="Plus">新建应用</el-button>
      </div>
    </template>
    <div class="apps-widget">
      <el-button v-for="app in apps" :key="app.label" class="apps-widget__item" text>
        <span class="apps-widget__icon" :class="`apps-widget__icon--${app.tone}`">
          <el-icon><component :is="app.icon" /></el-icon>
        </span>
        <span>{{ app.label }}</span>
      </el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.apps-widget {
  display: flex;
  align-items: flex-end;
  height: 100%;
  gap: 28px;

  &__actions {
    display: flex;
    width: 320px;
    gap: 8px;
  }
  &__item {
    display: inline-flex;
    flex-direction: column;
    height: auto;
    margin: 0;
    color: var(--el-text-color-primary);
    line-height: 1.5;
  }
  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    margin-bottom: 8px;
    color: var(--el-color-white);
    border-radius: var(--el-border-radius-base);
    font-size: 20px;

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
