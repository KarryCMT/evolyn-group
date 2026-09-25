import type {
  IntelligentActionConfig,
  IntelligentFieldOption,
  IntelligentFieldValueSource,
  IntelligentUpdateFilter,
  IntelligentUpdateFilterOperator,
} from './types';

export const intelligentUpdateOperatorOptions: readonly {
  value: IntelligentUpdateFilterOperator;
  label: string;
}[] = [
  { value: 'equals', label: '等于' },
  { value: 'not-equals', label: '不等于' },
  { value: 'equals-any', label: '等于任意一个' },
  { value: 'not-equals-any', label: '不等于任意一个' },
  { value: 'is-empty', label: '为空' },
  { value: 'is-not-empty', label: '不为空' },
];

export function createIntelligentUpdateFilter(): IntelligentUpdateFilter {
  return {
    id: `update_filter_${globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2)}`,
    targetFieldId: '',
    targetWidgetName: '',
    operator: 'equals',
  };
}

export function isIntelligentUnaryUpdateOperator(
  operator: IntelligentUpdateFilterOperator,
): boolean {
  return operator === 'is-empty' || operator === 'is-not-empty';
}

export interface IntelligentUpdateValidation {
  target: boolean;
  filters: boolean;
  assignments: boolean;
  complete: boolean;
}

function hasCompleteValueSource(
  source: IntelligentUpdateFilter['source'] | IntelligentFieldValueSource,
): boolean {
  return source?.type !== 'node-field' || Boolean(source.nodeId && source.field);
}

/**
 * 设计期校验只判断前端文档是否闭合；运行时仍需以发布快照重新校验字段权限与类型。
 */
export function validateIntelligentUpdateConfig(
  config: IntelligentActionConfig | undefined,
  fields: readonly IntelligentFieldOption[],
): IntelligentUpdateValidation {
  const fieldIds = new Set(fields.map((field) => field.fieldId));
  const mode = config?.updateTargetMode ?? 'form';
  const target =
    mode === 'node'
      ? Boolean(config?.targetNodeId && config.targetFormCode)
      : Boolean(config?.targetFormCode);
  const filters =
    mode === 'node'
      ? true
      : Boolean(
          config?.updateFilters?.length &&
          config.updateFilters.every(
            (filter) =>
              fieldIds.has(filter.targetFieldId) &&
              (isIntelligentUnaryUpdateOperator(filter.operator) ||
                Boolean(filter.source && hasCompleteValueSource(filter.source))),
          ),
        );
  const configuredTargets = new Set<string>();
  const assignments = Boolean(
    config?.fieldAssignments?.length &&
    config.fieldAssignments.every((assignment) => {
      if (
        !fieldIds.has(assignment.targetFieldId) ||
        configuredTargets.has(assignment.targetFieldId)
      ) {
        return false;
      }
      configuredTargets.add(assignment.targetFieldId);
      return hasCompleteValueSource(assignment.source);
    }),
  );
  return { target, filters, assignments, complete: target && filters && assignments };
}
