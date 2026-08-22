<script setup lang="ts">
import { ElScrollbar } from 'element-plus';
import { EvolynGrid, type EvolynGridItem } from '@evolyn.do/ui';
import { computed, markRaw, ref } from 'vue';
import type { GridStack as GridStackInstance, GridStackNode } from 'gridstack';
import type { DashboardWidget, DashboardWidgetContent } from '~/types/dashboard';
import WorkbenchEditorWidgetHost from './WorkbenchEditorWidgetHost.vue';

const props = defineProps<{
  modelValue: DashboardWidget[];
  device: 'desktop' | 'mobile';
  disabledKeys: string[];
}>();
const emit = defineEmits<{
  'update:modelValue': [value: DashboardWidget[]];
}>();

const components = { WorkbenchEditorWidgetHost: markRaw(WorkbenchEditorWidgetHost) };
const grid = ref<{ getGrid: () => GridStackInstance | null }>();
const editorItems = computed<DashboardWidget[]>(() =>
  props.modelValue.map((item) => ({
    ...item,
    component: 'WorkbenchEditorWidgetHost',
    props: { widget: toWidgetContent(item), onRemove: removeWidget },
  })),
);
const options = computed(() => ({
  column: props.device === 'desktop' ? 12 : 1,
  cellHeight: props.device === 'desktop' ? 72 : 92,
  // 相邻卡片的上下边距各 8px，视觉间距为 16px。
  margin: '8px 14px',
  float: true,
  acceptWidgets: (element: Element) =>
    element instanceof HTMLElement &&
    element.matches('.widget-palette__drag-source') &&
    !props.disabledKeys.includes(element.dataset.widgetKey ?? ''),
  draggable: { handle: '.dashboard-widget__drag-handle' },
  // 右侧拖宽，右下角对角手柄同时调整宽高；仅悬停时显示操作入口。
  resizable: { handles: 'e,se', autoHide: true },
}));

/** 编辑器宿主只上报操作，画布作为布局数据的唯一写入方。 */
function removeWidget(id: string) {
  emit(
    'update:modelValue',
    props.modelValue.filter((item) => item.id !== id),
  );
}

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

/** GridStack 为可重复拖入的同名节点追加序号，持久化时仍需记录原始预设键。 */
function getPaletteKey(widgetID: string) {
  return widgetID.replace(/^palette-/, '').replace(/_\d+$/, '');
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
      maxW: current.maxW,
      maxH: current.maxH,
      component: 'WorkbenchEditorWidgetHost',
      presetKey: getPaletteKey(droppedId),
    });
  }

  emit('update:modelValue', next);
}
</script>

<template>
  <section class="workbench-design-canvas">
    <el-scrollbar class="workbench-design-canvas__scrollbar" always>
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
  background: #f3f3f8;

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
