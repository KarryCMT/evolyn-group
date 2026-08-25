<template>
  <div class="form-subform-table">
    <table class="form-subform-table__content">
      <thead>
        <tr>
          <th class="form-subform-table__actions-column"></th>
          <th v-for="{ column } in tableColumns" :key="column.field">
            <el-dropdown trigger="click" @command="changeColumnMode(column.field, $event)">
              <button class="form-subform-table__column-trigger" type="button">
                <span>{{ fieldMap[column.field]?.fieldLabel }}</span>
                <el-icon><RiArrowDownSFill /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="custom" :disabled="column.mode === 'custom'">
                    {{ '自定义' }}
                  </el-dropdown-item>
                  <el-dropdown-item command="depend" :disabled="column.mode === 'depend'">
                    {{ '字段值' }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, rowIndex) in rows" :key="row.rowKey">
          <td class="form-subform-table__row-actions">
            <button type="button" :title="'展开'" @click="openRow(rowIndex)">
              <el-icon><RiFullscreenFill /></el-icon>
            </button>
            <button type="button" :title="'删除'" @click="removeRow(row.rowKey)">
              <el-icon><RiDeleteBinFill /></el-icon>
            </button>
          </td>
          <td v-for="{ column, childField } in tableColumns" :key="column.field">
            <FormSubformCell
              :child-field="childField"
              :column="column"
              :row="row"
              :row-index="rowIndex"
              :model-value="getCellBinding(row, column)"
              :is-smart-assistant="isSmartAssistant"
              :selector-options="selectorOptions"
              @update:model-value="(value) => updateCell(row.rowKey, column.field, value)"
            >
              <template v-if="$slots['depend-field']" #depend-field="slotProps">
                <slot name="depend-field" v-bind="slotProps" />
              </template>
            </FormSubformCell>
          </td>
        </tr>
        <tr v-if="!rows.length">
          <td :colspan="tableColumns.length + 1" class="form-subform-table__empty">
            {{ '暂无数据' }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RiArrowDownSFill, RiDeleteBinFill, RiFullscreenFill } from '@remixicon/vue';
import type { FormDesignSelectorOptionsResolver, FormDesignTemplateField } from '../../types';
import FormSubformCell from './FormSubformCell.vue';
import type {
  FormSubformCellBinding,
  FormSubformColumnBinding,
  FormSubformColumnMode,
  FormSubformRowBinding,
} from './types';

/**
 * 插件子表单表格，负责列模式切换、行操作和横向表格渲染。
 * @property fields 子表单字段定义。
 * @property columns 子表单列绑定配置。
 * @property rows 子表单行绑定配置。
 * @property isSmartAssistant 是否使用智能助手场景的人员/部门下拉选择器。
 * @property selectorOptions 根据子字段获取智能助手人员/部门下拉选项的方法。
 */
const props = defineProps<{
  fields: FormDesignTemplateField[];
  columns: FormSubformColumnBinding[];
  rows: FormSubformRowBinding[];
  isSmartAssistant?: boolean;
  selectorOptions?: FormDesignSelectorOptionsResolver;
}>();

const emits = defineEmits<{
  (event: 'change-column-mode', fieldKey: string, mode: FormSubformColumnMode): void;
  (event: 'open-row', rowIndex: number): void;
  (event: 'remove-row', rowKey: string): void;
  (event: 'update-cell', rowKey: string, fieldKey: string, value: FormSubformCellBinding): void;
}>();

const fieldMap = computed<Record<string, FormDesignTemplateField>>(() => {
  return Object.fromEntries(props.fields.map((field) => [field.fieldKey, field]));
});

/**
 * 仅渲染仍能在当前子表单中找到定义的列，避免历史配置引用已删除字段时渲染失败。
 */
const tableColumns = computed(() => {
  return props.columns.flatMap((column) => {
    const childField = fieldMap.value[column.field];
    return childField ? [{ column, childField }] : [];
  });
});

/** 为尚未写入的单元格提供与列模式一致的临时默认值。 */
const getCellBinding = (
  row: FormSubformRowBinding,
  column: FormSubformColumnBinding,
): FormSubformCellBinding => {
  const cell = row.cells[column.field];
  if (cell) return cell;
  if (column.mode === 'depend') {
    return { linkNodeId: '', dependField: '', beforeDependField: '', dependParentKey: null };
  }
  return { sourceData: null };
};

const changeColumnMode = (fieldKey: string, mode: unknown) => {
  if (mode !== 'custom' && mode !== 'depend') return;
  emits('change-column-mode', fieldKey, mode);
};

const openRow = (rowIndex: number) => emits('open-row', rowIndex);
const removeRow = (rowKey: string) => emits('remove-row', rowKey);
const updateCell = (rowKey: string, fieldKey: string, value: FormSubformCellBinding) =>
  emits('update-cell', rowKey, fieldKey, value);
</script>

<style lang="scss" scoped>
.form-subform-table {
  width: 100%;
  overflow-x: auto;
  border: 1px solid var(--gp-border-color-sm);
  border-radius: var(--gp-radius-sm);

  &__content {
    width: max-content;
    min-width: 100%;
    border-spacing: 0;
    border-collapse: separate;

    th,
    td {
      box-sizing: border-box;
      min-width: 160px;
      height: var(--gp-space-5xl);
      padding: var(--gp-space-xs) var(--gp-space-md);
      border-right: 1px solid var(--gp-border-color-sm);
      border-bottom: 1px solid var(--gp-border-color-sm);
    }

    th {
      padding: 0;
      font-size: var(--gp-text-xs);
      font-weight: 500;
      color: var(--el-text-color-primary);
      text-align: left;
      background-color: var(--el-fill-color-light);
    }

    tr:last-child td {
      border-bottom: 0;
    }

    th:last-child,
    td:last-child {
      border-right: 0;
    }
  }

  &__actions-column,
  &__row-actions {
    position: sticky;
    left: 0;
    z-index: 1;
    min-width: 64px !important;
    width: 64px;
  }

  &__actions-column {
    z-index: 2;
  }

  &__row-actions {
    white-space: nowrap;
    background-color: var(--el-bg-color);

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: var(--gp-space-3xl);
      height: var(--gp-space-3xl);
      padding: 0;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background: transparent;
      border: 0;

      &:hover {
        color: var(--el-color-primary);
      }
    }
  }

  &__column-trigger {
    display: flex;
    gap: var(--gp-space-md);
    align-items: center;
    justify-content: space-between;
    width: 100%;
    height: var(--gp-space-5xl);
    padding: 0 var(--gp-space-lg);
    font-size: var(--gp-text-xs);
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  &__empty {
    height: var(--gp-space-6xl) !important;
    font-size: var(--gp-text-xs);
    color: var(--el-text-color-secondary);
    text-align: center;
  }
}
</style>
