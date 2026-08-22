<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { computed, ref } from 'vue';
import { createDefaultWorkbenchWidgets } from '~/dashboard/defaultWorkbench';
import {
  isDashboardWidgetPresetInLayout,
  isDashboardWidgetPresetRepeatable,
  type DashboardWidget,
  type DashboardWidgetPreset,
} from '~/types/dashboard';
import WidgetPalette from './WidgetPalette.vue';
import WorkbenchDesignCanvas from './WorkbenchDesignCanvas.vue';

defineOptions({ name: 'WorkbenchEditorShell' });

const props = defineProps<{ device: 'desktop' | 'mobile' }>();

const widgets = ref<DashboardWidget[]>(createDefaultWorkbenchWidgets());
const disabledPaletteKeys = computed(() =>
  widgets.value
    .filter(
      (widget): widget is DashboardWidget & { presetKey: string } =>
        Boolean(widget.presetKey) && !isDashboardWidgetPresetRepeatable({ key: widget.presetKey! }),
    )
    .map((widget) => widget.presetKey),
);

function addWidget(preset: DashboardWidgetPreset) {
  if (
    !isDashboardWidgetPresetRepeatable(preset) &&
    isDashboardWidgetPresetInLayout(preset, widgets.value)
  ) {
    return;
  }

  const id = `${preset.key}-${Date.now()}`;
  const width = props.device === 'mobile' ? 1 : preset.w;
  const height = preset.h;
  const position = findAvailablePosition(width, height);
  widgets.value.push({
    id,
    type: preset.type,
    title: preset.title,
    x: position.x,
    y: position.y,
    w: width,
    h: height,
    minW: props.device === 'mobile' ? 1 : preset.minW,
    minH: preset.minH,
    component: 'WorkbenchEditorWidgetHost',
    config: preset.config,
    presetKey: preset.key,
  });
}

/**
 * 点击组件时，在当前 12 列画布中寻找首个可用位置。
 * 拖入场景则由 GridStack 引擎负责实时碰撞与避让。
 */
function findAvailablePosition(width: number, height: number) {
  const column = 12;
  const lastRow = widgets.value.reduce((bottom, item) => Math.max(bottom, item.y + item.h), 0);
  const canPlace = (x: number, y: number) =>
    x >= 0 &&
    x + width <= column &&
    !widgets.value.some(
      (item) =>
        x < item.x + item.w && x + width > item.x && y < item.y + item.h && y + height > item.y,
    );

  for (let y = 0; y <= lastRow; y += 1) {
    for (let x = 0; x <= column - width; x += 1) {
      if (canPlace(x, y)) return { x, y };
    }
  }

  return { x: 0, y: lastRow };
}

function notify(action: string) {
  ElMessage.success(`${action}功能已就绪`);
}
</script>

<template>
  <div class="workbench-editor-shell">
    <WidgetPalette :disabled-keys="disabledPaletteKeys" @add="addWidget" />
    <WorkbenchDesignCanvas
      v-model="widgets"
      :device="device"
      :disabled-keys="disabledPaletteKeys"
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
