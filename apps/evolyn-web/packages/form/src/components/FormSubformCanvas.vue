<template>
  <div class="form-subform-canvas">
    <div class="form-subform-canvas__table" :class="{ 'is-empty': !childFields.length }">
      <div class="form-subform-canvas__index-column">
        <div class="form-subform-canvas__index-cell form-subform-canvas__index-cell--header"></div>
        <div class="form-subform-canvas__index-cell">1</div>
      </div>
      <Draggable
        :list="childFields"
        :group="subformDragGroup"
        item-key="fieldKey"
        ghost-class="form-subform-canvas__column--ghost"
        chosen-class="form-subform-canvas__column--chosen"
        :class="['form-subform-canvas__columns', { 'is-empty': !childFields.length }]"
        :animation="180"
        @add="handleDragAdd"
      >
        <template #item="{ element: childField }">
          <div
            class="form-subform-canvas__column"
            :class="{ 'is-active': selectedFieldKey === childField.fieldKey }"
            @click.stop="selectField(childField.fieldKey)"
          >
            <div
              v-if="selectedFieldKey === childField.fieldKey"
              class="form-subform-canvas__actions"
            >
              <button type="button" @click.stop="copyField(childField.fieldKey)">
                <el-icon><CopyDocument /></el-icon>
              </button>
              <el-popconfirm
                placement="bottom-end"
                :width="220"
                hide-icon
                confirm-button-type="danger"
                :title="t('确定删除此项？删除后不可恢复，请确保你的工作不受影响')"
                :cancel-button-text="t('取消')"
                :confirm-button-text="t('删除')"
                @confirm="removeField(childField.fieldKey)"
              >
                <template #reference>
                  <!-- 子字段与外层字段保持一致，仅在二次确认后执行删除。 -->
                  <button type="button" @click.stop>
                    <el-icon><Delete /></el-icon>
                  </button>
                </template>
              </el-popconfirm>
            </div>
            <!-- 空名称仍保留表头单元格，确保操作按钮位置和各列高度保持一致。 -->
            <div class="form-subform-canvas__cell form-subform-canvas__cell--header">
              <el-tooltip
                v-if="hasFieldLabel(childField)"
                :content="childField.fieldLabel"
                :visible="visibleTooltipFieldKey === childField.fieldKey"
                placement="top"
              >
                <span
                  class="form-subform-canvas__field-label"
                  @mouseenter="showFieldLabelTooltip($event, childField.fieldKey)"
                  @mouseleave="hideFieldLabelTooltip(childField.fieldKey)"
                >
                  {{ childField.fieldLabel }}
                </span>
              </el-tooltip>
            </div>
            <div class="form-subform-canvas__cell form-subform-canvas__cell--body">
              <FormDesignFieldControl
                :field="childField"
                :model-value="childField.defaultValue"
                disabled
              />
            </div>
          </div>
        </template>
        <template #footer>
          <div v-if="!childFields.length" class="form-subform-canvas__empty">
            {{ t('从左侧拖拽来添加字段') }}
          </div>
        </template>
      </Draggable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElIcon, ElPopconfirm, ElTooltip } from 'element-plus';
import { computed, ref } from 'vue';
import Draggable from 'vuedraggable';
import { CopyDocument, Delete } from '@element-plus/icons-vue';
import type { FormDesignDragField, FormDesignTemplateField } from '../types';
import FormDesignFieldControl from './FormDesignFieldControl.vue';

/**
 * 新子表单设计画布预览组件。
 * @property fields 子表单子字段列表，来源于父字段的 fieldConf.fields。
 */
const props = defineProps<{
  fields?: FormDesignTemplateField[];
  selectedFieldKey?: string;
}>();

const t = (text: string) => text;
const emits = defineEmits<{
  (event: 'select-field', fieldKey: string): void;
  (event: 'copy-field', fieldKey: string): void;
  (event: 'remove-field', fieldKey: string): void;
  (event: 'add-drag-field', value: FormDesignDragField): void;
}>();

const childFields = computed(() => props.fields || []);
const visibleTooltipFieldKey = ref('');

// 子字段显示名称仅包含空白时按空名称处理，避免渲染无意义文本节点。
const hasFieldLabel = (field: FormDesignTemplateField) => Boolean(field.fieldLabel?.trim());

/**
 * 字段名称实际发生横向溢出时才展示完整内容，避免短名称出现多余提示。
 * @param event 字段名称的鼠标移入事件。
 * @param fieldKey 当前子字段标识。
 */
const showFieldLabelTooltip = (event: MouseEvent, fieldKey: string) => {
  const labelElement = event.currentTarget;
  if (!(labelElement instanceof HTMLElement)) return;
  visibleTooltipFieldKey.value =
    labelElement.scrollWidth > labelElement.clientWidth ? fieldKey : '';
};

// 鼠标离开当前名称时仅关闭对应提示，避免快速切换字段造成提示闪烁。
const hideFieldLabelTooltip = (fieldKey: string) => {
  if (visibleTooltipFieldKey.value === fieldKey) visibleTooltipFieldKey.value = '';
};

const subformDragGroup = {
  name: 'plugin-design-fields',
  pull: false,
  put: (_to: unknown, _from: unknown, dragElement?: HTMLElement) => {
    // 子表单内部只允许普通字段，禁止继续嵌套子表单，避免生成多层表格结构。
    const widgetName = dragElement?.dataset?.widgetName;
    return Boolean(
      dragElement?.classList.contains('form-design-palette__item') && widgetName !== 'subforms',
    );
  },
};

const isPaletteCloneField = (field?: FormDesignTemplateField) => {
  return Boolean(field?.fieldKey?.startsWith('palette_'));
};

const findPaletteCloneFieldIndex = (preferredIndex: number) => {
  // 空子表单存在 footer 占位时，Sortable 的 DOM 下标可能偏移，这里按临时克隆项本身反查真实数组位置。
  if (preferredIndex >= 0 && isPaletteCloneField(childFields.value[preferredIndex])) {
    return preferredIndex;
  }
  return childFields.value.findIndex(isPaletteCloneField);
};

const removePaletteCloneField = (preferredIndex: number) => {
  const cloneIndex = findPaletteCloneFieldIndex(preferredIndex);
  if (cloneIndex >= 0) childFields.value.splice(cloneIndex, 1);
  return cloneIndex;
};

const handleDragAdd = (event: {
  newIndex?: number;
  newDraggableIndex?: number;
  item?: HTMLElement;
}) => {
  const index = event.newDraggableIndex ?? event.newIndex ?? -1;
  const cloneIndex = findPaletteCloneFieldIndex(index);
  const dragItemIndex = cloneIndex >= 0 ? cloneIndex : index;
  const dragItem = dragItemIndex >= 0 ? childFields.value[dragItemIndex] : undefined;
  const widgetName = dragItem?.widgetName || event.item?.dataset?.widgetName;
  const dataType = dragItem?.dataType;
  const insertIndex = removePaletteCloneField(index);
  // Sortable 某些情况下可能绕过 put 后仍触发 add，这里再兜底拦截子表单嵌套。
  if (!widgetName || typeof dataType !== 'string' || widgetName === 'subforms') return;
  emits('add-drag-field', {
    index: insertIndex >= 0 ? insertIndex : index,
    widgetName,
    dataType,
  });
};

const selectField = (fieldKey: string) => {
  emits('select-field', fieldKey);
};

const copyField = (fieldKey: string) => {
  emits('copy-field', fieldKey);
};

const removeField = (fieldKey: string) => {
  emits('remove-field', fieldKey);
};
</script>

<style lang="scss">
.form-subform-canvas {
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;

  &__table {
    // 表格宽度随固定列累加，同时至少铺满可视区，滚动后边框和操作按钮仍位于完整表格内。
    box-sizing: border-box;
    display: inline-flex;
    width: max-content;
    min-width: 100%;
    background-color: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);

    &.is-empty {
      display: inline-flex;
      width: auto;
      min-width: 0;
    }
  }

  &__columns {
    display: flex;
    flex: 1 0 auto;
    width: max-content;
    min-width: 0;
    min-height: calc(var(--el-space-6xl) + var(--el-space-4xl));

    &.is-empty {
      flex: 0 0 176px;
      min-width: 176px;
    }

    .form-design-palette__item,
    .sortable-ghost {
      position: relative;
      display: block;
      flex: 0 0 128px;
      align-self: stretch;
      width: 128px;
      min-width: 128px;
      min-height: calc(var(--el-space-6xl) + var(--el-space-4xl));
      padding: 0;
      margin: 0;
      color: transparent;
      background-color: var(--el-fill-color-extra-light);
      border: 1px dashed var(--el-border-color);
      border-radius: 0;
      box-shadow: none;
    }

    .form-design-palette__item .el-icon,
    .sortable-ghost .el-icon {
      display: none;
    }

    .form-design-palette__item span,
    .sortable-ghost span {
      display: none;
    }
  }

  &__column {
    position: relative;
    flex: 0 0 160px;
    min-width: 160px;
    cursor: pointer;

    &.is-active::after {
      position: absolute;
      inset: 0;
      z-index: 3;
      pointer-events: none;
      content: '';
      border: 1px dashed var(--el-color-primary);
    }

    &--ghost {
      background-color: var(--el-fill-color-light);
      opacity: 0.5;
    }

    &--chosen {
      box-shadow: var(--el-box-shadow);
    }
  }

  &__index-column {
    flex: 0 0 56px;
    min-width: 56px;
    border-right: 1px solid var(--el-border-color);
  }

  &__index-cell {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: var(--el-space-6xl);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-regular);

    &--header {
      min-height: var(--el-space-4xl);
      background-color: var(--el-fill-color-light);
      border-bottom: 1px solid var(--el-border-color);
    }
  }

  &__cell {
    position: relative;
    box-sizing: border-box;
    display: flex;
    align-items: center;
    padding: var(--el-space-xs) var(--el-space-sm);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
    border-right: 1px solid var(--el-border-color);

    &--header {
      min-height: var(--el-space-4xl);
      background-color: var(--el-fill-color-light);
      border-bottom: 1px solid var(--el-border-color);
    }

    &--body {
      min-height: var(--el-space-6xl);
    }

    .form-subform-canvas__field-label {
      display: block;
      min-width: 0;
      max-width: 100%;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .el-input,
    .el-select,
    .el-date-editor,
    .el-input-number,
    .el-button {
      width: 100%;
      pointer-events: none;
    }
  }

  &__column.is-active &__cell--header {
    padding-right: var(--el-space-6xl);
  }

  &__selector {
    width: 100%;
  }

  &__actions {
    position: absolute;
    top: var(--el-space-xs);
    right: var(--el-space-xs);
    z-index: 2;
    display: flex;
    overflow: hidden;
    background-color: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow);

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

  &__empty {
    box-sizing: border-box;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 176px;
    min-height: calc(var(--el-space-6xl) + var(--el-space-4xl));
    padding: 0 var(--el-space-lg);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }
}
</style>
