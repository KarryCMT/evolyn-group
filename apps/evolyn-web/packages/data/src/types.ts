import type { Component } from 'vue';
import type { EvolynTableColumn, EvolynTableRow } from '@evolyn.do/ui';

/** 数据工作台的稳定动作键；业务端可通过 string 扩展特定动作。 */
export type DataActionKey =
  | 'create'
  | 'import'
  | 'export'
  | 'remove'
  | 'batch'
  | 'operation-log'
  | 'recycle-bin'
  | 'filter'
  | (string & {});

/** 工具栏动作只描述展示和可用性，具体业务由消费端接收 action 事件执行。 */
export interface DataAction {
  key: DataActionKey;
  label: string;
  icon?: Component;
  disabled?: boolean;
  tone?: 'default' | 'primary' | 'danger';
}

/** 列配置沿用 UI 包的 VTable 封装，数据包不重复维护表格渲染能力。 */
export type DataColumn = EvolynTableColumn;
export type DataRecord = EvolynTableRow;

export interface DataQuery {
  keyword: string;
  page: number;
  pageSize: number;
}

export interface DataPagination {
  total: number;
  page: number;
  pageSize: number;
  pageSizes?: number[];
}

/** 业务侧从后端或本地草稿加载后的标准分页结果。 */
export interface DataPage<Record extends DataRecord = DataRecord> {
  records: Record[];
  total: number;
}

/**
 * 数据包只声明数据读写边界，不绑定 HTTP、表单 Schema 或权限实现。
 * 应用侧可按表单、聚合表、关联数据等资源分别提供 adapter。
 */
export interface DataSourceAdapter<Record extends DataRecord = DataRecord> {
  load: (query: DataQuery) => Promise<DataPage<Record>>;
  create?: (input: Record<string, unknown>) => Promise<Record>;
  remove?: (ids: string[]) => Promise<void>;
  import?: (file: File) => Promise<void>;
  export?: (query: DataQuery) => Promise<Blob>;
}

/** 数据源能力由业务端根据权限、套餐和资源类型注入，禁止在包内写死。 */
export interface DataCapabilities {
  create?: boolean;
  import?: boolean;
  export?: boolean;
  remove?: boolean;
  batch?: boolean;
  operationLog?: boolean;
  recycleBin?: boolean;
  filter?: boolean;
}
