<script setup lang="ts">
import type { DashboardWidgetContent } from '~/types/dashboard';

defineOptions({ name: 'GreetingWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <section class="greeting-widget">
    <el-avatar :size="36">李</el-avatar>
    <strong>李同学 下午好！</strong>
  </section>
</template>

<style scoped lang="scss">
.greeting-widget {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  height: 100%;
  padding: 16px;
  gap: 8px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-lighter);
}
</style>
