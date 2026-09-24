import type {
  IntelligentFieldAssignment,
  IntelligentFieldOption,
  IntelligentFieldValueKind,
  IntelligentSourceFieldGroup,
} from './types';

const KIND_FAMILIES: Readonly<Record<IntelligentFieldValueKind, readonly IntelligentFieldValueKind[]>> = {
  text: ['text', 'choice'],
  number: ['number'],
  date: ['date'],
  choice: ['choice', 'text'],
  'multi-choice': ['multi-choice'],
  member: ['member'],
  members: ['members', 'member'],
  department: ['department'],
  departments: ['departments', 'department'],
  address: ['address', 'text'],
};

/** 字段来源只做无损或业务含义明确的赋值，避免设计期静默类型转换。 */
export function isIntelligentFieldCompatible(
  target: IntelligentFieldOption,
  source: IntelligentFieldOption,
): boolean {
  return KIND_FAMILIES[target.valueKind].includes(source.valueKind);
}

function normalizedFieldText(value: string): string {
  return value.trim().toLocaleLowerCase().replace(/[\s_\-—]/g, '');
}

/** 推荐分只用于设计器排序；零分字段仍可在兼容字段目录中手动选择。 */
export function intelligentFieldRecommendationScore(
  target: IntelligentFieldOption,
  source: IntelligentFieldOption,
): number {
  if (!isIntelligentFieldCompatible(target, source)) return 0;
  if (target.fieldId && target.fieldId === source.fieldId) return 100;
  if (target.widgetName === source.widgetName) return 95;
  if (normalizedFieldText(target.label) === normalizedFieldText(source.label)) return 90;
  return 0;
}

interface QuickFillResult {
  assignments: IntelligentFieldAssignment[];
  filledCount: number;
}

/**
 * 快捷填充仅补齐未配置字段，并保持来源字段一对一使用。整个结果由调用方一次提交，
 * 因而在设计器历史中表现为一个可撤销操作。
 */
export function quickFillIntelligentAssignments(
  targets: readonly IntelligentFieldOption[],
  sourceGroups: readonly IntelligentSourceFieldGroup[],
  current: readonly IntelligentFieldAssignment[],
): QuickFillResult {
  const next = current.map((assignment) => ({
    ...assignment,
    source: { ...assignment.source },
  }));
  const configuredTargets = new Set(next.map((assignment) => assignment.targetFieldId));
  const usedSources = new Set(
    next.flatMap((assignment) =>
      assignment.source.type === 'node-field'
        ? [`${assignment.source.nodeId}:${assignment.source.field}`]
        : [],
    ),
  );
  let filledCount = 0;

  for (const target of targets) {
    if (configuredTargets.has(target.fieldId)) continue;
    const candidates = sourceGroups
      .flatMap((group) =>
        group.fields.map((field) => ({
          group,
          field,
          score: intelligentFieldRecommendationScore(target, field),
        })),
      )
      .filter(
        (candidate) =>
          candidate.score > 0 &&
          !usedSources.has(`${candidate.group.nodeId}:${candidate.field.widgetName}`),
      )
      .sort((left, right) => right.score - left.score);
    const best = candidates[0];
    if (!best || candidates[1]?.score === best.score) continue;

    next.push({
      targetFieldId: target.fieldId,
      targetWidgetName: target.widgetName,
      source: {
        type: 'node-field',
        nodeId: best.group.nodeId,
        field: best.field.widgetName,
      },
    });
    usedSources.add(`${best.group.nodeId}:${best.field.widgetName}`);
    filledCount += 1;
  }

  return { assignments: next, filledCount };
}
