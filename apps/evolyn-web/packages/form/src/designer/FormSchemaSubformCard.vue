<script setup lang="ts">
import { RiDeleteBin6Fill, RiFileCopyFill } from '@remixicon/vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElIcon, ElPopconfirm } from 'element-plus';
import { computed, nextTick, ref, watch } from 'vue';
import Draggable from 'vuedraggable';
import type { FormItem, SubformWidget } from '../schema/types';
import { SUBFORM_ALLOWED_WIDGET_TYPES } from '../schema/types';
import {
  FORM_SCHEMA_DRAG_GROUP,
  FORM_SCHEMA_SUBFORM_DRAG_GROUP,
  type FormSchemaPaletteDrag,
} from './palette';
import { subformSelectionKey } from './useFormSchemaEditor';
import FormSchemaItemPreview from './FormSchemaItemPreview.vue';

/**
 * 子表单设计卡片：外层负责整块选中，内层以“字段列 + 一行预览”表达桌面表格形态。
 * 嵌套列表只接收素材面板载荷和自身排序，不允许子字段拖回顶层，避免布局引用跨作用域泄漏。
 */
const props = defineProps<{ item: FormItem<SubformWidget>; selectedKey: string }>();

const emit = defineEmits<{
  select: [key: string];
  selectChild: [parentKey: string, childKey: string];
  copy: [item: FormItem];
  remove: [key: string];
  replaceChildren: [parentKey: string, entries: unknown[]];
  copyChild: [parentKey: string, childKey: string];
  removeChild: [parentKey: string, childKey: string];
}>();

const widget = computed(() => props.item.widget);
const parentKey = computed(() => widget.value.widgetName);
const isOuterSelected = computed(() => props.selectedKey === parentKey.value);
const tableRef = ref<HTMLElement>();

function childKey(entry: unknown): string {
  if (isFormItem(entry)) return entry.widget.widgetName;
  return (entry as Partial<FormSchemaPaletteDrag> | null)?.paletteType ?? '';
}

function isChildSelected(item: FormItem): boolean {
  return props.selectedKey === subformSelectionKey(parentKey.value, item.widget.widgetName);
}

/** 子字段选中时，父级不应因鼠标仍位于其内部而显示悬停/选中视觉。 */
const hasChildSelected = computed(() => widget.value.items.some(isChildSelected));

/** 新增或选中超出可视区的子字段时，将横向表格定位到该字段，避免字段标题停在裁切边缘。 */
watch(
  () => props.selectedKey,
  async () => {
    const table = tableRef.value;
    const child = widget.value.items.find(isChildSelected);
    if (!table || !child) return;

    await nextTick();
    const column = Array.from(
      table.querySelectorAll<HTMLElement>('.form-schema-subform-card__column'),
    ).find((element) => element.dataset.childKey === child.widget.widgetName);
    column?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
  },
  { flush: 'post' },
);

function allowSubformMove(event: { draggedContext?: { element?: unknown } }): boolean {
  const entry = event.draggedContext?.element;
  if (isFormItem(entry)) {
    return widget.value.items.includes(entry);
  }
  const type = (entry as Partial<FormSchemaPaletteDrag> | null)?.paletteType;
  return typeof type === 'string' && SUBFORM_ALLOWED_WIDGET_TYPES.includes(type as never);
}

function isFormItem(entry: unknown): entry is FormItem {
  return Boolean(entry && typeof entry === 'object' && 'widget' in entry);
}
</script>

<template>
  <article
    class="form-schema-subform-card"
    :class="{ 'is-active': isOuterSelected, 'has-child-selected': hasChildSelected }"
    @click="emit('select', parentKey)"
  >
    <header class="form-schema-subform-card__header">
      <span class="form-schema-subform-card__title">
        <span v-if="!item.widget.allowBlank" class="form-schema-subform-card__required">*</span>
        {{ item.label }}
      </span>
      <div class="form-schema-subform-card__actions">
        <button type="button" title="复制子表单" @click.stop="emit('copy', item)">
          <el-icon><RiFileCopyFill /></el-icon>
        </button>
        <el-popconfirm
          placement="bottom-end"
          :width="220"
          hide-icon
          confirm-button-type="danger"
          title="确定删除此子表单？其中子字段也会一并删除"
          cancel-button-text="取消"
          confirm-button-text="删除"
          @confirm="emit('remove', parentKey)"
        >
          <template #reference>
            <button type="button" title="删除子表单" @click.stop>
              <el-icon><RiDeleteBin6Fill /></el-icon>
            </button>
          </template>
        </el-popconfirm>
      </div>
    </header>

    <EvolynScrollbar class="form-schema-subform-card__table" @click.stop>
      <div ref="tableRef" class="form-schema-subform-card__table-content">
        <div class="form-schema-subform-card__index-column" aria-hidden="true">
          <div class="form-schema-subform-card__column-head"></div>
          <div class="form-schema-subform-card__column-preview">1</div>
        </div>
        <Draggable
          :model-value="widget.items"
          :group="{
            name: FORM_SCHEMA_SUBFORM_DRAG_GROUP,
            pull: false,
            put: [FORM_SCHEMA_DRAG_GROUP, FORM_SCHEMA_SUBFORM_DRAG_GROUP],
          }"
          :item-key="childKey"
          class="form-schema-subform-card__columns"
          ghost-class="form-schema-subform-card__ghost"
          :animation="180"
          :dragover-bubble="false"
          handle=".form-schema-subform-card__column-head"
          filter="button"
          :prevent-on-filter="false"
          :move="allowSubformMove"
          @update:model-value="emit('replaceChildren', parentKey, $event)"
        >
          <template #item="{ element }">
            <section
              v-if="isFormItem(element)"
              class="form-schema-subform-card__column"
              :class="{ 'is-active': isChildSelected(element) }"
              :data-child-key="element.widget.widgetName"
              @click.stop="emit('selectChild', parentKey, element.widget.widgetName)"
            >
              <header class="form-schema-subform-card__column-head">
                <span :title="element.label">{{ element.label }}</span>
                <div class="form-schema-subform-card__column-actions">
                  <button
                    type="button"
                    title="复制子字段"
                    @click.stop="emit('copyChild', parentKey, element.widget.widgetName)"
                  >
                    <el-icon><RiFileCopyFill /></el-icon>
                  </button>
                  <el-popconfirm
                    placement="bottom-end"
                    :width="200"
                    hide-icon
                    confirm-button-type="danger"
                    title="确定删除此子字段？"
                    cancel-button-text="取消"
                    confirm-button-text="删除"
                    @confirm="emit('removeChild', parentKey, element.widget.widgetName)"
                  >
                    <template #reference>
                      <button type="button" title="删除子字段" @click.stop>
                        <el-icon><RiDeleteBin6Fill /></el-icon>
                      </button>
                    </template>
                  </el-popconfirm>
                </div>
              </header>
              <div class="form-schema-subform-card__column-preview">
                <FormSchemaItemPreview :item="element" />
              </div>
            </section>
            <div v-else class="form-schema-subform-card__pending">正在添加子字段…</div>
          </template>
          <template #footer>
            <button
              v-if="widget.items.length === 0"
              class="form-schema-subform-card__empty"
              type="button"
              @click="emit('select', parentKey)"
            >
              从左侧拖入字段，或在右侧添加子字段
            </button>
          </template>
        </Draggable>
      </div>
    </EvolynScrollbar>
  </article>
</template>

<style scoped lang="scss">
.form-schema-subform-card {
  position: relative;
  box-sizing: border-box;
  width: 100%;
  min-height: 168px;
  padding: var(--el-space-xl);
  cursor: grab;
  background: var(--el-fill-color-lighter);
  border: 1px solid transparent;
  border-radius: var(--el-border-radius-base);

  &:hover:not(.has-child-selected),
  &.is-active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-7);
  }

  &:hover:not(.has-child-selected) &__actions,
  &.is-active &__actions {
    pointer-events: auto;
    opacity: 1;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 28px;
    margin-bottom: var(--el-space-md);
  }

  &__title {
    font-size: var(--el-font-size-base);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__required {
    color: var(--el-color-error);
  }

  &__actions,
  &__column-actions {
    display: flex;
    overflow: hidden;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow-light);
  }

  &__actions {
    pointer-events: none;
    opacity: 0;
  }

  &__actions button,
  &__column-actions button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  &__table {
    width: 100%;
    min-height: 96px;
    // 子表单是横向字段表格，拖拽占位块的边框不应触发纵向滚动条。
    overflow-y: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__table-content {
    display: flex;
    min-width: max-content;
    min-height: 96px;
  }

  &__index-column {
    flex: 0 0 72px;
    text-align: center;
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__columns {
    display: flex;
    min-width: 220px;
  }

  &__column,
  &__pending {
    box-sizing: border-box;
    flex: 0 0 220px;
    min-width: 220px;
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__column {
    cursor: grab;
    outline: 0;
  }

  &__column.is-active {
    outline: 1px dashed var(--el-color-primary);
    outline-offset: -1px;
  }

  &__column-head {
    box-sizing: border-box;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    column-gap: var(--el-space-sm);
    align-items: center;
    min-height: 38px;
    padding: 0 var(--el-space-md);
    overflow: hidden;
    font-size: var(--el-font-size-extra-small);
    font-weight: 500;
    color: var(--el-text-color-primary);
    white-space: nowrap;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__column-head > span {
    // 网格首列总是扣除操作区后的可用宽度，标题过长时才省略。
    display: block;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  &__column-actions {
    margin-left: 0;
    opacity: 0;
  }

  &__column:hover &__column-actions,
  &__column.is-active &__column-actions {
    opacity: 1;
  }

  &__column-preview {
    box-sizing: border-box;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 58px;
    padding: var(--el-space-sm);
  }

  &__empty {
    width: 360px;
    min-height: 96px;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 1px dashed transparent;
  }

  &__empty:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
  }

  &__pending,
  &__ghost {
    color: transparent;
    background: var(--el-color-primary-light-9);
    border: 1px dashed var(--el-color-primary);
  }
}
</style>
