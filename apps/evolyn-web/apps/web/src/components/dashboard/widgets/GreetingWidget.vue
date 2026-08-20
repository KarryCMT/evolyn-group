<script setup lang="ts">
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'GreetingWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <DashboardWidgetFrame title="工作台">
    <div class="greeting-widget">
      <el-avatar :size="36">李</el-avatar>
      <strong>李同学 下午好！</strong>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.greeting-widget {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 8px;
  color: var(--el-text-color-primary);
}
</style>
