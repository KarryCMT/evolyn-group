<script setup lang="ts" generic="TType extends string">
import { EvolynGrid, type EvolynGridItem, type EvolynGridOptions } from '@evolyn.do/ui';
import type { GridStack as GridStackInstance, GridStackNode } from 'gridstack';
import { computed, markRaw, type Component, useTemplateRef } from 'vue';
import {
  createDashboardGridItems,
  mergeDashboardWidgetLayout,
  toDashboardWidgetContent,
  type DashboardSchema,
  type DashboardWidget,
  type DashboardWidgetContent,
} from '../schema';
import DashboardDesignWidgetHost from './DashboardDesignWidgetHost.vue';

const props = withDefaults(
  defineProps<{
    modelValue: DashboardSchema<TType>;
    widgetRegistry: Partial<Record<TType, Component>>;
    getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
    selectedWidgetId?: string | null;
    preview?: 'desktop' | 'mobile';
    disabledPresetKeys?: string[];
    dragSourceSelector?: string;
  }>(),
  {
    preview: 'desktop',
    disabledPresetKeys: () => [],
    dragSourceSelector: '.dashboard-widget-palette__drag-source',
  },
);
const emit = defineEmits<{
  'update:modelValue': [value: DashboardSchema<TType>];
  remove: [id: string];
  select: [id: string];
}>();

const grid = useTemplateRef<{ getGrid: () => GridStackInstance | null }>('grid');
const components = { DashboardDesignWidgetHost: markRaw(DashboardDesignWidgetHost) };
const editorItems = computed(() =>
  createDashboardGridItems(props.modelValue.widgets, {
    component: 'DashboardDesignWidgetHost',
    createProps: getWidgetProps,
  }),
);
const gridOptions = computed<EvolynGridOptions>(() => ({
  column: props.preview === 'desktop' ? 12 : 1,
  cellHeight: props.preview === 'desktop' ? 72 : 92,
  // 相邻卡片的上下边距各 8px，视觉间距为 16px。
  margin: '8px 14px',
  float: true,
  acceptWidgets: (element: Element) =>
    element instanceof HTMLElement &&
    element.matches(props.dragSourceSelector) &&
    !props.disabledPresetKeys.includes(element.dataset.widgetKey ?? ''),
  draggable: { handle: '.dashboard-widget__drag-handle' },
  // 右侧拖宽，右下角对角手柄同时调整宽高；仅悬停时显示操作入口。
  resizable: { handles: 'e,se', autoHide: true },
}));

/** GridStack 返回运行时节点；回写前只保留 schema 允许持久化的布局字段。 */
function updateLayout(items: EvolynGridItem[]) {
  const current = new Map(props.modelValue.widgets.map((item) => [item.id, item]));
  emitSchema(
    items.flatMap((item) => {
      const source = current.get(item.id);
      return source ? [mergeDashboardWidgetLayout(source, item)] : [];
    }),
  );
}

function getWidgetProps(widget: DashboardWidget<TType>) {
  return {
    widget: toDashboardWidgetContent(widget),
    widgetRegistry: props.widgetRegistry,
    getComponentProps: props.getComponentProps,
    selected: widget.id === props.selectedWidgetId,
    onRemove: () => emit('remove', widget.id),
    onSelect: () => emit('select', widget.id),
  };
}

/** GridStack 为可重复拖入的同名节点追加序号，schema 中只记录原始预设键。 */
function getPresetKey(widgetID: string) {
  return widgetID.replace(/^palette-/, '').replace(/_\d+$/, '');
}

/** GridStack 释放后一次性读取引擎最终布局，保留其原生的碰撞避让结果。 */
function handleDropped(_previous: GridStackNode | undefined, current: GridStackNode) {
  const content = (
    current as GridStackNode & { props?: { widget?: DashboardWidgetContent<TType> } }
  ).props?.widget;
  const droppedID = current.id;
  if (!content || !droppedID) return;

  const existing = new Map(props.modelValue.widgets.map((item) => [item.id, item]));
  const nodes = grid.value?.getGrid()?.engine.nodes ?? [current];
  const widgets: DashboardWidget<TType>[] = [];
  for (const node of nodes) {
    const widget = existing.get(node.id ?? '');
    if (widget) {
      widgets.push(
        mergeDashboardWidgetLayout(widget, {
          x: node.x,
          y: node.y,
          w: node.w,
          h: node.h,
        }),
      );
      continue;
    }
    if (node.id !== droppedID) continue;

    widgets.push({
      ...content,
      id: droppedID,
      x: current.x ?? 0,
      y: current.y ?? 0,
      w: current.w ?? 1,
      h: current.h ?? 1,
      minW: current.minW,
      minH: current.minH,
      maxW: current.maxW,
      maxH: current.maxH,
      presetKey: getPresetKey(droppedID),
    });
  }

  emitSchema(widgets);
}

function emitSchema(widgets: DashboardWidget<TType>[]) {
  emit('update:modelValue', { ...props.modelValue, widgets });
}
</script>

<template>
  <section class="dashboard-design-canvas">
    <div class="dashboard-design-canvas__scroll">
      <div
        class="dashboard-design-canvas__surface"
        :class="`dashboard-design-canvas__surface--${preview}`"
      >
        <EvolynGrid
          ref="grid"
          :model-value="editorItems"
          :options="gridOptions"
          :components="components"
          editable
          @dropped="handleDropped"
          @update:model-value="updateLayout"
        />
      </div>
    </div>
  </section>
</template>

<style scoped lang="scss">
.dashboard-design-canvas {
  box-sizing: border-box;
  flex: 1;
  width: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--el-bg-color-page);

  &__scroll {
    width: 100%;
    height: 100%;
    overflow: auto;
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

  /* GridStack 的 east 手柄默认只有热区，补充双竖线作为可见的调宽提示。 */
  :deep(.grid-stack-item > .ui-resizable-e) {
    right: calc(var(--gs-item-margin-right) + 4px);
    width: 14px;

    &::before {
      position: absolute;
      top: 50%;
      left: 50%;
      width: 4px;
      height: 18px;
      content: '';
      border-right: 2px solid var(--el-text-color-secondary);
      border-left: 2px solid var(--el-text-color-secondary);
      border-radius: 1px;
      opacity: 0.55;
      transform: translate(-50%, -50%);
    }
  }

  /* 右下角以三角形提示双向调整，保留 GridStack 的宽高拖拽能力。 */
  :deep(.grid-stack-item > .ui-resizable-se) {
    right: calc(var(--gs-item-margin-right) + 2px);
    bottom: calc(var(--gs-item-margin-bottom) + 2px);
    width: 24px;
    height: 24px;
    background: none;
    transform: none;

    &::before {
      position: absolute;
      right: 4px;
      bottom: 4px;
      width: 12px;
      height: 12px;
      content: '';
      background: var(--el-text-color-secondary);
      clip-path: polygon(100% 0, 100% 100%, 0 100%);
      opacity: 0.5;
    }
  }

  /* 单行问候语只允许横向调整，避免显示无效的右下角双向手柄。 */
  :deep(.grid-stack-item[gs-max-h='1'] > .ui-resizable-se) {
    display: none;
  }
}
</style>
