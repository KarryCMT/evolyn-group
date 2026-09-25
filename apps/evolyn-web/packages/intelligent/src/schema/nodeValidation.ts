import type {
  FormTriggerCondition,
  IntelligentActionConfig,
  IntelligentFieldAssignment,
  IntelligentNode,
  IntelligentTrigger,
  IntelligentUpdateFilter,
} from './types';

export interface IntelligentNodeConfigurationState {
  /** 当前配置是否满足设计器的最小闭合要求。 */
  configured: boolean;
  /** 是否还没有录入任何与当前动作相关的业务配置。 */
  empty: boolean;
}

function hasText(value: string | undefined): boolean {
  return Boolean(value?.trim());
}

function hasCompleteAssignment(assignment: IntelligentFieldAssignment): boolean {
  if (!assignment.targetFieldId || !assignment.targetWidgetName) return false;
  if (assignment.source.type === 'node-field') {
    return hasText(assignment.source.nodeId) && hasText(assignment.source.field);
  }
  return assignment.source.type === 'empty' || assignment.source.value !== undefined;
}

function hasCompleteFilter(filter: IntelligentUpdateFilter): boolean {
  if (!filter.targetFieldId || !filter.targetWidgetName) return false;
  if (filter.operator === 'is-empty' || filter.operator === 'is-not-empty') return true;
  if (filter.source?.type === 'node-field') {
    return hasText(filter.source.nodeId) && hasText(filter.source.field);
  }
  return filter.source?.type === 'custom' && filter.source.value !== undefined;
}

function hasCompleteTriggerCondition(condition: FormTriggerCondition): boolean {
  if (!condition.fieldId) return false;
  if (condition.operator === 'is-empty' || condition.operator === 'is-not-empty') return true;
  return condition.values.length > 0;
}

function validateActionConfig(
  node: IntelligentNode,
  config: IntelligentActionConfig,
): IntelligentNodeConfigurationState {
  const named = hasText(node.name);
  switch (node.actionType) {
    case 'create-record': {
      const target = hasText(config.targetFormCode);
      return { configured: named && target, empty: !target && !config.fieldAssignments?.length };
    }
    case 'update-record': {
      const mode = config.updateTargetMode ?? 'form';
      const target =
        mode === 'node'
          ? hasText(config.targetNodeId) && hasText(config.targetFormCode)
          : hasText(config.targetFormCode);
      const filters =
        mode === 'node'
          ? true
          : Boolean(config.updateFilters?.length && config.updateFilters.every(hasCompleteFilter));
      const assignments = Boolean(
        config.fieldAssignments?.length && config.fieldAssignments.every(hasCompleteAssignment),
      );
      return {
        configured: named && target && filters && assignments,
        empty:
          !target &&
          !config.updateFilters?.length &&
          !config.fieldAssignments?.length,
      };
    }
    case 'delete-record': {
      // 删除节点的专用属性面板接入前，允许稳定编码或已录入的表单名称作为目标。
      const target = hasText(config.targetFormCode) || hasText(config.targetFormName);
      return { configured: named && target, empty: !target };
    }
    case 'send-notification': {
      const message = hasText(config.message);
      return { configured: named && message, empty: !message };
    }
    case 'http-request': {
      const requestUrl = hasText(config.requestUrl);
      return { configured: named && requestUrl, empty: !requestUrl };
    }
    case 'condition':
    case 'data-transform': {
      const expression = hasText(config.expression);
      return { configured: named && expression, empty: !expression };
    }
    default:
      return { configured: false, empty: true };
  }
}

function validateTrigger(trigger: IntelligentTrigger): IntelligentNodeConfigurationState {
  if (trigger.type !== 'form') {
    const configured = hasText(trigger.eventName);
    return { configured, empty: !configured };
  }
  const hasForm = hasText(trigger.formCode) || hasText(trigger.formName);
  const actionsComplete =
    trigger.actions.length > 0 &&
    trigger.actions.every(
      (action) =>
        action.type !== 'update' ||
        action.updateScope !== 'specified-fields' ||
        Boolean(action.fieldIds?.length),
    );
  const conditionsComplete = trigger.conditions.every(hasCompleteTriggerCondition);
  return {
    configured: hasForm && actionsComplete && conditionsComplete,
    empty: !hasForm && trigger.actions.length === 0 && trigger.conditions.length === 0,
  };
}

/**
 * 统一计算画布节点的配置状态。属性面板仍负责更细粒度的输入提示，这里只决定
 * 节点卡片是否需要呈现发布校验失败态，避免各画布组件各自猜测配置完整性。
 */
export function getIntelligentNodeConfigurationState(
  node: IntelligentNode,
  trigger: IntelligentTrigger,
): IntelligentNodeConfigurationState {
  if (node.type === 'end') return { configured: true, empty: false };
  if (node.type === 'trigger') return validateTrigger(trigger);
  return validateActionConfig(node, node.config ?? {});
}
