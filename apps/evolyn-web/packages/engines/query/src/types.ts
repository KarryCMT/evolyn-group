/** Query DSL v1；服务端必须将其解释为参数化查询，而非透传 SQL。 */
export const QUERY_DSL_VERSION = 1 as const;

export type QueryScalar = string | number | boolean | null;
export type QueryValue = QueryScalar | readonly QueryScalar[];

export type QueryFieldType =
  | 'text'
  | 'decimal'
  | 'number'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'enum'
  | 'member'
  | 'department';

export type QueryOperator =
  | 'eq'
  | 'neq'
  | 'contains'
  | 'notContains'
  | 'startsWith'
  | 'endsWith'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'in'
  | 'notIn'
  | 'between'
  | 'isNull'
  | 'isNotNull';

export type QueryConjunction = 'and' | 'or';
export type QuerySortDirection = 'asc' | 'desc';
export type QueryAggregateOperator = 'count' | 'sum' | 'avg' | 'min' | 'max';

export interface QueryCondition {
  type: 'condition';
  /** 发布 schema 中的稳定字段标识，不得使用 label、SQL 表达式或物理列名。 */
  field: string;
  operator: QueryOperator;
  value?: QueryValue;
}

export interface QueryGroup {
  type: 'group';
  conjunction: QueryConjunction;
  children: readonly QueryExpression[];
}

export type QueryExpression = QueryCondition | QueryGroup;

export interface QuerySort {
  field: string;
  direction: QuerySortDirection;
}

export interface QueryPaging {
  page: number;
  pageSize: number;
}

export interface QueryAggregate {
  field: string;
  operator: QueryAggregateOperator;
  /** 聚合结果稳定标识，组件 encoding 只引用 alias，不引用渲染器配置。 */
  alias: string;
}

/**
 * 可安全跨网络传输的查询文档。权限约束须由后端或外部 Policy 合成，不能内嵌
 * 在此协议以免前端投影被误当作授权事实源。
 */
export interface QueryDocument {
  version: typeof QUERY_DSL_VERSION;
  filter?: QueryExpression;
  sorts: readonly QuerySort[];
  paging: QueryPaging;
  projection?: readonly string[];
  groupBy?: readonly string[];
  aggregates?: readonly QueryAggregate[];
}

/** 字段目录同时约束设计器选项和服务端逻辑计划，不能由浏览器自行扩大。 */
export interface QueryFieldCapability {
  type: QueryFieldType;
  filterable: boolean;
  sortable: boolean;
  projectable: boolean;
  groupable: boolean;
  aggregates: readonly QueryAggregateOperator[];
}

export interface QueryComplexityBudget {
  maxDepth: number;
  maxConditions: number;
  maxSorts: number;
  maxProjection: number;
  maxGroupBy: number;
  maxAggregates: number;
  maxPageSize: number;
}

export const DEFAULT_QUERY_COMPLEXITY_BUDGET: Readonly<QueryComplexityBudget> = Object.freeze({
  maxDepth: 12,
  maxConditions: 50,
  maxSorts: 8,
  maxProjection: 100,
  maxGroupBy: 8,
  maxAggregates: 16,
  maxPageSize: 100,
});

export type QueryDiagnosticCode =
  | 'QUERY_EMPTY_FIELD'
  | 'QUERY_EMPTY_GROUP'
  | 'QUERY_INVALID_EXPRESSION'
  | 'QUERY_INVALID_OPERATOR'
  | 'QUERY_OPERATOR_NOT_ALLOWED'
  | 'QUERY_INVALID_VALUE'
  | 'QUERY_INVALID_SORT'
  | 'QUERY_INVALID_PROJECTION'
  | 'QUERY_INVALID_GROUP_BY'
  | 'QUERY_INVALID_AGGREGATE'
  | 'QUERY_DUPLICATE_ALIAS'
  | 'QUERY_UNKNOWN_FIELD'
  | 'QUERY_FIELD_NOT_PROJECTABLE'
  | 'QUERY_FIELD_NOT_GROUPABLE'
  | 'QUERY_FIELD_NOT_SORTABLE'
  | 'QUERY_AGGREGATE_NOT_ALLOWED'
  | 'QUERY_INVALID_PAGING'
  | 'QUERY_COMPLEXITY_EXCEEDED'
  | 'QUERY_INVALID_VERSION'
  | 'QUERY_DOCUMENT_INVALID';

export interface QueryDiagnostic {
  code: QueryDiagnosticCode;
  message: string;
  path: string;
}

export interface QueryValidationOptions {
  /** 完整字段能力目录；提供时所有字段引用都必须命中目录并满足相应能力。 */
  fields?: Readonly<Record<string, QueryFieldCapability>>;
  /** 兼容表单列表现有调用；新代码应提供 fields。 */
  fieldTypes?: Readonly<Record<string, QueryFieldType>>;
  budget?: Partial<QueryComplexityBudget>;
}

export interface QueryValidationResult {
  document: QueryDocument | null;
  diagnostics: readonly QueryDiagnostic[];
}

export interface QueryLogicalPlan {
  filter?: QueryExpression;
  projection: readonly string[];
  groupBy: readonly string[];
  aggregates: readonly QueryAggregate[];
  sorts: readonly QuerySort[];
  paging: QueryPaging;
  aggregate: boolean;
  complexity: {
    depth: number;
    conditions: number;
  };
}

/** 字段类型到允许操作符的唯一映射，设计器和数据适配器应共享此常量。 */
export const QUERY_OPERATORS_BY_FIELD_TYPE: Readonly<
  Record<QueryFieldType, readonly QueryOperator[]>
> = {
  text: [
    'eq',
    'neq',
    'contains',
    'notContains',
    'startsWith',
    'endsWith',
    'in',
    'notIn',
    'isNull',
    'isNotNull',
  ],
  decimal: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'in', 'notIn', 'between', 'isNull', 'isNotNull'],
  number: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'in', 'notIn', 'between', 'isNull', 'isNotNull'],
  boolean: ['eq', 'neq', 'isNull', 'isNotNull'],
  date: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'between', 'isNull', 'isNotNull'],
  datetime: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'between', 'isNull', 'isNotNull'],
  enum: ['eq', 'neq', 'in', 'notIn', 'isNull', 'isNotNull'],
  member: ['eq', 'neq', 'in', 'notIn', 'isNull', 'isNotNull'],
  department: ['eq', 'neq', 'in', 'notIn', 'isNull', 'isNotNull'],
};
