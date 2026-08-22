<script setup lang="ts">
import { useDashboardEditor } from '@evolyn.do/dashboard';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import {
  isDashboardWidgetPresetRepeatable,
  type DashboardWidget,
  type DashboardWidgetPreset,
} from '~/types/dashboard';
import WidgetPalette from './WidgetPalette.vue';
import WorkbenchDesignCanvas from './WorkbenchDesignCanvas.vue';

defineOptions({ name: 'WorkbenchEditorShell' });

const props = defineProps<{ device: 'desktop' | 'mobile' }>();

const { schema, disabledPresetKeys, addWidget, removeWidget } = useDashboardEditor<
  DashboardWidget,
  DashboardWidget['type']
>({
  initialSchema: createDefaultWorkbenchSchema(),
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
</script>

<template>
  <div class="workbench-editor-shell">
    <WidgetPalette :disabled-keys="disabledPresetKeys" @add="addWidget" />
    <WorkbenchDesignCanvas
      v-model="schema"
      :device="device"
      :disabled-keys="disabledPresetKeys"
      @remove="removeWidget"
    />
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
