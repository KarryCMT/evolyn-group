<script setup lang="ts">
import { ElScrollbar } from 'element-plus';
import { EvolynGrid, type EvolynGridItem } from '@evolyn.do/ui';
import { computed, markRaw } from 'vue';
import type { DashboardWidget, DashboardWidgetContent } from '~/types/dashboard';
import WorkbenchEditorWidgetHost from './WorkbenchEditorWidgetHost.vue';

const props = defineProps<{
  modelValue: DashboardWidget[];
  device: 'desktop' | 'mobile';
}>();
const emit = defineEmits<{ 'update:modelValue': [value: DashboardWidget[]] }>();

const components = { WorkbenchEditorWidgetHost: markRaw(WorkbenchEditorWidgetHost) };
const editorItems = computed<DashboardWidget[]>(() => props.modelValue.map(item => ({
  ...item,
  component: 'WorkbenchEditorWidgetHost',
  props: { widget: toWidgetContent(item) },
})));
const options = computed(() => ({
  column: props.device === 'desktop' ? 12 : 1,
  cellHeight: props.device === 'desktop' ? 72 : 92,
  margin: 14,
  float: true,
  draggable: { handle: '.dashboard-widget__drag-handle' },
}));

function updateLayout(items: EvolynGridItem[]) {
  const current = new Map(props.modelValue.map(item => [item.id, item]));
  emit('update:modelValue', items.flatMap(item => {
    const source = current.get(item.id);
    return source ? [{ ...source, ...item, component: 'WorkbenchEditorWidgetHost', props: { widget: toWidgetContent(source) } }] : [];
  }));
}

function toWidgetContent(widget: DashboardWidget): DashboardWidgetContent {
  return { id: widget.id, type: widget.type, title: widget.title, config: widget.config };
}
</script>

<template>
  <section class="workbench-design-canvas">
    <el-scrollbar class="workbench-design-canvas__scrollbar" always>
      <div class="workbench-design-canvas__surface" :class="`workbench-design-canvas__surface--${device}`">
        <EvolynGrid :model-value="editorItems" :options="options" :components="components" editable @update:model-value="updateLayout" />
      </div>
    </el-scrollbar>
  </section>
</template>

<style scoped lang="scss">
.workbench-design-canvas {
  box-sizing: border-box;
  flex: 1;
  width: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--el-fill-color-light);

  &__scrollbar { height: 100%; }
  &__surface { box-sizing: border-box; min-height: 100%; padding: 12px; }
  &__surface--desktop { min-width: 0; }
  &__surface--mobile { min-width: 420px; max-width: 480px; margin: 0 auto; }
}
</style>
