import type { CompiledRuleGraph, RuleGraphEvaluationContext, RuleGraphRule } from './types.js';

/**
 * 将声明式规则编译成稳定的依赖图。引擎不解释 Condition 内容；调用方只需提供
 * 字段依赖与目标字段，即可获得拓扑顺序、全量求值和增量重算闭包。
 */
export function compileRuleGraph<Condition>(
  sourceRules: readonly RuleGraphRule<Condition>[],
): CompiledRuleGraph<Condition> {
  const rules: RuleGraphRule<Condition>[] = [];
  const ownerRuleId = new Map<string, string>();
  const dependents = new Map<string, Set<string>>();

  for (const source of sourceRules) {
    const rule = normalizeRule(source);
    if (!rule) continue;
    rules.push(rule);
    for (const target of rule.targets) {
      // 同一目标仅由首条规则拥有；领域校验器应在发布前把冲突报告为 Diagnostic。
      if (!ownerRuleId.has(target)) ownerRuleId.set(target, rule.id);
    }
    for (const condition of rule.conditions) {
      const field = readConditionField(condition);
      if (!field) continue;
      let targets = dependents.get(field);
      if (!targets) {
        targets = new Set<string>();
        dependents.set(field, targets);
      }
      for (const target of rule.targets) targets.add(target);
    }
  }

  return {
    rules,
    ownerRuleId,
    dependents,
    topologicalOrder: topologicalSort(ownerRuleId.keys(), dependents),
  };
}

export function isEmptyCompiledRuleGraph(graph: CompiledRuleGraph): boolean {
  return graph.ownerRuleId.size === 0;
}

/**
 * 按依赖拓扑序计算所有规则目标。环不进入拓扑序，因此调用方可以将未求值字段
 * 按 fail-closed 策略处理；是否允许环由领域校验器决定。
 */
export function evaluateRuleGraph<Condition>(
  graph: CompiledRuleGraph<Condition>,
  context: RuleGraphEvaluationContext<Condition>,
): Map<string, boolean> {
  const result = new Map<string, boolean>();
  if (isEmptyCompiledRuleGraph(graph)) return result;

  const visible = new Map<string, boolean>();
  const rulesByID = new Map(graph.rules.map((rule) => [rule.id, rule]));
  for (const field of graph.topologicalOrder) {
    const ruleID = graph.ownerRuleId.get(field);
    if (!ruleID) continue;
    const rule = rulesByID.get(ruleID);
    const matched = rule
      ? context.matchRule(rule, {
          isFieldVisible: (name) => context.isBaseVisible(name) && (visible.get(name) ?? true),
        })
      : true;
    result.set(field, matched);
    visible.set(field, matched);
  }
  return result;
}

/** 返回受字段变更影响的规则目标，顺序与完整求值的拓扑序一致。 */
export function downstreamRuleTargets(
  graph: CompiledRuleGraph,
  changedField: string,
): readonly string[] {
  const firstLevel = graph.dependents.get(changedField);
  if (!firstLevel?.size) return [];

  const order = new Map(graph.topologicalOrder.map((field, index) => [field, index]));
  const visited = new Set<string>([changedField]);
  const queue = [...firstLevel];
  const affected: string[] = [];
  while (queue.length > 0) {
    const field = queue.shift()!;
    if (visited.has(field)) continue;
    visited.add(field);
    if (graph.ownerRuleId.has(field)) affected.push(field);
    queue.push(...(graph.dependents.get(field) ?? []));
  }
  return affected.sort((left, right) => (order.get(left) ?? 0) - (order.get(right) ?? 0));
}

function normalizeRule<Condition>(
  source: RuleGraphRule<Condition>,
): RuleGraphRule<Condition> | null {
  if (!source || typeof source.id !== 'string' || source.id === '') return null;
  const conditions = Array.isArray(source.conditions)
    ? source.conditions.filter((condition) => Boolean(readConditionField(condition)))
    : [];
  const targets = Array.isArray(source.targets)
    ? source.targets.filter(
        (target): target is string => typeof target === 'string' && target !== '',
      )
    : [];
  if (conditions.length === 0 || targets.length === 0) return null;
  return { id: source.id, rel: source.rel === 'or' ? 'or' : 'and', conditions, targets };
}

function readConditionField(condition: unknown): string | undefined {
  if (!condition || typeof condition !== 'object') return undefined;
  const field = (condition as { field?: unknown }).field;
  return typeof field === 'string' && field !== '' ? field : undefined;
}

function topologicalSort(
  targets: Iterable<string>,
  dependents: ReadonlyMap<string, ReadonlySet<string>>,
): string[] {
  const nodes = new Set(targets);
  for (const [source, children] of dependents) {
    nodes.add(source);
    for (const child of children) nodes.add(child);
  }
  const indegree = new Map([...nodes].map((node) => [node, 0]));
  for (const children of dependents.values()) {
    for (const child of children) indegree.set(child, (indegree.get(child) ?? 0) + 1);
  }
  const queue = [...indegree].filter(([, degree]) => degree === 0).map(([node]) => node);
  const order: string[] = [];
  while (queue.length > 0) {
    const node = queue.shift()!;
    order.push(node);
    for (const child of dependents.get(node) ?? []) {
      const degree = (indegree.get(child) ?? 0) - 1;
      indegree.set(child, degree);
      if (degree === 0) queue.push(child);
    }
  }
  return order;
}
