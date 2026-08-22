<script setup lang="ts">
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import { computed } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';

defineOptions({ name: 'ShortcutWidget' });
const props = defineProps<{ widget: DashboardWidgetContent }>();
const placeholder = computed(() => {
  const value = props.widget.config?.placeholder;
  return typeof value === 'string' && value.trim() ? value : '暂未设置快捷入口';
});
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <div class="shortcut-widget">{{ placeholder }}</div>
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
