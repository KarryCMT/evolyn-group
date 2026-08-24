<template>
  <div class="form-field-canvas">
    <Draggable
      :list="fields"
      :group="{ name: 'plugin-design-fields', pull: true, put: true }"
      item-key="fieldKey"
      ghost-class="form-field-canvas__field--ghost"
      chosen-class="form-field-canvas__field--chosen"
      class="form-field-canvas__list"
      :animation="180"
      :empty-insert-threshold="120"
      :swap-threshold="0.55"
      :invert-swap="true"
      @add="handleDragAdd"
      @update="handleSortFields"
    >
      <template #item="{ element }">
        <div
          class="form-field-canvas__field"
          :class="{
            'is-active': selectedFieldKey === element.fieldKey,
            'is-label-empty': !hasFieldLabel(element),
          }"
          @click="$emit('select-field', element.fieldKey)"
        >
          <!-- 显示名称为空时保留操作行，避免按钮与下方控件重叠。 -->
          <div class="form-field-canvas__field-header">
            <span v-if="hasFieldLabel(element)" class="form-field-canvas__field-label">
              {{ element.fieldLabel }}
              <!-- 画布预览与属性面板的必填状态保持同步，星号紧跟字段文案展示。 -->
              <span
                v-if="element.isRequired"
                class="form-field-canvas__field-required"
                aria-hidden="true"
                >*</span
              >
            </span>
            <div class="form-field-canvas__actions">
              <button type="button" @click.stop="$emit('copy-field', element)">
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
                @confirm="$emit('remove-field', element.fieldKey)"
              >
                <template #reference>
                  <!-- 删除按钮仅打开局部确认浮层，确认后才沿用原有删除事件。 -->
                  <button type="button" @click.stop>
                    <el-icon><Delete /></el-icon>
                  </button>
                </template>
              </el-popconfirm>
            </div>
          </div>
          <!-- 设计画布中的控件仅用于预览，点击控件区域时统一选中外层字段。 -->
          <div v-if="!isSubformField(element)" class="form-field-canvas__control">
            <FormDesignFieldControl
              :field="element"
              disabled
              :model-value="element.defaultValue"
              @update:model-value="$emit('update-field-default-value', element.fieldKey, $event)"
            />
          </div>
          <FormSubformCanvas
            v-else
            :fields="getSubformFields(element)"
            :selected-field-key="
              selectedFieldKey === element.fieldKey ? selectedSubformChildFieldKey : ''
            "
            @add-drag-field="
              $emit('add-subform-drag-field', { parentKey: element.fieldKey, ...$event })
            "
            @copy-field="
              $emit('copy-subform-child', { parentKey: element.fieldKey, childKey: $event })
            "
            @remove-field="
              $emit('remove-subform-child', { parentKey: element.fieldKey, childKey: $event })
            "
            @select-field="
              $emit('select-subform-child', { parentKey: element.fieldKey, childKey: $event })
            "
          />
        </div>
      </template>
    </Draggable>
    <!-- 空状态仅负责居中展示，不进入 Sortable 排序流，保证首次拖入投影从画布顶部出现。 -->
    <div v-if="fields.length === 0" class="form-field-canvas__empty" role="status">
      从左侧选择字段或将字段拖入画布
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElIcon, ElPopconfirm } from 'element-plus';
import Draggable from 'vuedraggable';
import { CopyDocument, Delete } from '@element-plus/icons-vue';
import type {
  FormDesignDragField,
  FormDesignField,
  FormDesignFieldDefaultValue,
  FormDesignTemplateField,
} from '../types';
import FormDesignFieldControl from './FormDesignFieldControl.vue';
import FormSubformCanvas from './FormSubformCanvas.vue';

const props = defineProps<{
  fields: FormDesignField[];
  selectedFieldKey: string;
  selectedSubformChildFieldKey: string;
}>();

const t = (text: string) => text;
const emits = defineEmits<{
  (event: 'select-field', fieldKey: string): void;
  (event: 'copy-field', field: FormDesignField): void;
  (event: 'remove-field', fieldKey: string): void;
  (event: 'copy-subform-child', value: { parentKey: string; childKey: string }): void;
  (event: 'remove-subform-child', value: { parentKey: string; childKey: string }): void;
  (event: 'select-subform-child', value: { parentKey: string; childKey: string }): void;
  (event: 'add-subform-drag-field', value: FormDesignDragField & { parentKey: string }): void;
  (event: 'add-drag-field', value: FormDesignDragField): void;
  (event: 'sort-fields', fieldKeys: string[]): void;
  (event: 'update-field-default-value', fieldKey: string, value: FormDesignFieldDefaultValue): void;
}>();

// 跨列表克隆时事件下标可能与响应式数组更新不同步，需要按调色板临时字段反查真实位置。
const findPaletteCloneFieldIndex = (preferredIndex: number) => {
  if (preferredIndex >= 0 && props.fields[preferredIndex]?.fieldKey?.startsWith('palette_')) {
    return preferredIndex;
  }
  return props.fields.findIndex((field) => field.fieldKey?.startsWith('palette_'));
};

const handleDragAdd = (event: {
  newIndex?: number;
  newDraggableIndex?: number;
  item?: HTMLElement;
}) => {
  const eventIndex = event.newDraggableIndex ?? event.newIndex ?? -1;
  const cloneIndex = findPaletteCloneFieldIndex(eventIndex);
  const index = cloneIndex >= 0 ? cloneIndex : eventIndex;
  const dragField = index >= 0 ? props.fields[index] : undefined;
  const widgetName = dragField?.widgetName || event.item?.dataset?.widgetName;
  const dataType = dragField?.dataType;
  if (!widgetName || typeof dataType !== 'string') {
    // 无法识别控件时只清理调色板临时项，避免 palette_* 数据残留并进入属性面板。
    if (cloneIndex >= 0) props.fields.splice(cloneIndex, 1);
    return;
  }
  const formFieldType =
    typeof (dragField as { formFieldType?: unknown } | undefined)?.formFieldType === 'string'
      ? (dragField as { formFieldType: string }).formFieldType
      : undefined;
  emits('add-drag-field', { index, widgetName, dataType, formFieldType });
};

/** 根画布排序完成后仅上报字段键顺序，由 Schema 编辑器统一持久化。 */
const handleSortFields = () =>
  emits(
    'sort-fields',
    props.fields.map((field) => field.fieldKey),
  );

const isSubformField = (field: FormDesignField) => field.widgetName === 'subforms';

// 空白字符不作为有效显示名称，避免继续渲染无意义的名称和必填标识。
const hasFieldLabel = (field: FormDesignField) => Boolean(field.fieldLabel?.trim());

const getSubformFields = (field: FormDesignField): FormDesignTemplateField[] => {
  const fields = field.fieldConf?.fields;
  return Array.isArray(fields) ? (fields as FormDesignTemplateField[]) : [];
};
</script>

<style lang="scss">
.form-field-canvas {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 100%;
  padding: var(--gp-space-2xl);
  border-top: 1px solid var(--gp-border-color-sm);
  border-right: 1px solid var(--gp-border-color-sm);
  border-bottom: 1px solid var(--gp-border-color-sm);
  border-top-right-radius: var(--gp-radius-md);
  border-bottom-right-radius: var(--gp-radius-md);

  &__list {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 100%;
    padding-bottom: var(--gp-space-4xl);

    // 根画布统一使用紧凑的单层虚线投影，明确提示字段最终插入位置。
    & > .form-design-palette__item,
    & > .form-field-canvas__field--ghost,
    & > .sortable-ghost {
      position: relative;
      box-sizing: border-box;
      display: block;
      width: 100%;
      min-height: var(--gp-space-4xl);
      padding: 0;
      margin: 0 0 var(--gp-space-lg);
      color: transparent;
      background-color: transparent;
      border: 1px dashed var(--gp-color-primary);
      border-radius: var(--gp-radius-sm);
      box-shadow: none;
      opacity: 1;
    }

    & > .form-design-palette__item > *,
    & > .form-field-canvas__field--ghost > *,
    & > .sortable-ghost > * {
      display: none;
    }
  }

  &__empty {
    position: absolute;
    inset: var(--gp-space-2xl);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 0;
    font-size: var(--gp-text-xs);
    color: var(--gp-text-color-secondary);
    pointer-events: none;
  }

  &__empty-illustration {
    position: relative;
    width: 280px;
    height: 150px;
    margin-bottom: var(--gp-space-lg);
    background-color: var(--gp-fill-color-md);
    border: 1px solid var(--gp-border-color-xs);
    border-radius: var(--gp-radius-md);
  }

  &__empty-card {
    position: absolute;
    top: 62px;
    left: 58px;
    width: 168px;
    height: 36px;
    background-color: var(--gp-bg-color);
    border: 1px solid var(--gp-color-primary);
    border-radius: var(--gp-radius-sm);
  }

  &__empty-row {
    position: absolute;
    left: 72px;
    width: 110px;
    height: 16px;
    background-color: var(--gp-fill-color-sm);
    border-radius: var(--gp-radius-sm);

    &.is-active {
      top: 34px;
    }

    &:not(.is-active) {
      bottom: 28px;
    }
  }

  &__field {
    position: relative;
    padding: var(--gp-space-md) var(--gp-space-xl);
    margin-bottom: var(--gp-space-lg);
    cursor: grab;
    background-color: transparent;
    border: 1px solid transparent;
    border-radius: var(--gp-radius-sm);
    transition:
      background-color 0.16s ease,
      border-color 0.16s ease;

    &:active {
      cursor: grabbing;
    }

    &:hover {
      background-color: var(--gp-fill-color-sm);
    }

    &.is-active {
      background-color: var(--gp-color-primary-light-1);
      border-color: var(--gp-color-primary-light-2);
    }

    // 空名称仍为顶部操作区预留按钮高度，保持操作按钮原有位置并避免覆盖控件。
    &.is-label-empty {
      .form-field-canvas__field-header {
        min-height: var(--gp-space-3xl);
      }
    }

    &:hover .form-field-canvas__actions,
    &.is-active .form-field-canvas__actions {
      pointer-events: auto;
      opacity: 1;
    }

    &--ghost {
      background-color: var(--gp-fill-color-sm);
      border-color: var(--gp-color-primary);
      opacity: 1;
    }

    &--chosen {
      box-shadow: var(--gp-box-shadow-xs);
    }
  }

  &__field-header {
    display: flex;
    gap: var(--gp-space-xs);
    align-items: center;
    // 操作区为 24px 高，标题行须完整占位，才能与下方预览控件保留 4px 间距。
    min-height: var(--gp-space-3xl);
    padding-right: var(--gp-space-6xl);
    margin-bottom: var(--gp-space-xs);
    font-size: var(--gp-text-xs);
    font-weight: 600;
    color: var(--gp-text-color-primary);
  }

  &__field-label {
    flex: 1;
    min-width: 0;
  }

  &__field-required {
    margin-left: var(--gp-space-xs);
    color: var(--gp-color-error);
  }

  &__control {
    pointer-events: none;
  }

  .el-input,
  .el-select {
    cursor: default;
  }

  &__actions {
    position: absolute;
    top: var(--gp-space-xs);
    right: var(--gp-space-xl);
    display: flex;
    overflow: hidden;
    pointer-events: none;
    background-color: var(--gp-bg-color);
    border-radius: var(--gp-radius-sm);
    box-shadow: var(--gp-box-shadow-xs);
    opacity: 0;
    transition: opacity 0.16s ease;

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 24px;
      height: 24px;
      color: var(--gp-text-color-secondary);
      cursor: pointer;
      background-color: transparent;
      border: 0;

      & + button {
        border-left: 1px solid var(--gp-border-color-sm);
      }

      &:hover {
        color: var(--gp-text-color-primary);
        background-color: var(--gp-fill-color-sm);
      }
    }
  }
}
</style>
