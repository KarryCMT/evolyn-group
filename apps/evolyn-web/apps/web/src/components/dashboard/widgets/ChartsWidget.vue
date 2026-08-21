<script setup lang="ts">
import { ElEmpty } from 'element-plus';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'ChartsWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <DashboardWidgetFrame title="我的图表">
    <template #actions><el-button text type="primary">添加</el-button></template>
    <el-empty class="charts-widget" description="暂无图表" :image-size="48" />
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.charts-widget {
  height: 100%;
  padding: 0;
}
</style>
