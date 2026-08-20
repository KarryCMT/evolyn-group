<script setup lang="ts">
import { ElScrollbar } from 'element-plus';
import { EvolynGrid, type EvolynGridItem } from '@evolyn.do/ui';
import { computed, markRaw, nextTick, ref, watch } from 'vue';
import type { GridStack as GridStackInstance, GridStackNode } from 'gridstack';
import type { DashboardWidget, DashboardWidgetContent } from '~/types/dashboard';
import WorkbenchEditorWidgetHost from './WorkbenchEditorWidgetHost.vue';

const props = defineProps<{
  modelValue: DashboardWidget[];
  device: 'desktop' | 'mobile';
}>();
const emit = defineEmits<{
  'update:modelValue': [value: DashboardWidget[]];
}>();

const components = { WorkbenchEditorWidgetHost: markRaw(WorkbenchEditorWidgetHost) };
const scrollbar = ref<{ setScrollTop: (scrollTop: number) => void }>();
const grid = ref<{ getGrid: () => GridStackInstance | null }>();
const editorItems = computed<DashboardWidget[]>(() =>
  props.modelValue.map((item) => ({
    ...item,
    component: 'WorkbenchEditorWidgetHost',
    props: { widget: toWidgetContent(item) },
  })),
);
const options = computed(() => ({
  column: props.device === 'desktop' ? 12 : 1,
  cellHeight: props.device === 'desktop' ? 72 : 92,
  margin: 14,
  float: true,
  acceptWidgets: '.widget-palette__drag-source',
  draggable: { handle: '.dashboard-widget__drag-handle' },
}));
function updateLayout(items: EvolynGridItem[]) {
  const current = new Map(props.modelValue.map((item) => [item.id, item]));
  emit(
    'update:modelValue',
    items.flatMap((item) => {
      const source = current.get(item.id);
      return source
        ? [
            {
              ...source,
              ...item,
              component: 'WorkbenchEditorWidgetHost',
              props: { widget: toWidgetContent(source) },
            },
          ]
        : [];
    }),
  );
}

function toWidgetContent(widget: DashboardWidget): DashboardWidgetContent {
  return { id: widget.id, type: widget.type, title: widget.title, config: widget.config };
}

/** GridStack 释放后一次性读取引擎最终布局，保留其原生的碰撞避让结果。 */
function handleDropped(_previous: GridStackNode | undefined, current: GridStackNode) {
  const content = (current as GridStackNode & { props?: { widget?: DashboardWidgetContent } }).props
    ?.widget;
  const droppedId = current.id;
  if (!content || !droppedId) return;

  const existing = new Map(props.modelValue.map((item) => [item.id, item]));
  const nodes = grid.value?.getGrid()?.engine.nodes ?? [current];
  const next: DashboardWidget[] = [];
  for (const node of nodes) {
    const widget = existing.get(node.id ?? '');
    if (widget) {
      next.push({
        ...widget,
        x: node.x ?? widget.x,
        y: node.y ?? widget.y,
        w: node.w ?? widget.w,
        h: node.h ?? widget.h,
      });
      continue;
    }
    if (node.id !== droppedId) continue;

    next.push({
      id: droppedId,
      type: content.type,
      title: content.title,
      config: content.config,
      x: current.x ?? 0,
      y: current.y ?? 0,
      w: current.w ?? 1,
      h: current.h ?? 1,
      minW: current.minW,
      minH: current.minH,
      component: 'WorkbenchEditorWidgetHost',
    });
  }

  emit('update:modelValue', next);
}

watch(
  () => props.modelValue.length,
  async (length, previousLength) => {
    if (length <= previousLength) return;

    const added = props.modelValue.at(-1);
    if (!added) return;

    await nextTick();
    const cellHeight = props.device === 'desktop' ? 72 : 92;
    scrollbar.value?.setScrollTop(Math.max(0, added.y * (cellHeight + 14) - 14));
  },
);
</script>

<template>
  <section class="workbench-design-canvas">
    <el-scrollbar ref="scrollbar" class="workbench-design-canvas__scrollbar" always>
      <div
        class="workbench-design-canvas__surface"
        :class="`workbench-design-canvas__surface--${device}`"
      >
        <EvolynGrid
          ref="grid"
          :model-value="editorItems"
          :options="options"
          :components="components"
          editable
          @dropped="handleDropped"
          @update:model-value="updateLayout"
        />
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

  &__scrollbar {
    height: 100%;
  }
  &__surface {
    box-sizing: border-box;
    min-height: 100%;
    padding: 12px;
  }
  &__surface--desktop {
    min-width: 0;
  }
  &__surface--mobile {
    min-width: 420px;
    max-width: 480px;
    margin: 0 auto;
  }

  /* 拖拽把手只在设计画布显示，成员端保持静态、干净的卡片外观。 */
  :deep(.dashboard-widget__drag-handle) {
    width: var(--el-component-size-small);
  }
}
</style>
