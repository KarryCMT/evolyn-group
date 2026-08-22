<script setup lang="ts">
import { watch } from 'vue';
import { DashboardDesignCanvas, useDashboardEditor } from '@evolyn.do/dashboard';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import {
  type DashboardSchema,
  isDashboardWidgetPresetRepeatable,
  type DashboardWidgetContent,
  type DashboardWidget,
  type DashboardWidgetPreset,
} from '~/types/dashboard';
import { dashboardWidgetRegistry, getDashboardWidgetComponentProps } from '../widget-registry';
import WidgetInspector from './WidgetInspector.vue';
import WidgetPalette from './WidgetPalette.vue';

defineOptions({ name: 'WorkbenchEditorShell' });

const props = defineProps<{
  device: 'desktop' | 'mobile';
  modelValue: DashboardSchema;
}>();
const emit = defineEmits<{
  'update:modelValue': [value: DashboardSchema];
}>();

const {
  schema,
  selectedWidgetId,
  selectedWidget,
  disabledPresetKeys,
  addWidget,
  removeWidget,
  selectWidget,
  replaceSchema,
  updateWidget,
} = useDashboardEditor<DashboardWidget, DashboardWidget['type']>({
  initialSchema: props.modelValue ?? createDefaultWorkbenchSchema(),
  isPresetRepeatable: isDashboardWidgetPresetRepeatable,
  getWidgetSize: (preset) => ({
    w: props.device === 'mobile' ? 1 : preset.w,
    h: preset.h,
  }),
  getColumnCount: () => (props.device === 'mobile' ? 1 : 12),
  createWidget: (preset: DashboardWidgetPreset, position) => ({
    id: `${preset.key}-${Date.now()}`,
    type: preset.type,
    title: preset.title,
    x: position.x,
    y: position.y,
    w: props.device === 'mobile' ? 1 : preset.w,
    h: preset.h,
    minW: props.device === 'mobile' ? 1 : preset.minW,
    minH: preset.minH,
    maxW: preset.maxW,
    maxH: preset.maxH,
    config: preset.config,
    presetKey: preset.key,
  }),
});

/** 画布内部保留编辑草稿，父页面持有加载、保存后的唯一文档来源。 */
watch(
  () => props.modelValue,
  (value) => {
    if (value !== schema.value) replaceSchema(value);
  },
);
watch(schema, (value) => {
  if (value !== props.modelValue) emit('update:modelValue', value);
});

function getEditorWidgetProps(widget: DashboardWidgetContent) {
  return getDashboardWidgetComponentProps(widget, true);
}
</script>

<template>
  <div class="workbench-editor-shell">
    <WidgetPalette :disabled-keys="disabledPresetKeys" @add="addWidget" />
    <DashboardDesignCanvas
      v-model="schema"
      :widget-registry="dashboardWidgetRegistry"
      :get-component-props="getEditorWidgetProps"
      :preview="device"
      :disabled-preset-keys="disabledPresetKeys"
      :selected-widget-id="selectedWidgetId"
      @remove="removeWidget"
      @select="selectWidget"
    />
    <WidgetInspector :widget="selectedWidget" @update="updateWidget" />
  </div>
</template>

<style scoped lang="scss">
.workbench-editor-shell {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
