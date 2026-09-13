import type { DataRecord } from '@evolyn.do/data';
import { computed, nextTick, shallowRef } from 'vue';
import { describe, expect, it } from 'vitest';
import {
  DATA_WORKSPACE_SELECTION_FIELD,
  useDataWorkspaceSelection,
} from '../useDataWorkspaceSelection';

describe('useDataWorkspaceSelection', () => {
  it('uses one checkbox column and deduplicates expanded subform rows by parent id', async () => {
    const records = shallowRef<DataRecord[]>([
      { id: 7, name: '订单 A' },
      { id: 7, name: '订单 A' },
      { id: 8, name: '订单 B' },
    ]);
    const changes: Array<Array<string | number>> = [];
    const selection = useDataWorkspaceSelection({
      records: computed(() => records.value),
      onChange: (ids) => changes.push(ids),
    });

    expect(selection.selectionColumn.value).toMatchObject({
      field: DATA_WORKSPACE_SELECTION_FIELD,
      headerType: 'checkbox',
      cellType: 'checkbox',
    });

    selection.handleCheckboxStateChange({
      field: DATA_WORKSPACE_SELECTION_FIELD,
      checked: true,
      originData: { id: 7 },
    });
    expect(changes[changes.length - 1]).toEqual([7]);
    expect(
      selection.selectionColumn.value.checked({
        col: 0,
        row: 0,
        table: { getRecordByCell: () => ({ id: 7 }) },
      }),
    ).toBe(true);

    // header event 没有 originData，选择当前页所有父记录而非三条展开行。
    selection.handleCheckboxStateChange({
      field: DATA_WORKSPACE_SELECTION_FIELD,
      checked: true,
    });
    expect(changes[changes.length - 1]).toEqual([7, 8]);

    records.value = [{ id: 8, name: '订单 B' }];
    await nextTick();
    expect(changes[changes.length - 1]).toEqual([8]);

    selection.clearSelection();
    expect(changes[changes.length - 1]).toEqual([]);
  });

  it('merges checkbox cells only when adjacent rows share a parent id', () => {
    const records = shallowRef<DataRecord[]>([]);
    const selection = useDataWorkspaceSelection({ records: computed(() => records.value) });
    const table = {
      getRecordByCell: (_col: number, row: number) => [{ id: 7 }, { id: 7 }, { id: 8 }][row],
    };

    expect(
      selection.selectionColumn.value.mergeCell(undefined, undefined, {
        source: { col: 0, row: 0 },
        target: { col: 0, row: 1 },
        table,
      }),
    ).toBe(true);
    expect(
      selection.selectionColumn.value.mergeCell(undefined, undefined, {
        source: { col: 0, row: 1 },
        target: { col: 0, row: 2 },
        table,
      }),
    ).toBe(false);
  });
});
