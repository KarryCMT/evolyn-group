import type {
  QueryAggregate,
  QueryExpression,
  QuerySort,
} from '@evolyn.do/query';

/** 记录仅要求是键值对象；表格、表单或流程可各自投影其展示模型。 */
export type DataRecord = Record<string, unknown>;

/**
 * Data Engine 将列表交互的关键字、分页与 Query DSL 组合为一次数据访问请求。
 * Query DSL 的 AST/校验权威仍在 @evolyn.do/query，禁止在此重复定义。
 */
export interface DataQuery {
  keyword: string;
  page: number;
  pageSize: number;
  filter?: QueryExpression;
  sorts?: readonly QuerySort[];
  projection?: readonly string[];
  groupBy?: readonly string[];
  aggregates?: readonly QueryAggregate[];
}

export interface DataPagination {
  total: number;
  page: number;
  pageSize: number;
  pageSizes?: number[];
}

/** 业务侧从后端或本地草稿加载后的标准分页结果。 */
export interface DataPage<Row extends DataRecord = DataRecord> {
  records: readonly Row[];
  total: number;
}

/**
 * 数据访问上下文由领域侧提供，数据引擎不读取路由、会话或 UI 状态。
 * metadata 用于在不扩展通用协议的情况下携带版本、租户等受控上下文。
 */
export interface DataContext {
  resource: string;
  metadata?: Readonly<Record<string, unknown>>;
}

/** 数据源只描述能力；HTTP、缓存和权限策略由平台适配器在外层实现。 */
export interface DataSource<Row extends DataRecord = DataRecord> {
  load: (context: DataContext, query: DataQuery) => Promise<DataPage<Row>>;
  create?: (context: DataContext, input: Record<string, unknown>) => Promise<Row>;
  update?: (
    context: DataContext,
    id: string,
    input: Record<string, unknown>,
  ) => Promise<Row>;
  remove?: (context: DataContext, ids: readonly string[]) => Promise<void>;
  save?: (context: DataContext, record: Row) => Promise<Row>;
  refresh?: (context: DataContext) => Promise<void>;
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

/** 统一的写操作信封，供日志、队列或适配器在无需了解 UI 的场景下消费。 */
export type DataMutation<Row extends DataRecord = DataRecord> =
  | { type: 'create'; input: Record<string, unknown> }
  | { type: 'update'; id: string; input: Record<string, unknown> }
  | { type: 'remove'; ids: readonly string[] }
  | { type: 'save'; record: Row }
  | { type: 'refresh' };
