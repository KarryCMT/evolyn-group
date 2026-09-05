/** Query DSL v1；服务端必须将其解释为参数化查询，而非透传 SQL。 */
export const QUERY_DSL_VERSION = 1 as const;

export type QueryScalar = string | number | boolean | null;
export type QueryValue = QueryScalar | readonly QueryScalar[];

export type QueryFieldType = 'text' | 'number' | 'boolean' | 'date' | 'datetime' | 'enum';

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

export interface QueryDiagnostic {
  code:
    | 'QUERY_EMPTY_FIELD'
    | 'QUERY_EMPTY_GROUP'
    | 'QUERY_INVALID_OPERATOR'
    | 'QUERY_OPERATOR_NOT_ALLOWED'
    | 'QUERY_INVALID_VALUE'
    | 'QUERY_INVALID_SORT'
    | 'QUERY_INVALID_PROJECTION'
    | 'QUERY_INVALID_AGGREGATE'
    | 'QUERY_INVALID_VERSION';
  message: string;
  path: string;
}

export interface QueryValidationOptions {
  /** 可选的字段类型目录；未提供时只校验 DSL 自身形状。 */
  fieldTypes?: Readonly<Record<string, QueryFieldType>>;
}

export interface QueryValidationResult {
  document: QueryDocument | null;
  diagnostics: readonly QueryDiagnostic[];
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
  number: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'in', 'notIn', 'between', 'isNull', 'isNotNull'],
  boolean: ['eq', 'neq', 'isNull', 'isNotNull'],
  date: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'between', 'isNull', 'isNotNull'],
  datetime: ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'between', 'isNull', 'isNotNull'],
  enum: ['eq', 'neq', 'in', 'notIn', 'isNull', 'isNotNull'],
};
