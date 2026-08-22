<script setup lang="ts">
import { ElEmpty } from 'element-plus';
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import type { DashboardWidgetContent } from '~/types/dashboard';

defineOptions({ name: 'ChartsWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template v-if="!props.editorMode" #actions>
      <el-button text type="primary">添加</el-button>
    </template>
    <el-empty class="charts-widget" description="暂无图表" :image-size="48" />
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.charts-widget {
  height: 100%;
  padding: 0;
}
</style>
