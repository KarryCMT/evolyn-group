<template>
  <EvolynScrollbar class="form-schema-canvas">
    <Draggable
      :list="items"
      :group="{ name: FORM_SCHEMA_DRAG_GROUP, pull: true, put: true }"
      :item-key="itemKeyOf"
      ghost-class="form-schema-canvas__item--ghost"
      chosen-class="form-schema-canvas__item--chosen"
      class="form-schema-canvas__list"
      :animation="180"
      :empty-insert-threshold="120"
      :swap-threshold="0.55"
      :invert-swap="true"
      @add="handleDragAdd"
    >
      <template #item="{ element }">
        <div
          class="form-schema-canvas__item"
          :class="{
            'is-active': element.widget && selectedKey === itemKeyOf(element),
            'is-label-empty': !hasLabel(element),
            'is-layout': isLayoutItem(element),
            'form-schema-canvas__item--pending': !element.widget,
          }"
          :style="fieldSpanStyle(element)"
          @click="onItemClick(element)"
        >
          <!-- vuedraggable 要求 item 插槽单根元素（顶层注释也不行）：
               素材面板拖入的临时对象（仅 paletteType、无 widget，add 事件内会被
               真实字段项替换）与真实字段共用同一根节点，经 --pending 修饰符切换
               占位形态，内部内容按 widget 存在性分支。 -->
          <template v-if="element.widget">
            <!-- 分割线等布局项直接整行预览，无操作头布局差异 -->
            <div class="form-schema-canvas__item-header">
              <span v-if="hasLabel(element)" class="form-schema-canvas__item-label">
                {{ element.label }}
                <span
                  v-if="!element.widget.allowBlank"
                  class="form-schema-canvas__item-required"
                  aria-hidden="true"
                  >*</span
                >
              </span>
              <span class="form-schema-canvas__item-widget-name" :title="element.widget.widgetName">
                {{ element.widget.widgetName }}
              </span>
              <div class="form-schema-canvas__actions">
                <button type="button" title="复制字段" @click.stop="$emit('copy-item', element)">
                  <el-icon><CopyDocument /></el-icon>
                </button>
                <el-popconfirm
                  placement="bottom-end"
                  :width="220"
                  hide-icon
                  confirm-button-type="danger"
                  title="确定删除此项？删除后不可恢复，请确保你的工作不受影响"
                  cancel-button-text="取消"
                  confirm-button-text="删除"
                  @confirm="$emit('remove-item', element.widget.widgetName)"
                >
                  <template #reference>
                    <button type="button" title="删除字段" @click.stop>
                      <el-icon><Delete /></el-icon>
                    </button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
            <!-- 画布内控件仅用于设计预览，点击控件区域统一选中字段本身。 -->
            <div class="form-schema-canvas__control">
              <FormSchemaItemPreview :item="element" />
            </div>
          </template>
        </div>
      </template>
    </Draggable>
    <!-- 空状态不进入排序流，保证首次拖入投影从画布顶部出现。 -->
    <div v-if="items.length === 0" class="form-schema-canvas__empty" role="status">
      从左侧选择字段或将字段拖入画布
    </div>
  </EvolynScrollbar>
</template>

<script setup lang="ts">
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElIcon, ElPopconfirm } from 'element-plus';
import { CopyDocument, Delete } from '@element-plus/icons-vue';
import Draggable from 'vuedraggable';
import type { FormItem } from '../schema/types';
import { isLayoutWidgetType } from '../schema/codec';
import { FORM_SCHEMA_DRAG_GROUP, type FormSchemaPaletteDrag } from './palette';
import FormSchemaItemPreview from './FormSchemaItemPreview.vue';

/**
 * 字段设计画布：目标协议 items 的拖拽排序、选择、复制与删除。
 * 画布状态由页面持有（useFormSchemaEditor），组件只透传操作事件；
 * 素材面板拖入的临时对象（paletteType 标记）经 add 事件上报，由页面
 * 替换为 createWidgetItem 生成的真实字段项。
 */
const props = defineProps<{
  items: FormItem[];
  selectedKey: string;
}>();

const emits = defineEmits<{
  (event: 'select-item', key: string): void;
  (event: 'copy-item', item: FormItem): void;
  (event: 'remove-item', key: string): void;
  (event: 'add-field', value: { type: string; index: number }): void;
}>();

// item-key 兼容素材面板拖入的临时对象（paletteKey）与真实字段项（widgetName）。
const itemKeyOf = (element: unknown): string => {
  const item = element as FormItem & Partial<FormSchemaPaletteDrag>;
  return item.widget?.widgetName ?? item.paletteType ?? '';
};

// 跨列表克隆时事件下标可能与响应式数组更新不同步，按 palette 标记反查真实位置。
const handleDragAdd = (event: { newIndex?: number; newDraggableIndex?: number }) => {
  const eventIndex = event.newDraggableIndex ?? event.newIndex ?? -1;
  const markerIndex = props.items.findIndex(
    (item) => (item as FormItem & Partial<FormSchemaPaletteDrag>).paletteType !== undefined,
  );
  const index = markerIndex >= 0 ? markerIndex : eventIndex;
  if (index < 0 || index >= props.items.length) return;
  const marker = props.items[index] as FormItem & Partial<FormSchemaPaletteDrag>;
  const type = marker.paletteType;
  if (typeof type !== 'string') return;
  emits('add-field', { type, index });
};

// 可选链防御瞬时临时对象/异常数据（无 label 的元素不渲染标题行）。
const hasLabel = (item: FormItem) => Boolean(item.label?.trim());
const widgetOf = (item: FormItem) => (item as FormItem & Partial<FormSchemaPaletteDrag>).widget;
const isLayoutItem = (item: FormItem) => isLayoutWidgetType(widgetOf(item)?.type ?? '');
// 临时对象（无 widget）只渲染占位形态，点击不产生选中事件。
const onItemClick = (item: FormItem) => {
  const key = widgetOf(item)?.widgetName;
  if (key) emits('select-item', key);
};
// 预览区按 lineWidth 收敛展示宽度（12 栅格语义与运行时一致；布局项整行）。
const fieldSpanStyle = (item: FormItem) => {
  if (!widgetOf(item) || isLayoutItem(item)) return undefined;
  const width =
    Number.isInteger(item.lineWidth) && item.lineWidth >= 1 && item.lineWidth <= 12
      ? item.lineWidth
      : 12;
  return { width: `${(width / 12) * 100}%` };
};
</script>

<style lang="scss">
.form-schema-canvas {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: var(--el-space-2xl);
  border-top: 1px solid var(--el-border-color);

  &__list {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: flex-start;
    min-height: 0;
    padding-bottom: var(--el-space-4xl);

    // 单层虚线投影明确提示字段最终插入位置。
    & > .form-schema-palette__item,
    & > .form-schema-canvas__item--ghost,
    & > .sortable-ghost {
      position: relative;
      box-sizing: border-box;
      display: block;
      width: 100%;
      min-height: var(--el-space-4xl);
      padding: 0;
      margin: 0 0 var(--el-space-lg);
      color: transparent;
      background-color: transparent;
      border: 1px dashed var(--el-color-primary);
      border-radius: var(--el-border-radius-base);
      box-shadow: none;
      opacity: 1;
    }

    & > .form-schema-palette__item > *,
    & > .form-schema-canvas__item--ghost > *,
    & > .sortable-ghost > * {
      display: none;
    }
  }

  &__empty {
    position: absolute;
    inset: var(--el-space-2xl);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
    pointer-events: none;
  }

  &__item {
    position: relative;
    box-sizing: border-box;
    width: 100%;
    padding: var(--el-space-md) var(--el-space-xl);
    margin-bottom: var(--el-space-lg);
    cursor: grab;
    background-color: transparent;
    border: 1px solid transparent;
    border-radius: var(--el-border-radius-base);
    transition:
      background-color 0.16s ease,
      border-color 0.16s ease;

    &:active {
      cursor: grabbing;
    }

    &:hover {
      background-color: var(--el-fill-color-light);
    }

    &.is-active {
      background-color: var(--el-color-primary-light-3);
      border-color: var(--el-color-primary);
    }

    &.is-label-empty .form-schema-canvas__item-header {
      min-height: var(--el-space-3xl);
    }

    &.is-layout {
      .form-schema-canvas__item-label {
        font-weight: 600;
      }

      .form-schema-canvas__control {
        pointer-events: none;
      }
    }

    &:hover .form-schema-canvas__actions,
    &.is-active .form-schema-canvas__actions {
      pointer-events: auto;
      opacity: 1;
    }

    &--ghost {
      background-color: var(--el-fill-color-light);
      border-color: var(--el-color-primary);
      opacity: 1;
    }

    &--chosen {
      box-shadow: var(--el-box-shadow);
    }

    // 素材拖入的临时对象占位（正常会在事件内被真实字段替换）
    &--pending {
      display: block;
      width: 100%;
      min-height: var(--el-space-4xl);
      margin: 0 0 var(--el-space-lg);
      border: 1px dashed var(--el-color-primary);
      border-radius: var(--el-border-radius-base);
    }
  }

  &__item-header {
    display: flex;
    gap: var(--el-space-xs);
    align-items: center;
    min-height: var(--el-space-3xl);
    padding-right: var(--el-space-6xl);
    margin-bottom: var(--el-space-xs);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__item-label {
    flex-shrink: 0;
    min-width: 0;
  }

  &__item-required {
    margin-left: var(--el-space-xs);
    color: var(--el-color-error);
  }

  &__item-widget-name {
    overflow: hidden;
    font-weight: 400;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__control {
    pointer-events: none;

    .el-input,
    .el-select,
    .el-input-number {
      width: 100%;
    }
  }

  &__actions {
    position: absolute;
    top: var(--el-space-xs);
    right: var(--el-space-xl);
    display: flex;
    overflow: hidden;
    pointer-events: none;
    background-color: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow);
    opacity: 0;
    transition: opacity 0.16s ease;

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 24px;
      height: 24px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background-color: transparent;
      border: 0;

      & + button {
        border-left: 1px solid var(--el-border-color);
      }

      &:hover {
        color: var(--el-text-color-primary);
        background-color: var(--el-fill-color-light);
      }
    }
  }
}
</style>
