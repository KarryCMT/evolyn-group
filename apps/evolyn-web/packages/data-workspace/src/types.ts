import type { Component } from 'vue';
import type { DataQuery } from '@evolyn.do/data';

/** 数据工作台的展示动作；图标与视觉色调只属于 Vue 展示层。 */
export interface DataAction {
  key: string;
  label: string;
  icon?: Component;
  disabled?: boolean;
  tone?: 'default' | 'primary' | 'danger';
}

/**
 * 复用 UI 表格的列投影，避免数据引擎耦合任一渲染库。icon 是列设置面板
 * 行内的字段类型图标（工作台展示元信息），渲染前由 DataWorkspace 剥离，
 * 不透传 VTable。
 */
export interface DataColumnLeaf {
  field: string;
  title: string;
  /** 列设置面板行内展示的字段类型图标（Vue 组件，建议 markRaw 传入）。 */
  icon?: Component;
  /** VTable 原生叶子列配置由工作台原样透传。 */
  [key: string]: any;
}

/** 数据工作台分组列：只用于层级表头，不参与列设置的直接勾选。 */
export interface DataColumnGroup {
  title: string;
  columns: DataColumn[];
  /** VTable 原生分组列配置由工作台原样透传。 */
  [key: string]: any;
}

export type DataColumn = DataColumnLeaf | DataColumnGroup;

/** 数据管理记录的稳定选择键；当前表单记录为数值 ID，同时保留字符串兼容性。 */
export type DataRecordId = string | number;

export interface DataColumnSettingItem {
  column: DataColumnLeaf;
  /** 含父级表头的展示路径，例如「订单明细 / 金额」。 */
  titlePath: string;
}

export function isDataColumnGroup(column: DataColumn): column is DataColumnGroup {
  return 'columns' in column;
}

/**
 * 列设置始终以可取值的叶子列为粒度；分组本身随其可见子项自动出现或隐藏。
 */
export function flattenDataColumns(
  columns: readonly DataColumn[],
  parentTitles: readonly string[] = [],
): DataColumnSettingItem[] {
  return columns.flatMap((column) => {
    if (isDataColumnGroup(column)) {
      return flattenDataColumns(column.columns, [...parentTitles, column.title]);
    }
    return [{ column, titlePath: [...parentTitles, column.title].join(' / ') }];
  });
}

export interface DataPagination {
  total: number;
  page: number;
  pageSize: number;
  pageSizes?: readonly number[];
}

export interface UseDataWorkspaceOptions {
  initialQuery?: Partial<DataQuery>;
}
