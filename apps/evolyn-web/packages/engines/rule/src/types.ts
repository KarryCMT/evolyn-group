/**
 * 规则的领域无关结构。条件的具体语义由调用方通过 `matchRule` 注入，
 * 这样 Form、Workflow 和 Dashboard 可以共享依赖图而不共享彼此的 DSL。
 */
export interface RuleGraphRule<Condition = unknown> {
  id: string;
  rel: 'and' | 'or';
  conditions: readonly Condition[];
  targets: readonly string[];
}

export interface CompiledRuleGraph<Condition = unknown> {
  rules: readonly RuleGraphRule<Condition>[];
  ownerRuleId: ReadonlyMap<string, string>;
  dependents: ReadonlyMap<string, ReadonlySet<string>>;
  topologicalOrder: readonly string[];
}

export interface RuleGraphMatchContext {
  isFieldVisible: (field: string) => boolean;
}

export interface RuleGraphEvaluationContext<Condition = unknown> {
  isBaseVisible: (field: string) => boolean;
  matchRule: (rule: RuleGraphRule<Condition>, context: RuleGraphMatchContext) => boolean;
}
