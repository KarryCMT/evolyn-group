<script setup lang="ts">
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import type { DashboardWidgetContent } from '~/types/dashboard';

defineOptions({ name: 'TodoWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
const entries = ['我发起的', '我处理的', '抄送我的', '发起流程', '待办委托'];
</script>

<template>
  <DashboardWidgetFrame title="我的待办">
    <div class="todo-widget">
      <el-button v-for="entry in entries" :key="entry" text>{{ entry }}</el-button>
    </div>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.todo-widget {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}
.todo-widget :deep(.el-button + .el-button) {
  margin-left: 0;
}
</style>
