<script setup lang="ts">
import { computed } from 'vue';
import { emptyWidgetValue, readWidgetOptions } from '../../../schema/codec';
import type { FormItem, FormJsonValue, SubformWidget } from '../../../schema/types';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 子表单基础行编辑器：行数据保持为不可变 JSON 对象数组，编辑操作通过顶层字段的
 * update:modelValue 上送 Runtime。P4 首批仅开放已有值语义的基础字段，避免行内
 * 出现尚无目录/上传能力的控件。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as SubformWidget);
const fields = computed(() => widget.value.items.filter((item) => item.widget.visible));
const rows = computed(() => normalizeRows(props.modelValue));
const maxRows = computed(() => widget.value.maxRowCount ?? 200);
const minRows = computed(() => widget.value.minRowCount ?? 0);
const readOnly = computed(() => props.disabled || props.readonly);
const canCreate = computed(
  () => !readOnly.value && widget.value.subformCreate && rows.value.length < maxRows.value,
);
const canInsert = computed(
  () => !readOnly.value && widget.value.subformInsert && rows.value.length < maxRows.value,
);
const canEdit = computed(() => !readOnly.value && widget.value.subformEdit);
const canDelete = computed(
  () => !readOnly.value && widget.value.subformDelete && rows.value.length > minRows.value,
);

function normalizeRows(value: FormValue): Record<string, FormJsonValue>[] {
  if (!Array.isArray(value)) return [];
  return value.filter(
    (row): row is Record<string, FormJsonValue> =>
      typeof row === 'object' && row !== null && !Array.isArray(row),
  );
}

function emptyRow(): Record<string, FormJsonValue> {
  const row: Record<string, FormJsonValue> = {};
  fields.value.forEach((field) => {
    row[field.widget.widgetName] = emptyWidgetValue(field.widget.type);
  });
  return row;
}

function emitRows(next: Record<string, FormJsonValue>[]): void {
  emit('update:modelValue', next);
  emit('blur');
}

function addRow(): void {
  if (!canCreate.value) return;
  emitRows([...rows.value, emptyRow()]);
}

function insertRow(afterIndex: number): void {
  if (!canInsert.value) return;
  const next = [...rows.value];
  next.splice(afterIndex + 1, 0, emptyRow());
  emitRows(next);
}

function removeRow(index: number): void {
  if (!canDelete.value) return;
  emitRows(rows.value.filter((_, rowIndex) => rowIndex !== index));
}

function updateCell(rowIndex: number, field: FormItem, rawValue: unknown): void {
  if (!canEdit.value || !field.widget.enable) return;
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

function cellValue(row: Record<string, FormJsonValue>, field: FormItem): FormJsonValue {
  return row[field.widget.widgetName] ?? emptyWidgetValue(field.widget.type);
}

function inputType(field: FormItem): string {
  if (field.widget.type === 'number') return 'number';
  if (field.widget.type === 'datetime') {
    return field.widget.format === 'datetime'
      ? 'datetime-local'
      : (field.widget.format ?? 'datetime-local');
  }
  return 'text';
}

function inputDisplayValue(row: Record<string, FormJsonValue>, field: FormItem): string | number {
  const value = cellValue(row, field);
  if (typeof value === 'number') return value;
  if (typeof value !== 'string') return '';
  return field.widget.type === 'datetime' && field.widget.format === 'datetime'
    ? value.replace(' ', 'T').slice(0, 16)
    : value;
}

function onInput(rowIndex: number, field: FormItem, event: Event): void {
  const raw = (event.target as HTMLInputElement | HTMLTextAreaElement).value;
  if (field.widget.type === 'datetime' && field.widget.format === 'datetime' && raw !== '') {
    updateCell(rowIndex, field, `${raw.replace('T', ' ')}:00`);
    return;
  }
  updateCell(rowIndex, field, raw);
}

function onMultiChoice(
  rowIndex: number,
  field: FormItem,
  optionValue: string,
  checked: boolean,
): void {
  const current = cellValue(rows.value[rowIndex] ?? {}, field);
  const selected = Array.isArray(current)
    ? current.filter((value): value is string => typeof value === 'string')
    : [];
  updateCell(
    rowIndex,
    field,
    checked ? [...selected, optionValue] : selected.filter((value) => value !== optionValue),
  );
}

function isSelected(
  row: Record<string, FormJsonValue>,
  field: FormItem,
  optionValue: string,
): boolean {
  const value = cellValue(row, field);
  return Array.isArray(value) && value.includes(optionValue);
}

function options(field: FormItem) {
  return readWidgetOptions(field.widget);
}

function placeholder(field: FormItem): string {
  if ('placeholder' in field.widget && typeof field.widget.placeholder === 'string') {
    return field.widget.placeholder;
  }
  return field.widget.type === 'datetime' ? '请选择' : '请输入';
}
</script>

<template>
  <section class="evf-subform" :aria-invalid="errors.length > 0 || undefined">
    <div v-if="rows.length === 0" class="evf-subform__empty">暂无明细行</div>
    <div v-else class="evf-subform__scroll">
      <table class="evf-subform__table">
        <thead class="evf-subform__head">
          <tr class="evf-subform__head-row">
            <th
              v-for="field in fields"
              :key="field.widget.widgetName"
              class="evf-subform__head-cell"
              scope="col"
            >
              <span v-if="!field.widget.allowBlank" class="evf-subform__required" aria-hidden="true"
                >*</span
              >
              {{ field.label }}
            </th>
            <th v-if="canInsert || canDelete" class="evf-subform__actions-head" scope="col">
              操作
            </th>
          </tr>
        </thead>
        <tbody class="evf-subform__body">
          <tr v-for="(row, rowIndex) in rows" :key="rowIndex" class="evf-subform__row">
            <td v-for="field in fields" :key="field.widget.widgetName" class="evf-subform__cell">
              <textarea
                v-if="field.widget.type === 'textarea'"
                class="evf-subform__input evf-subform__textarea"
                :value="inputDisplayValue(row, field)"
                :placeholder="placeholder(field)"
                :disabled="!canEdit || !field.widget.enable"
                @input="onInput(rowIndex, field, $event)"
              />
              <select
                v-else-if="field.widget.type === 'radiogroup' || field.widget.type === 'combo'"
                class="evf-subform__input"
                :value="inputDisplayValue(row, field)"
                :disabled="!canEdit || !field.widget.enable"
                @change="updateCell(rowIndex, field, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">请选择</option>
                <option v-for="option in options(field)" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
              <div
                v-else-if="
                  field.widget.type === 'checkboxgroup' || field.widget.type === 'combocheck'
                "
                class="evf-subform__choices"
              >
                <label
                  v-for="option in options(field)"
                  :key="option.value"
                  class="evf-subform__choice"
                >
                  <input
                    type="checkbox"
                    :checked="isSelected(row, field, option.value)"
                    :disabled="!canEdit || !field.widget.enable"
                    @change="
                      onMultiChoice(
                        rowIndex,
                        field,
                        option.value,
                        ($event.target as HTMLInputElement).checked,
                      )
                    "
                  />
                  <span>{{ option.label }}</span>
                </label>
              </div>
              <input
                v-else
                class="evf-subform__input"
                :type="inputType(field)"
                :value="inputDisplayValue(row, field)"
                :placeholder="placeholder(field)"
                :disabled="!canEdit || !field.widget.enable"
                @input="onInput(rowIndex, field, $event)"
              />
            </td>
            <td v-if="canInsert || canDelete" class="evf-subform__actions">
              <button
                v-if="canInsert"
                class="evf-subform__action"
                type="button"
                @click="insertRow(rowIndex)"
              >
                插入
              </button>
              <button
                v-if="canDelete"
                class="evf-subform__action evf-subform__action--danger"
                type="button"
                @click="removeRow(rowIndex)"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <button v-if="canCreate" class="evf-subform__add" type="button" @click="addRow">
      添加明细
    </button>
  </section>
</template>

<style scoped>
.evf-subform {
  display: grid;
  gap: 10px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--evf-border-color, #dcdfe6);
  border-radius: 6px;
}

.evf-subform__empty {
  padding: 12px;
  color: var(--evf-muted-color, #909399);
  text-align: center;
}

.evf-subform__scroll {
  overflow-x: auto;
}
.evf-subform__table {
  width: 100%;
  min-width: 520px;
  border-collapse: collapse;
}
.evf-subform__head-cell,
.evf-subform__cell,
.evf-subform__actions,
.evf-subform__actions-head {
  padding: 8px;
  border: 1px solid var(--evf-border-color, #ebeef5);
  text-align: left;
  vertical-align: top;
}
.evf-subform__head-cell,
.evf-subform__actions-head {
  background: var(--evf-subform-head-bg, #f5f7fa);
  font-weight: 600;
}
.evf-subform__input {
  box-sizing: border-box;
  width: 100%;
  min-width: 120px;
  min-height: 32px;
  padding: 5px 8px;
  border: 1px solid var(--evf-border-color, #dcdfe6);
  border-radius: 4px;
  background: transparent;
  color: inherit;
}
.evf-subform__textarea {
  min-height: 64px;
  resize: vertical;
}
.evf-subform__choices {
  display: grid;
  gap: 6px;
  min-width: 140px;
}
.evf-subform__choice {
  display: flex;
  gap: 6px;
  align-items: center;
}
.evf-subform__actions {
  white-space: nowrap;
}
.evf-subform__action,
.evf-subform__add {
  padding: 4px 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--evf-primary-color, #409eff);
  cursor: pointer;
}
.evf-subform__action--danger {
  color: var(--evf-danger-color, #f56c6c);
}
.evf-subform__add {
  justify-self: start;
  background: var(--evf-primary-color, #409eff);
  color: #fff;
}

@media (max-width: 640px) {
  .evf-subform {
    padding: 8px;
  }
  .evf-subform__table {
    min-width: 460px;
  }
}
</style>
