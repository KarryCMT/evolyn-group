<script setup lang="ts">
import { ElAlert, ElButton, ElDialog, ElIcon } from 'element-plus';
import { RiAddLine, RiDeleteBin6Line, RiFileCopyLine } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import { emptyWidgetValue, validateWidgetValue } from '../../schema/codec';
import type { FormItem, FormJsonValue, SubformWidget } from '../../schema/types';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';
import WebSubformCellEditor from './WebSubformCellEditor.vue';
import WebSubformTable, { type SubformRow, type SubformRowCommand } from './WebSubformTable.vue';

type RowValidationErrors = Record<number, Record<string, readonly string[]>>;

/**
 * Web 子表单协调器：统一管理值、行级命令、选择与弹层状态；表格与单元格编辑被拆分为
 * 独立组件，未来增加汇总、审批态或服务端分页时无需改动现有编辑器。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as SubformWidget);
const fields = computed(() => widget.value.items.filter((item) => item.widget.visible));
const rows = computed(() => normalizeRows(props.modelValue));
const hasFields = computed(() => fields.value.length > 0);
const maxRows = computed(() => widget.value.maxRowCount ?? 200);
const minRows = computed(() => widget.value.minRowCount ?? 0);
const readOnly = computed(() => props.disabled || props.readonly);
const canCreate = computed(
  () =>
    hasFields.value &&
    !readOnly.value &&
    widget.value.subformCreate &&
    rows.value.length < maxRows.value,
);
const canInsert = computed(
  () => !readOnly.value && widget.value.subformInsert && rows.value.length < maxRows.value,
);
const canEdit = computed(() => !readOnly.value && widget.value.subformEdit);
const canDelete = computed(
  () => !readOnly.value && widget.value.subformDelete && rows.value.length > minRows.value,
);
const canQuickFill = computed(
  () => canCreate.value && canEdit.value && widget.value.quickFill && rows.value.length > 0,
);
const pcStickyColumnLimit = computed(() => {
  const config = widget.value.pcStickyColumn;
  if (!config?.enable) return 0;
  return Math.min(fields.value.length, Math.max(0, config.limit));
});

const batchMode = shallowRef(false);
const selectedRowIndexes = shallowRef<number[]>([]);
const activeEditorRowIndex = shallowRef<number | null>(null);
const fullscreenOpen = shallowRef(false);
const isRowEditorOpen = computed({
  get: () => activeEditorRowIndex.value !== null,
  set: (open: boolean) => {
    if (!open) activeEditorRowIndex.value = null;
  },
});
const activeRow = computed<SubformRow | null>(() => {
  const index = activeEditorRowIndex.value;
  return index === null ? null : (rows.value[index] ?? null);
});
const selectedRowCount = computed(() => selectedRowIndexes.value.length);
const canDeleteSelected = computed(
  () =>
    canDelete.value &&
    selectedRowCount.value > 0 &&
    rows.value.length - selectedRowCount.value >= minRows.value,
);
const rowValidationErrors = computed<RowValidationErrors>(() => {
  // 表单外壳仅在失焦/提交后提供 errors；此时按同一 schema 校验器定位到单元格，
  // 既不复制校验规则，也能让服务端与客户端的文案保持一致。
  if (props.errors.length === 0) return {};
  return rows.value.reduce<RowValidationErrors>((allRows, row, rowIndex) => {
    const fieldErrors = fields.value.reduce<Record<string, readonly string[]>>(
      (allFields, field) => {
        const errors = validateWidgetValue(field, cellValue(row, field));
        if (errors.length > 0) allFields[field.widget.widgetName] = errors;
        return allFields;
      },
      {},
    );
    if (Object.keys(fieldErrors).length > 0) allRows[rowIndex] = fieldErrors;
    return allRows;
  }, {});
});
const validationMessage = computed(() => props.errors[0] ?? '');

watch(
  rows,
  (nextRows) => {
    // 行没有稳定 ID，结构操作后索引会变化，必须一次性收敛选中态和弹层目标，避免误删。
    selectedRowIndexes.value = selectedRowIndexes.value.filter((index) => index < nextRows.length);
    if (activeEditorRowIndex.value !== null && activeEditorRowIndex.value >= nextRows.length) {
      activeEditorRowIndex.value = null;
    }
  },
  { deep: false },
);

function normalizeRows(value: FormValue): SubformRow[] {
  if (!Array.isArray(value)) return [];
  return value.filter(
    (row): row is SubformRow => typeof row === 'object' && row !== null && !Array.isArray(row),
  );
}

function emptyRow(): SubformRow {
  const row: SubformRow = {};
  fields.value.forEach((field) => {
    row[field.widget.widgetName] = emptyWidgetValue(field.widget.type);
  });
  return row;
}

function cloneRow(row: SubformRow): SubformRow {
  // 子表单协议只允许 JSON 值；深拷贝防止未来的数组/对象型控件与源行共享引用。
  return JSON.parse(JSON.stringify(row)) as SubformRow;
}

function emitRows(next: SubformRow[]): void {
  emit('update:modelValue', next);
  emit('blur');
}

function addRow(): void {
  if (!canCreate.value) return;
  emitRows([...rows.value, emptyRow()]);
}

function quickFillRow(): void {
  const lastRow = rows.value[rows.value.length - 1];
  if (!canQuickFill.value || !lastRow) return;
  emitRows([...rows.value, cloneRow(lastRow)]);
}

function insertEmptyRow(index: number): void {
  if (!canInsert.value) return;
  const next = [...rows.value];
  next.splice(index, 0, emptyRow());
  emitRows(next);
}

function copyRow(index: number, destination: 'next' | 'last'): void {
  const source = rows.value[index];
  if (!canInsert.value || !source) return;
  const next = [...rows.value];
  if (destination === 'next') next.splice(index + 1, 0, cloneRow(source));
  else next.push(cloneRow(source));
  emitRows(next);
}

function removeRow(index: number): void {
  if (!canDelete.value || !rows.value[index]) return;
  emitRows(rows.value.filter((_, rowIndex) => rowIndex !== index));
}

function deleteSelectedRows(): void {
  if (!canDeleteSelected.value) return;
  const selected = new Set(selectedRowIndexes.value);
  selectedRowIndexes.value = [];
  emitRows(rows.value.filter((_, rowIndex) => !selected.has(rowIndex)));
}

function updateCell(rowIndex: number, field: FormItem, rawValue: unknown): void {
  if (!canEdit.value || !field.widget.enable || !rows.value[rowIndex]) return;
  const next = rows.value.map((row, index) =>
    index === rowIndex
      ? { ...row, [field.widget.widgetName]: normalizeCellValue(field, rawValue) }
      : row,
  );
  emitRows(next);
}

function normalizeCellValue(field: FormItem, value: unknown): FormJsonValue {
  switch (field.widget.type) {
    case 'number':
      if (typeof value === 'number' && Number.isFinite(value)) return value;
      return typeof value === 'string' && value.trim() !== '' && Number.isFinite(Number(value))
        ? Number(value)
        : null;
    case 'checkboxgroup':
    case 'combocheck':
      return Array.isArray(value)
        ? value.filter((entry): entry is string => typeof entry === 'string')
        : [];
    default:
      return typeof value === 'string' && value !== '' ? value : null;
  }
}

function cellValue(row: SubformRow, field: FormItem): FormJsonValue {
  return row[field.widget.widgetName] ?? emptyWidgetValue(field.widget.type);
}

function handleRowCommand(command: SubformRowCommand, rowIndex: number): void {
  switch (command) {
    case 'copy-next':
      copyRow(rowIndex, 'next');
      break;
    case 'copy-last':
      copyRow(rowIndex, 'last');
      break;
    case 'insert-above':
      insertEmptyRow(rowIndex);
      break;
    case 'insert-below':
      insertEmptyRow(rowIndex + 1);
      break;
  }
}

function toggleBatchMode(): void {
  batchMode.value = !batchMode.value;
  if (!batchMode.value) selectedRowIndexes.value = [];
}
</script>

<template>
  <section
    class="evf-web-subform"
    :class="{ 'evf-web-subform--empty-schema': !hasFields }"
    :aria-invalid="errors.length > 0 || undefined"
  >
    <div v-if="!hasFields" class="evf-web-subform__empty">暂未配置子字段</div>
    <template v-else>
      <ElAlert
        v-if="validationMessage"
        class="evf-web-subform__validation"
        type="warning"
        show-icon
        :closable="false"
        :title="validationMessage"
      />
      <WebSubformTable
        :fields="fields"
        :rows="rows"
        :disabled="disabled"
        :readonly="readonly"
        :can-edit="canEdit"
        :can-insert="canInsert"
        :can-delete="canDelete"
        :sticky-field-count="pcStickyColumnLimit"
        :batch-mode="batchMode"
        :selected-row-indexes="selectedRowIndexes"
        :validation-errors="rowValidationErrors"
        @cell-change="updateCell"
        @cell-blur="emit('blur')"
        @open-row-editor="activeEditorRowIndex = $event"
        @remove-row="removeRow"
        @row-command="handleRowCommand"
        @change-selection="selectedRowIndexes = $event"
        @toggle-batch-mode="toggleBatchMode"
        @open-fullscreen="fullscreenOpen = true"
      />

      <div class="evf-web-subform__toolbar">
        <template v-if="batchMode">
          <ElButton text data-action="cancel-batch" @click="toggleBatchMode">取消选择</ElButton>
          <ElButton
            type="danger"
            text
            :disabled="!canDeleteSelected"
            data-action="delete-selected"
            @click="deleteSelectedRows"
          >
            <ElIcon><RiDeleteBin6Line /></ElIcon>删除选中{{
              selectedRowCount ? ` (${selectedRowCount})` : ''
            }}
          </ElButton>
        </template>
        <template v-else>
          <ElButton v-if="canCreate" type="primary" plain data-action="add" @click="addRow">
            <ElIcon><RiAddLine /></ElIcon>添加
          </ElButton>
          <ElButton
            v-if="canQuickFill"
            type="primary"
            text
            data-action="quick-fill"
            @click="quickFillRow"
          >
            <ElIcon><RiFileCopyLine /></ElIcon>快速填报
          </ElButton>
        </template>
      </div>
    </template>

    <ElDialog
      v-model="isRowEditorOpen"
      :title="activeEditorRowIndex === null ? '编辑明细' : `编辑第 ${activeEditorRowIndex + 1} 行`"
      width="min(920px, calc(100vw - 32px))"
      append-to-body
      destroy-on-close
      class="evf-web-subform__row-dialog"
    >
      <div v-if="activeRow" class="evf-web-subform__row-editor">
        <label
          v-for="field in fields"
          :key="field.widget.widgetName"
          class="evf-web-subform__row-field"
        >
          <span class="evf-web-subform__row-label">
            <span v-if="!field.widget.allowBlank" class="evf-web-subform__required">*</span
            >{{ field.label }}
          </span>
          <WebSubformCellEditor
            :field="field"
            :model-value="cellValue(activeRow, field)"
            :input-id="`evf-subform-dialog-${activeEditorRowIndex}-${field.widget.widgetName}`"
            :disabled="disabled || !canEdit"
            :readonly="readonly"
            :invalid="
              Boolean(rowValidationErrors[activeEditorRowIndex!]?.[field.widget.widgetName]?.length)
            "
            @update:model-value="updateCell(activeEditorRowIndex!, field, $event)"
            @blur="emit('blur')"
          />
        </label>
      </div>
    </ElDialog>

    <ElDialog
      v-model="fullscreenOpen"
      :title="`${item.label} · 全屏编辑`"
      fullscreen
      append-to-body
      class="evf-web-subform__fullscreen-dialog"
    >
      <WebSubformTable
        :fields="fields"
        :rows="rows"
        :disabled="disabled"
        :readonly="readonly"
        :can-edit="canEdit"
        :can-insert="canInsert"
        :can-delete="canDelete"
        :sticky-field-count="pcStickyColumnLimit"
        :batch-mode="batchMode"
        :selected-row-indexes="selectedRowIndexes"
        :validation-errors="rowValidationErrors"
        @cell-change="updateCell"
        @cell-blur="emit('blur')"
        @open-row-editor="activeEditorRowIndex = $event"
        @remove-row="removeRow"
        @row-command="handleRowCommand"
        @change-selection="selectedRowIndexes = $event"
        @toggle-batch-mode="toggleBatchMode"
      />
    </ElDialog>
  </section>
</template>

<style scoped lang="scss">
.evf-web-subform {
  display: grid;
  gap: var(--el-space-lg);
  width: 100%;

  &__toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: var(--el-space-sm);
    align-items: center;

    .el-button + .el-button {
      margin-left: 0;
    }
  }

  &__required {
    margin-right: 2px;
    color: var(--el-color-danger);
  }

  &__empty {
    padding: var(--el-space-xl);
    color: var(--el-text-color-secondary);
    text-align: center;
    background: var(--el-fill-color-lighter);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }

  &__row-editor {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-xl) var(--el-space-lg);
  }

  &__row-field {
    display: grid;
    gap: var(--el-space-sm);
    min-width: 0;
  }

  &__row-label {
    color: var(--el-text-color-primary);
    font-weight: var(--el-font-weight-primary);
  }
}

@media (width <= 768px) {
  .evf-web-subform__row-editor {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
