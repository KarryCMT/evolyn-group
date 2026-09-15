import type { DataRecord } from '@evolyn.do/data';
import { computed, shallowRef, watch, type ComputedRef } from 'vue';
import type { DataColumnLeaf, DataRecordId } from '../types.js';

/** 表格展示私有字段，不参与查询、筛选和列设置。 */
export const DATA_WORKSPACE_SELECTION_FIELD = '__evolyn_record_selection';

interface UseDataWorkspaceSelectionOptions {
  /** 当前页表格行；子表单展开时，同一父记录可能出现多次。 */
  records: ComputedRef<readonly DataRecord[]>;
  onChange?: (ids: DataRecordId[]) => void;
}

interface CheckboxStateChangeEvent {
  field?: unknown;
  checked?: unknown;
  originData?: unknown;
}

/** VTable 回调实际需要的最小表格能力，避免组合式逻辑依赖具体表格实例类型。 */
interface RecordLookupTable {
  getRecordByCell(col: number, row: number): unknown;
}

/**
 * 数据工作台首列复选框状态。选择以父记录 ID 为唯一事实源，因此子表单展开的
 * 多行明细只算作一条可批量操作的记录。
 */
export function useDataWorkspaceSelection(options: UseDataWorkspaceSelectionOptions) {
  const selectedRecordIds = shallowRef<ReadonlySet<DataRecordId>>(new Set());
  const selectableRecordIds = computed(() => {
    const ids = new Set<DataRecordId>();
    for (const record of options.records.value) {
      const id = recordIdOf(record);
      if (id !== null) ids.add(id);
    }
    return [...ids];
  });
  const allRowsSelected = computed(
    () =>
      selectableRecordIds.value.length > 0 &&
      selectableRecordIds.value.every((id) => selectedRecordIds.value.has(id)),
  );

  // 切换分页、筛选或刷新后，不保留已离开当前数据集的选择项。
  watch(selectableRecordIds, (recordIds) => {
    const available = new Set(recordIds);
    const next = new Set([...selectedRecordIds.value].filter((id) => available.has(id)));
    if (next.size !== selectedRecordIds.value.size) setSelectedRecordIds(next);
  });

  const selectionColumn = computed<DataColumnLeaf>(() => {
    // 捕获当前快照以建立 Vue 依赖；selectionColumn 随选择态换引用，外部表格
    // 才会执行 updateOption 重绘表头与单元格的 checkbox 状态。
    const selected = selectedRecordIds.value;
    const headerChecked = allRowsSelected.value;
    return {
      field: DATA_WORKSPACE_SELECTION_FIELD,
      title: '',
      width: 72,
      minWidth: 72,
      headerType: 'checkbox',
      cellType: 'checkbox',
      checked: (context: { col: number; row: number; table: RecordLookupTable }) => {
        const id = recordIdOf(context.table.getRecordByCell(context.col, context.row));
        return id === null ? headerChecked : selected.has(id);
      },
      // 同一父记录的子表单明细行共享一个勾选框。
      mergeCell: mergeSameRecordRows,
    };
  });

  function setSelectedRecordIds(next: ReadonlySet<DataRecordId>) {
    selectedRecordIds.value = new Set(next);
    options.onChange?.([...selectedRecordIds.value]);
  }

  /** 仅合并同一父记录的选择框，防止相邻普通记录的复选框被错误合并。 */
  function mergeSameRecordRows(
    _sourceValue: unknown,
    _targetValue: unknown,
    context: {
      source: { col: number; row: number };
      target: { col: number; row: number };
      table: RecordLookupTable;
    },
  ): boolean {
    const sourceId = recordIdOf(
      context.table.getRecordByCell(context.source.col, context.source.row),
    );
    const targetId = recordIdOf(
      context.table.getRecordByCell(context.target.col, context.target.row),
    );
    return sourceId !== null && sourceId === targetId;
  }

  function handleCheckboxStateChange(event: unknown) {
    const change = event as CheckboxStateChangeEvent;
    if (change.field !== DATA_WORKSPACE_SELECTION_FIELD || typeof change.checked !== 'boolean')
      return;

    const recordId = recordIdOf(change.originData);
    const next = new Set(selectedRecordIds.value);
    if (recordId === null) {
      for (const id of selectableRecordIds.value) {
        if (change.checked) next.add(id);
        else next.delete(id);
      }
    } else if (change.checked) {
      next.add(recordId);
    } else {
      next.delete(recordId);
    }
    setSelectedRecordIds(next);
  }

  function clearSelection() {
    if (selectedRecordIds.value.size === 0) return;
    setSelectedRecordIds(new Set());
  }

  return {
    selectionColumn,
    handleCheckboxStateChange,
    clearSelection,
  };
}

function recordIdOf(record: unknown): DataRecordId | null {
  if (!record || typeof record !== 'object') return null;
  const id = (record as DataRecord).id;
  return typeof id === 'string' || typeof id === 'number' ? id : null;
}
