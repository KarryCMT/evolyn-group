<script setup lang="ts" generic="TType extends string">
import { EvolynGrid, type EvolynGridItem, type EvolynGridOptions } from '@evolyn.do/ui';
import type { GridStack as GridStackInstance, GridStackNode } from 'gridstack';
import { type Component, computed, markRaw, useTemplateRef } from 'vue';
import {
  type DashboardSchema,
  type DashboardWidget,
  type DashboardWidgetContent,
  createDashboardGridItems,
  mergeDashboardWidgetLayout,
  toDashboardWidgetContent,
} from '../schema';
import { resolveDashboardWidgetActionPlacement } from './actionPlacement';
import DashboardDesignWidgetHost from './DashboardDesignWidgetHost.vue';

const props = withDefaults(
  defineProps<{
    modelValue: DashboardSchema<TType>;
    widgetRegistry: Partial<Record<TType, Component>>;
    getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
    selectedWidgetId?: string | null;
    preview?: 'desktop' | 'mobile';
    interactionMode?: 'move' | 'resize';
    rowHeight?: number;
    disabledPresetKeys?: string[];
    dragSourceSelector?: string;
  }>(),
  {
    preview: 'desktop',
    interactionMode: 'move',
    rowHeight: 72,
    getComponentProps: undefined,
    selectedWidgetId: null,
    disabledPresetKeys: () => [],
    dragSourceSelector: '.dashboard-widget-palette__drag-source',
  },
);
const emit = defineEmits<{
  'update:modelValue': [value: DashboardSchema<TType>];
  remove: [id: string];
  select: [id: string];
  edit: [id: string];
  duplicate: [id: string];
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
  cellHeight: props.rowHeight,
  // 行高由业务文档控制，组件间距保持紧凑以形成连续的仪表盘画布。
  margin: '4px',
  float: true,
  // 设计态即使没有组件也保留完整的垂直网格，作为外部拖入的命中区域。
  minRow: 12,
  // 移动模式仍保留卡片边缘缩放；“调整尺寸”只收紧为专用模式，避免误拖卡片位置。
  disableDrag: props.interactionMode === 'resize',
  disableResize: false,
  acceptWidgets: (element: Element) =>
    element instanceof HTMLElement &&
    element.matches(props.dragSourceSelector) &&
    !props.disabledPresetKeys.includes(element.dataset.widgetKey ?? ''),
  draggable: { handle: '.dashboard-widget__drag-handle' },
  // 右侧拖宽，右下角对角手柄同时调整宽高；常驻手柄与参考设计保持一致。
  resizable: { handles: 'e,se', autoHide: false },
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
    // 首行上方没有安全空间，操作条回落到卡片内部；其余行浮在卡片外避免遮挡内容。
    actionPlacement: resolveDashboardWidgetActionPlacement(widget.y),
    onRemove: () => emit('remove', widget.id),
    onSelect: () => emit('select', widget.id),
    onEdit: () => emit('edit', widget.id),
    onDuplicate: () => emit('duplicate', widget.id),
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
    <div
      class="dashboard-design-canvas__scroll"
      :class="{ 'dashboard-design-canvas__scroll--mobile': preview === 'mobile' }"
    >
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
/* Vue 的 :deep() 用于定制 GridStack 运行时生成的拖拽与缩放节点。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.dashboard-design-canvas {
  box-sizing: border-box;
  flex: 1;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--el-bg-color-page);

  &__scroll {
    box-sizing: border-box;
    width: 100%;
    min-width: 0;
    height: 100%;
    overflow: hidden auto;
  }

  /* 窄屏预览有固定最小宽度，仅该模式允许横向查看完整画布。 */
  &__scroll--mobile {
    overflow-x: auto;
  }

  &__surface {
    box-sizing: border-box;
    width: 100%;
    min-width: 0;
    height: 100%;
    min-height: 100%;
    padding: 8px;
  }

  &__surface--desktop {
    min-width: 0;
  }

  &__surface--mobile {
    width: 100%;
    min-width: 420px;
    max-width: 480px;
    margin: 0 auto;
  }

  /* GridStack 必须收敛到画布内容宽度，避免缩放手柄制造像素级横向溢出。 */
  :deep(.evolyn-grid) {
    width: 100%;
    min-width: 0;
  }

  /* 设计态允许操作条越过卡片边界，业务内容由宿主内部的裁切层负责收口。 */
  :deep(.evolyn-grid .grid-stack-item-content) {
    overflow: visible !important;
  }

  /* 浮在卡片外的操作条需要高于相邻网格项，选中和键盘聚焦状态保持一致。 */
  :deep(.evolyn-grid .grid-stack-item:has(.dashboard-design-widget--selected)),
  :deep(.evolyn-grid .grid-stack-item:focus-within),
  :deep(.evolyn-grid .grid-stack-item:hover) {
    z-index: 10;
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
      opacity: 0.5;
      clip-path: polygon(100% 0, 100% 100%, 0 100%);
    }
  }

  /* 单行问候语只允许横向调整，避免显示无效的右下角双向手柄。 */
  :deep(.grid-stack-item[gs-max-h='1'] > .ui-resizable-se) {
    display: none;
  }
}
</style>
