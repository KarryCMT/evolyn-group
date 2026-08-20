<script setup lang="ts">
import { Plus, Search } from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'AppsWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
const apps = ['简道云示例应用', '合同管理', 'IT项目管理', '任务管理'];
</script>

<template>
  <DashboardWidgetFrame title="我的应用">
    <template #actions>
      <div class="apps-widget__actions">
        <el-input placeholder="请输入名称搜索" :prefix-icon="Search" />
        <el-button type="primary" :icon="Plus">新建应用</el-button>
      </div>
    </template>
    <div class="apps-widget">
      <el-button v-for="app in apps" :key="app" class="apps-widget__item" text>{{ app }}</el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.apps-widget { display: flex; align-items: end; height: 100%; gap: 20px; }
.apps-widget__actions { display: flex; width: 320px; gap: 8px; }
.apps-widget__item { margin: 0; }
</style>
