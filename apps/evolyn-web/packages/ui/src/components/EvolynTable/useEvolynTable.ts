import { ListTable, type ListTableConstructorOptions } from '@visactor/vtable';
import {
  type Ref,
  type ShallowRef,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  toValue,
  watch,
} from 'vue';
import { EVOLYN_TABLE_EVENTS, type EvolynTableVTableEvent } from './events';
import type { EvolynTableRow } from './EvolynTable.types';

interface UseEvolynTableOptions {
  /** 表格挂载容器（模板 ref） */
  container: Ref<HTMLElement | null>;
  /** 组装完成的完整配置（不含 records，数据经 records 通道单独增量更新） */
  options: Ref<ListTableConstructorOptions>;
  /** 行数据 getter：数组引用变化即触发 setRecords */
  records: () => EvolynTableRow[] | undefined;
  /** 事件转发回调，入参为烤串事件名与原始事件参数 */
  onEvent: (name: string, args: unknown) => void;
}

/**
 * EvolynTable 的实例生命周期管理：创建、分级更新、事件绑定与销毁。
 *
 * 更新策略分级（区别于官方 vue-vtable 的一律全量 updateOption）：
 * - options 变化（列/主题/逃生舱，低频）→ updateOption 全量更新；
 * - records 引用变化（翻页/筛选后拉新数据，高频）→ setRecords 增量刷新，
 *   避免重新解析列与主题带来的整表重建开销。
 */
export function useEvolynTable({ container, options, records, onEvent }: UseEvolynTableOptions) {
  // 外部类实例按约定存 shallowRef，避免 Vue 代理 VTable 内部的大量画布对象
  const table = shallowRef<ListTable | null>(null);

  function bindEvents(instance: ListTable) {
    // 一次性绑全映射表：未监听的事件 emit 出去没有订阅者，开销可忽略
    for (const [vtableEvent, emitName] of Object.entries(EVOLYN_TABLE_EVENTS)) {
      instance.on(vtableEvent as EvolynTableVTableEvent, (args: unknown) =>
        onEvent(emitName, args),
      );
    }
  }

  function create() {
    if (!container.value || table.value) return;
    table.value = new ListTable(container.value, {
      ...toValue(options),
      records: toValue(records) ?? [],
    });
    bindEvents(table.value);
  }

  onMounted(create);

  // 结构性配置变化：全量更新，需携带当前数据避免表格被清空
  watch(options, (next) => {
    if (!table.value) {
      create();
      return;
    }
    table.value.updateOption({ ...next, records: toValue(records) ?? [] });
  });

  // 仅数据变化：走 setRecords 增量刷新
  watch(records, (next) => {
    if (!table.value) {
      create();
      return;
    }
    table.value.setRecords(next ?? []);
  });

  onBeforeUnmount(() => {
    // 释放 VTable 的事件监听与画布资源，DOM 由 Vue 负责移除
    table.value?.release();
    table.value = null;
  });

  return { table, getTable: (): ListTable | null => table.value };
}
