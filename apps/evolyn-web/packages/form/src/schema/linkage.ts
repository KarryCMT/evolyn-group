import type { DataLinkageDefinition, DataLinkageOperator, FormItem, FormWidgetType } from './types';

export interface LinkageSourceOption {
  sourceId: string;
  appId: number;
  name: string;
}

export interface LinkageSourceField {
  id: string;
  name: string;
  type: FormWidgetType;
  operators: DataLinkageOperator[];
}

/** 设计器通过宿主注入 API；form package 不感知认证、租户或请求地址。 */
export interface LinkageDesignerAdapter {
  listSources(signal: AbortSignal): Promise<LinkageSourceOption[]>;
  listSourceFields(sourceId: string, signal: AbortSignal): Promise<LinkageSourceField[]>;
}

export const DATA_LINKAGE_OPERATOR_LABELS: Readonly<Record<DataLinkageOperator, string>> = {
  eq: '等于',
  neq: '不等于',
  gt: '大于',
  gte: '大于等于',
  lt: '小于',
  lte: '小于等于',
  contains: '包含',
  notContains: '不包含',
  in: '等于任意一个',
  not_in: '不等于任意一个',
  empty: '为空',
  not_empty: '不为空',
};

export function createDataLinkageId(): string {
  return `linkage_${randomToken(12)}`;
}

export function createDataLinkageConditionId(): string {
  return `cond_${randomToken(10)}`;
}

function randomToken(length: number): string {
  const alphabet = '0123456789abcdefghijklmnopqrstuvwxyz';
  const bytes = new Uint32Array(length);
  crypto.getRandomValues(bytes);
  return [...bytes].map((value) => alphabet[value % alphabet.length]).join('');
}

export function createDataLinkageDefinition(
  appId: number,
  targetFieldId: string,
): DataLinkageDefinition {
  return {
    id: createDataLinkageId(),
    version: 1,
    enabled: true,
    source: { type: 'form', appId, sourceId: '' },
    filter: {
      logic: 'and',
      conditions: [
        {
          id: createDataLinkageConditionId(),
          sourceFieldId: '',
          operator: 'eq',
          value: { type: 'field', fieldId: targetFieldId },
        },
      ],
    },
    mappings: [{ sourceFieldId: '', targetFieldId }],
    result: { mode: 'first' },
    runtime: {
      trigger: 'dependency_change',
      runOnInit: true,
      debounceMs: 250,
      emptyStrategy: 'clear',
      errorStrategy: 'keep',
    },
  };
}

export function linkageRuleForTarget(
  rules: readonly DataLinkageDefinition[],
  targetFieldId: string,
): DataLinkageDefinition | undefined {
  return rules.find((rule) =>
    rule.mappings.some((mapping) => mapping.targetFieldId === targetFieldId),
  );
}

/**
 * 联动协议是 JSON 数据，但设计器传入的数组通常是 Vue Proxy，不能直接交给
 * structuredClone。显式复制各层既规避 DataCloneError，也避免弹窗草稿回写源 Schema。
 */
export function cloneDataLinkageDefinition(
  rule: DataLinkageDefinition,
): DataLinkageDefinition {
  return {
    ...rule,
    source: { ...rule.source },
    filter: {
      ...rule.filter,
      conditions: rule.filter.conditions.map((condition) => ({
        ...condition,
        value: condition.value ? { ...condition.value } : undefined,
      })),
    },
    mappings: rule.mappings.map((mapping) => ({ ...mapping })),
    result: {
      ...rule.result,
      orderBy: rule.result.orderBy?.map((order) => ({ ...order })),
    },
    runtime: { ...rule.runtime },
  };
}

export function cloneDataLinkageDefinitions(
  rules: readonly DataLinkageDefinition[],
): DataLinkageDefinition[] {
  return rules.map(cloneDataLinkageDefinition);
}

/** V1 严格类型兼容：不做隐式转换；等价数值家族允许共享 NUMERIC 值协议。 */
export function isLinkageFieldCompatible(
  sourceType: FormWidgetType,
  targetType: FormWidgetType,
): boolean {
  if (sourceType === targetType) return true;
  const numeric = new Set<FormWidgetType>(['number', 'decimal', 'money', 'percent']);
  return numeric.has(sourceType) && numeric.has(targetType);
}

export function linkageCurrentFields(items: readonly FormItem[]): LinkageSourceField[] {
  return items
    .filter((item) => !['separator', 'button', 'subform'].includes(item.widget.type))
    .map((item) => ({
      id: item.widget.widgetName,
      name: item.label,
      type: item.widget.type,
      operators: [],
    }));
}
