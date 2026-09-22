import { FORMULA_FUNCTION_BY_NAME, type FormulaNode, parseFormula } from '@evolyn.do/formula';
import type { FieldFormulaDefinition, FormItem } from './types';

export const FIELD_FORMULA_LIMITS = {
  maxRules: 200,
  formulaMaxLength: 4000,
  remarkMaxLength: 500,
} as const;

/** V1 只开放前后端都能确定性执行的函数，目录展示不得超出此集合。 */
export const FIELD_FORMULA_FUNCTIONS = new Set([
  'AND',
  'OR',
  'NOT',
  'IF',
  'TRUE',
  'FALSE',
  'ISBLANK',
  'ISEMPTY',
  'CONCATENATE',
  'LEN',
  'LOWER',
  'UPPER',
  'TRIM',
  'ABS',
  'ROUND',
  'SUM',
  'AVERAGE',
  'MIN',
  'MAX',
]);

export function fieldFormulaForTarget(
  formulas: readonly FieldFormulaDefinition[],
  targetFieldId: string,
): FieldFormulaDefinition | undefined {
  return formulas.find((formula) => formula.targetFieldId === targetFieldId);
}

/** AST 顺序提取稳定字段引用，既用于运行时依赖图，也用于发布期循环检测。 */
export function formulaDependencies(source: string): string[] {
  const { ast } = parseFormula(source);
  if (!ast) return [];
  const result: string[] = [];
  const seen = new Set<string>();
  visitFormulaNode(ast, (field) => {
    if (seen.has(field)) return;
    seen.add(field);
    result.push(field);
  });
  return result;
}

export function formulaFunctionNames(source: string): string[] {
  const { ast } = parseFormula(source);
  if (!ast) return [];
  const names: string[] = [];
  const seen = new Set<string>();
  visitFormulaCalls(ast, (name) => {
    if (seen.has(name)) return;
    seen.add(name);
    names.push(name);
  });
  return names;
}

/** 返回公式目标的拓扑序；循环时返回参与循环的目标字段。 */
export function sortFieldFormulas(
  formulas: readonly FieldFormulaDefinition[],
): { ordered: FieldFormulaDefinition[]; cycle: string[] } {
  const enabled = formulas.filter((formula) => formula.enabled);
  const byTarget = new Map(enabled.map((formula) => [formula.targetFieldId, formula]));
  const indegree = new Map(enabled.map((formula) => [formula.targetFieldId, 0]));
  const outgoing = new Map<string, string[]>();
  for (const formula of enabled) {
    for (const dependency of formulaDependencies(formula.formula)) {
      if (!byTarget.has(dependency)) continue;
      indegree.set(formula.targetFieldId, (indegree.get(formula.targetFieldId) ?? 0) + 1);
      const targets = outgoing.get(dependency) ?? [];
      targets.push(formula.targetFieldId);
      outgoing.set(dependency, targets);
    }
  }
  const queue = enabled
    .map((formula) => formula.targetFieldId)
    .filter((target) => indegree.get(target) === 0);
  const ordered: FieldFormulaDefinition[] = [];
  while (queue.length > 0) {
    const target = queue.shift()!;
    ordered.push(byTarget.get(target)!);
    for (const downstream of outgoing.get(target) ?? []) {
      const next = (indegree.get(downstream) ?? 0) - 1;
      indegree.set(downstream, next);
      if (next === 0) queue.push(downstream);
    }
  }
  return {
    ordered,
    cycle: enabled
      .map((formula) => formula.targetFieldId)
      .filter((target) => (indegree.get(target) ?? 0) > 0),
  };
}

export function formulaTargetType(items: readonly FormItem[], target: string): string | undefined {
  return items.find((item) => item.widget.widgetName === target)?.widget.type;
}

export function isFieldFormulaFunctionSupported(name: string): boolean {
  return FIELD_FORMULA_FUNCTIONS.has(name) && FORMULA_FUNCTION_BY_NAME.has(name);
}

function visitFormulaNode(node: FormulaNode, onField: (field: string) => void): void {
  if (node.kind === 'field') return onField(node.widgetName);
  if (node.kind === 'array') return node.elements.forEach((entry) => visitFormulaNode(entry, onField));
  if (node.kind === 'unary') return visitFormulaNode(node.argument, onField);
  if (node.kind === 'binary') {
    visitFormulaNode(node.left, onField);
    visitFormulaNode(node.right, onField);
    return;
  }
  if (node.kind === 'call') node.args.forEach((entry) => visitFormulaNode(entry, onField));
}

function visitFormulaCalls(node: FormulaNode, onCall: (name: string) => void): void {
  if (node.kind === 'array') return node.elements.forEach((entry) => visitFormulaCalls(entry, onCall));
  if (node.kind === 'unary') return visitFormulaCalls(node.argument, onCall);
  if (node.kind === 'binary') {
    visitFormulaCalls(node.left, onCall);
    visitFormulaCalls(node.right, onCall);
    return;
  }
  if (node.kind === 'call') {
    onCall(node.name);
    node.args.forEach((entry) => visitFormulaCalls(entry, onCall));
  }
}
