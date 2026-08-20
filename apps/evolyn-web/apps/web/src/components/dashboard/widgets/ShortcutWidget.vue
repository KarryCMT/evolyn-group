<script setup lang="ts">
import { ElEmpty } from 'element-plus';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';

defineOptions({ name: 'ShortcutWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <DashboardWidgetFrame title="未命名快捷入口">
    <el-empty class="shortcut-widget" description="暂未设置快捷入口" :image-size="48" />
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.shortcut-widget { height: 100%; padding: 0; }
</style>
