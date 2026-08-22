<script setup lang="ts">
import { DashboardDesignCanvas } from '@evolyn.do/dashboard';
import { markRaw } from 'vue';
import type { DashboardSchema, DashboardWidget } from '~/types/dashboard';
import WorkbenchEditorWidgetHost from './WorkbenchEditorWidgetHost.vue';

const props = defineProps<{
  modelValue: DashboardSchema;
  device: 'desktop' | 'mobile';
  disabledKeys: string[];
}>();
const emit = defineEmits<{
  'update:modelValue': [value: DashboardSchema];
  remove: [id: string];
}>();

const components = { WorkbenchEditorWidgetHost: markRaw(WorkbenchEditorWidgetHost) };
function getWidgetProps(widget: DashboardWidget) {
  return {
    widget,
    onRemove: () => emit('remove', widget.id),
  };
}
</script>

<template>
  <DashboardDesignCanvas
    :model-value="modelValue"
    :components="components"
    widget-component="WorkbenchEditorWidgetHost"
    :get-widget-props="getWidgetProps"
    :preview="device"
    :disabled-preset-keys="disabledKeys"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
