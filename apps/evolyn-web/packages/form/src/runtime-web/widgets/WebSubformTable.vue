<script setup lang="ts">
import {
  ElButton,
  ElCheckbox,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElIcon,
  ElTable,
  ElTableColumn,
  ElTooltip,
} from 'element-plus';
import {
  RiArrowDownLine,
  RiArrowUpLine,
  RiCheckboxMultipleLine,
  RiDeleteBin6Line,
  RiEditLine,
  RiFileCopyLine,
  RiFullscreenLine,
  RiMore2Fill,
} from '@remixicon/vue';
import { computed } from 'vue';
import type { FormItem, FormJsonValue } from '../../schema/types';
import WebSubformCellEditor from './WebSubformCellEditor.vue';

export type SubformRow = Record<string, FormJsonValue>;
export type SubformRowCommand = 'copy-next' | 'copy-last' | 'insert-above' | 'insert-below';

const props = withDefaults(
  defineProps<{
    fields: readonly FormItem[];
    rows: readonly SubformRow[];
    disabled: boolean;
    readonly: boolean;
    canEdit: boolean;
    canInsert: boolean;
    canDelete: boolean;
    batchMode?: boolean;
    selectedRowIndexes?: readonly number[];
    validationErrors?: Readonly<Record<number, Readonly<Record<string, readonly string[]>>>>;
  }>(),
  {
    batchMode: false,
    selectedRowIndexes: () => [],
    validationErrors: () => ({}),
  },
);

const emit = defineEmits<{
  cellChange: [rowIndex: number, field: FormItem, value: unknown];
  cellBlur: [];
  openRowEditor: [rowIndex: number];
  removeRow: [rowIndex: number];
  rowCommand: [command: SubformRowCommand, rowIndex: number];
  changeSelection: [rowIndexes: number[]];
  toggleBatchMode: [];
  openFullscreen: [];
}>();

// Element Plus 的运行时不会改写 data；该断言避免为适配其可变数组声明而复制整张表。
const tableRows = computed(() => props.rows as SubformRow[]);
const selectedRowIndexSet = computed(() => new Set(props.selectedRowIndexes));
const allRowsSelected = computed(
  () =>
    props.rows.length > 0 && props.rows.every((_, index) => selectedRowIndexSet.value.has(index)),
);
const selectionIndeterminate = computed(
  () => selectedRowIndexSet.value.size > 0 && !allRowsSelected.value,
);
const fieldValidationMessages = computed(() => {
  const messages = new Map<string, string>();
  for (const rowErrors of Object.values(props.validationErrors)) {
    for (const [fieldKey, errors] of Object.entries(rowErrors)) {
      if (!messages.has(fieldKey) && errors[0]) messages.set(fieldKey, errors[0]);
    }
  }
  return messages;
});

function cellValue(row: SubformRow, field: FormItem): FormJsonValue {
  return row[field.widget.widgetName] ?? null;
}

/** 日期时间和可多选控件有更高的内在宽度；给列留出下限，让表格横向滚动而不是控件越界。 */
function fieldMinWidth(field: FormItem): number {
  switch (field.widget.type) {
    case 'datetime':
      return 240;
    case 'combo':
    case 'combocheck':
      return 200;
    case 'textarea':
      return 200;
    default:
      return 160;
  }
}

function isInvalid(rowIndex: number, field: FormItem): boolean {
  return Boolean(props.validationErrors[rowIndex]?.[field.widget.widgetName]?.length);
}

function cellError(rowIndex: number, field: FormItem): string {
  return props.validationErrors[rowIndex]?.[field.widget.widgetName]?.[0] ?? '';
}

function fieldError(field: FormItem): string {
  return fieldValidationMessages.value.get(field.widget.widgetName) ?? '';
}

function setRowSelected(rowIndex: number, selected: boolean): void {
  const next = new Set(selectedRowIndexSet.value);
  if (selected) next.add(rowIndex);
  else next.delete(rowIndex);
  emit(
    'changeSelection',
    [...next].sort((left, right) => left - right),
  );
}

function setAllRowsSelected(selected: boolean): void {
  emit('changeSelection', selected ? props.rows.map((_, index) => index) : []);
}

function onRowCommand(command: SubformRowCommand, rowIndex: number): void {
  emit('rowCommand', command, rowIndex);
}
</script>

<template>
  <ElTable :data="tableRows" border class="evf-web-subform-table" empty-text="暂无明细行">
    <!-- 仅操作列固定在最左侧，所有业务字段始终随表格横向滚动。 -->
    <!-- 三个小型图标按钮的紧凑宽度，避免空白操作列挤占字段展示空间。 -->
    <ElTableColumn label="操作" :width="78" fixed="left" align="center">
      <template #header>
        <div class="evf-web-subform-table__operation-head">
          <div class="evf-web-subform-table__header-actions">
            <ElTooltip content="全屏编辑" placement="top">
              <ElButton
                circle
                text
                size="small"
                aria-label="全屏编辑子表单"
                data-action="fullscreen"
                @click="emit('openFullscreen')"
              >
                <ElIcon><RiFullscreenLine /></ElIcon>
              </ElButton>
            </ElTooltip>
            <ElTooltip v-if="canDelete" content="批量删除" placement="top">
              <ElButton
                circle
                text
                size="small"
                aria-label="批量删除子表单行"
                data-action="batch"
                @click="emit('toggleBatchMode')"
              >
                <ElIcon><RiCheckboxMultipleLine /></ElIcon>
              </ElButton>
            </ElTooltip>
          </div>
        </div>
      </template>
      <template #default="{ $index }">
        <div class="evf-web-subform-table__row-actions">
          <ElTooltip content="展开行编辑" placement="top">
            <ElButton
              circle
              text
              size="small"
              aria-label="编辑当前行"
              @click="emit('openRowEditor', $index)"
            >
              <ElIcon><RiEditLine /></ElIcon>
            </ElButton>
          </ElTooltip>
          <ElTooltip v-if="canDelete" content="删除当前行" placement="top">
            <ElButton
              circle
              text
              type="danger"
              size="small"
              aria-label="删除当前行"
              @click="emit('removeRow', $index)"
            >
              <ElIcon><RiDeleteBin6Line /></ElIcon>
            </ElButton>
          </ElTooltip>
          <ElDropdown v-if="canInsert" trigger="click" @command="onRowCommand($event, $index)">
            <ElButton circle text size="small" aria-label="更多行操作">
              <ElIcon><RiMore2Fill /></ElIcon>
            </ElButton>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="copy-next">
                  <ElIcon><RiFileCopyLine /></ElIcon>复制到下一行
                </ElDropdownItem>
                <ElDropdownItem command="copy-last">
                  <ElIcon><RiFileCopyLine /></ElIcon>复制到最后一行
                </ElDropdownItem>
                <ElDropdownItem command="insert-above">
                  <ElIcon><RiArrowUpLine /></ElIcon>向上插入行
                </ElDropdownItem>
                <ElDropdownItem command="insert-below">
                  <ElIcon><RiArrowDownLine /></ElIcon>向下插入行
                </ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </div>
      </template>
    </ElTableColumn>
    <ElTableColumn v-if="batchMode" :width="56" align="center">
      <template #header>
        <ElCheckbox
          :model-value="allRowsSelected"
          :indeterminate="selectionIndeterminate"
          aria-label="选择全部明细"
          @update:model-value="setAllRowsSelected(Boolean($event))"
        />
      </template>
      <template #default="{ $index }">
        <ElCheckbox
          :model-value="selectedRowIndexSet.has($index)"
          :aria-label="`选择第 ${$index + 1} 行`"
          @update:model-value="setRowSelected($index, Boolean($event))"
        />
      </template>
    </ElTableColumn>
    <ElTableColumn
      v-for="field in fields"
      :key="field.widget.widgetName"
      :min-width="fieldMinWidth(field)"
    >
      <template #header>
        <div class="evf-web-subform-table__column-header">
          <span class="evf-web-subform-table__column-label">
            <span v-if="!field.widget.allowBlank" class="evf-web-subform-table__required">*</span>
            {{ field.label }}
          </span>
          <span v-if="fieldError(field)" class="evf-web-subform-table__field-error">
            {{ fieldError(field) }}
          </span>
        </div>
      </template>
      <template #default="{ row, $index }">
        <ElTooltip
          :disabled="!cellError($index, field)"
          :content="cellError($index, field)"
          placement="top"
        >
          <div class="evf-web-subform-table__cell">
            <WebSubformCellEditor
              :field="field"
              :model-value="cellValue(row, field)"
              :input-id="`evf-subform-${$index}-${field.widget.widgetName}`"
              :disabled="disabled || !canEdit"
              :readonly="readonly"
              :invalid="isInvalid($index, field)"
              @update:model-value="emit('cellChange', $index, field, $event)"
              @blur="emit('cellBlur')"
            />
          </div>
        </ElTooltip>
      </template>
    </ElTableColumn>
  </ElTable>
</template>

<style scoped lang="scss">
.evf-web-subform-table {
  width: 100%;

  // Element Plus 在横向滚动时会移除最后一个左固定单元格的右边框，改用阴影提示。
  // 子表单的操作列只有 78px，阴影在深色主题中不够清晰，因此始终保留主题边框。
  :deep(.el-table__cell.el-table-fixed-column--left.is-last-column) {
    border-right: var(--el-table-border) !important;
    box-shadow: 1px 0 0 var(--el-table-border-color);
  }

  &__operation-head,
  &__header-actions,
  &__row-actions,
  &__column-label,
  &__column-header {
    display: inline-flex;
    gap: var(--el-space-xs);
    align-items: center;
  }

  &__operation-head {
    justify-content: center;
    width: 100%;
  }

  &__header-actions,
  &__row-actions {
    opacity: 0;
    transition: opacity var(--el-transition-duration-fast);
  }

  &__row-actions {
    gap: 0;
    justify-content: center;
  }

  &__required {
    color: var(--el-color-danger);
  }

  &__column-header {
    display: grid;
    gap: 2px;
  }

  &__field-error {
    overflow: hidden;
    color: var(--el-color-danger);
    font-size: var(--el-font-size-extra-small);
    font-weight: var(--el-font-weight-primary);
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__cell {
    display: flex;
    min-width: 0;
    overflow: hidden;

    // Element Plus 的日期、下拉与数字控件自带默认宽度；子表单列窄于默认值时，
    // 必须以表格单元格为边界收缩，避免输入值覆盖相邻列。
    :deep(.el-date-editor),
    :deep(.el-select),
    :deep(.el-input-number),
    :deep(.el-input),
    :deep(.el-textarea) {
      flex: 1 1 auto;
      width: 100%;
      min-width: 0;
      max-width: 100%;
    }

    :deep(.el-date-editor .el-input__wrapper),
    :deep(.el-date-editor .el-input__inner),
    :deep(.el-select__wrapper),
    :deep(.el-input__wrapper) {
      min-width: 0;
    }

    :deep(.el-date-editor .el-input__inner) {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  :deep(.el-table__header-wrapper:hover) &__header-actions,
  :deep(.el-table__row:hover) &__row-actions {
    opacity: 1;
  }

  :deep(.el-input.is-error .el-input__wrapper),
  :deep(.el-textarea.is-error .el-textarea__inner),
  :deep(.el-select.is-error .el-select__wrapper),
  :deep(.el-input-number.is-error .el-input__wrapper),
  :deep(.el-date-editor.is-error .el-input__wrapper) {
    box-shadow: 0 0 0 1px var(--el-color-danger) inset;
  }

  :deep(.form-department-selection.is-error .el-select__wrapper) {
    box-shadow: 0 0 0 1px var(--el-color-danger) inset;
  }
}
</style>
