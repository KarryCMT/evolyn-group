<script setup lang="ts">
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'ShortcutWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <DashboardWidgetFrame title="未命名快捷入口">
    <div class="shortcut-widget">暂未设置快捷入口</div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.shortcut-widget {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-small);
}
</style>
