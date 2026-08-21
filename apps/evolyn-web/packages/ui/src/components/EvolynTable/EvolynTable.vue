<script setup lang="ts" generic="Row extends EvolynTableRow = EvolynTableRow">
import type { ColumnDefine, ListTableConstructorOptions } from '@visactor/vtable';
import { computed, useTemplateRef } from 'vue';
import type {
  EvolynTableColumn,
  EvolynTableEmits,
  EvolynTableOptions,
  EvolynTableRow,
} from './EvolynTable.types';
import { createElementTheme } from './theme';
import { useEvolynTable } from './useEvolynTable';

defineOptions({ name: 'EvolynTable' });

const props = withDefaults(
  defineProps<{
    columns: EvolynTableColumn[];
    records?: Row[];
    options?: EvolynTableOptions;
    width?: number | string;
    height?: number | string;
    theme?: 'light' | 'dark';
    emptyText?: string;
  }>(),
  {
    width: '100%',
    height: '100%',
    theme: 'light',
    emptyText: '暂无数据',
  },
);

const emit = defineEmits<EvolynTableEmits>();

const container = useTemplateRef<HTMLElement>('container');

/** 业务列定义 → VTable 列定义：显式字段做名字归一化，其余经 ...rest 原样透传 */
function normalizeColumn(column: EvolynTableColumn): ColumnDefine {
  const {
    field,
    title,
    width,
    minWidth,
    align,
    sortable,
    cellType,
    format,
    customRender,
    style,
    ...rest
  } = column;

  const define: ColumnDefine = { field, title, ...rest };
  if (width !== undefined) define.width = width;
  if (minWidth !== undefined) define.minWidth = minWidth;
  if (cellType !== undefined) define.cellType = cellType;
  if (customRender !== undefined) define.customRender = customRender;
  if (format !== undefined) {
    // format 是 formatter 的窄化别名：去掉 table 实例入参，降低使用方心智
    define.formatter = (record, col, row) => format(record, col, row);
  }
  if (sortable) define.sort = true;
  if (align !== undefined || style !== undefined) {
    // 对齐方式并入列样式，逃生舱传入的原生 style 优先级更低
    define.style = {
      ...(style as Record<string, unknown> | undefined),
      ...(align !== undefined ? { textAlign: align } : {}),
    } as ColumnDefine['style'];
  }
  return define;
}

const normalizedColumns = computed<ColumnDefine[]>(() => props.columns.map(normalizeColumn));

const tableOptions = computed<ListTableConstructorOptions>(() => ({
  // 平台默认值：列宽自适应撑满容器；空态文案对齐简道云。可被 options 逃生舱覆盖
  widthMode: 'adaptive',
  emptyTip: { text: props.emptyText },
  ...props.options,
  // columns/theme 固定由组件管理，放在 options 之后避免被逃生舱覆盖
  columns: normalizedColumns.value,
  theme: createElementTheme(props.theme),
}));

/** 容器尺寸：数字按像素处理，字符串原样交给 CSS */
const containerStyle = computed(() => ({
  width: typeof props.width === 'number' ? `${props.width}px` : props.width,
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
}));

const { getTable } = useEvolynTable({
  container,
  options: tableOptions,
  records: () => props.records,
  // 动态事件名分发：两档负载由 emits 重载分别约束，运行时按事件名字符串直达监听器，
  // as never 只是让联合名通过重载检查，不影响运行时行为
  onEvent: (name, args) => emit(name as never, args as never),
});

defineExpose({ getTable });
</script>

<template>
  <div ref="container" class="evolyn-table" :style="containerStyle" />
</template>

<style lang="scss">
@use './EvolynTable.scss' as *;
</style>
