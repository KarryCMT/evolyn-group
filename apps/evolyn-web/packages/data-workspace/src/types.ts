import type { Component } from 'vue';
import type { EvolynTableColumn } from '@evolyn.do/ui';
import type { DataQuery } from '@evolyn.do/data';

/** 数据工作台的展示动作；图标与视觉色调只属于 Vue 展示层。 */
export interface DataAction {
  key: string;
  label: string;
  icon?: Component;
  disabled?: boolean;
  tone?: 'default' | 'primary' | 'danger';
}

/** 复用 UI 表格的列投影，避免数据引擎耦合任一渲染库。 */
export type DataColumn = EvolynTableColumn;

export interface DataPagination {
  total: number;
  page: number;
  pageSize: number;
  pageSizes?: readonly number[];
}

export interface UseDataWorkspaceOptions {
  initialQuery?: Partial<DataQuery>;
}
