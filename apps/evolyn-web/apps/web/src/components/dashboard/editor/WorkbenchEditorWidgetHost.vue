<script setup lang="ts">
import { Delete, Setting } from '@element-plus/icons-vue';
import { ref } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import DashboardWidgetHost from '../DashboardWidgetHost.vue';

defineOptions({ name: 'WorkbenchEditorWidgetHost' });
const props = defineProps<{ widget: DashboardWidgetContent }>();
const emit = defineEmits<{ remove: [id: string] }>();
const selected = ref(false);
</script>

<template>
  <div class="workbench-editor-widget" :class="{ 'workbench-editor-widget--selected': selected }" @click.stop="selected = true">
    <DashboardWidgetHost :widget="widget" />
    <div class="workbench-editor-widget__actions">
      <el-button text circle :icon="Setting" aria-label="配置卡片" @click.stop="selected = true" />
      <el-button text circle :icon="Delete" aria-label="删除卡片" @click.stop="emit('remove', widget.id)" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.workbench-editor-widget {
  position: relative;
  width: 100%; height: 100%;
  border: 1px solid transparent;

  &:hover, &--selected { border-color: var(--el-color-primary); }
  &__actions {
    position: absolute; top: 4px; right: 4px; z-index: 2;
    display: none; padding: 0 2px; background: var(--el-bg-color); border-radius: var(--el-border-radius-small);
  }
  &:hover &__actions, &--selected &__actions { display: flex; }
}
</style>
