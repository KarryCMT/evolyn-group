import { FORMULA_FUNCTION_BY_NAME } from './catalog';
import { parseFormula, type FormulaNode } from './parser';
import type { FormulaDiagnostic, FormulaEditorField, FormulaEditorFunction } from './types';

/**
 * 编辑期严格分析：语法、字段引用、函数收录与参数个数均按目录验证。
 * 不调用上下文函数，也不在浏览器模拟服务端时间/IP，避免产生伪权威的提交结果。
 */
export function collectFormulaDiagnostics(
  formula: string,
  fields: readonly FormulaEditorField[],
  functions: readonly FormulaEditorFunction[] = [],
): FormulaDiagnostic[] {
  const parsed = parseFormula(formula);
  const diagnostics = [...parsed.diagnostics];
  if (!parsed.ast) return diagnostics;

  const fieldsByName = new Map(fields.map((field) => [field.widgetName, field]));
  const functionCatalog = new Map(
    (functions.length > 0 ? functions : [...FORMULA_FUNCTION_BY_NAME.values()]).map((item) => [
      item.name,
      item,
    ]),
  );
  analyzeNode(parsed.ast, fieldsByName, functionCatalog, diagnostics);
  analyzeMoneyCurrencyCompatibility(parsed.ast, fieldsByName, diagnostics);
  return diagnostics;
}

const CURRENCY_SENSITIVE_BINARY_OPERATORS = new Set([
  '+',
  '-',
  '*',
  '/',
  '%',
  '>',
  '>=',
  '<',
  '<=',
  '==',
  '!=',
]);
const CURRENCY_SENSITIVE_FUNCTIONS = new Set([
  'SUM',
  'AVERAGE',
  'MIN',
  'MAX',
  'PRODUCT',
  'SUMIF',
  'SUMPRODUCT',
]);

/**
 * 货币不是无量纲数字。仅对真正把多个值放进同一个算式/聚合/比较的节点检查，
 * 因而 `AND($usd# > 0, $cny# > 0)` 仍可表达两个独立条件；而 `$usd# + $cny#`
 * 会被拒绝。返回值表示子树已报告错误，防止外层表达式重复报同一问题。
 */
function analyzeMoneyCurrencyCompatibility(
  node: FormulaNode,
  fields: ReadonlyMap<string, FormulaEditorField>,
  diagnostics: FormulaDiagnostic[],
): boolean {
  if (node.kind === 'array') {
    return node.elements.some((entry) =>
      analyzeMoneyCurrencyCompatibility(entry, fields, diagnostics),
    );
  }
  if (node.kind === 'unary')
    return analyzeMoneyCurrencyCompatibility(node.argument, fields, diagnostics);
  if (node.kind === 'binary') {
    const childReported =
      analyzeMoneyCurrencyCompatibility(node.left, fields, diagnostics) ||
      analyzeMoneyCurrencyCompatibility(node.right, fields, diagnostics);
    if (childReported || !CURRENCY_SENSITIVE_BINARY_OPERATORS.has(node.operator))
      return childReported;
    return reportMixedMoneyCurrencies(node, fields, diagnostics);
  }
  if (node.kind === 'call') {
    const childReported = node.args.some((entry) =>
      analyzeMoneyCurrencyCompatibility(entry, fields, diagnostics),
    );
    if (childReported || !CURRENCY_SENSITIVE_FUNCTIONS.has(node.name)) return childReported;
    return reportMixedMoneyCurrencies(node, fields, diagnostics);
  }
  return false;
}

function reportMixedMoneyCurrencies(
  node: FormulaNode,
  fields: ReadonlyMap<string, FormulaEditorField>,
  diagnostics: FormulaDiagnostic[],
): boolean {
  const currencies = moneyCurrenciesIn(node, fields);
  if (currencies.size < 2) return false;
  diagnostics.push({
    from: node.from,
    to: node.to,
    severity: 'error',
    message: `金额字段币种不一致（${[...currencies].sort().join('、')}），跨币种计算或比较需要先换汇`,
  });
  return true;
}

function moneyCurrenciesIn(
  node: FormulaNode,
  fields: ReadonlyMap<string, FormulaEditorField>,
): Set<string> {
  const currencies = new Set<string>();
  collectMoneyCurrencies(node, fields, currencies);
  return currencies;
}

function collectMoneyCurrencies(
  node: FormulaNode,
  fields: ReadonlyMap<string, FormulaEditorField>,
  currencies: Set<string>,
): void {
  if (node.kind === 'field') {
    const currency = fields.get(node.widgetName)?.currencyCode;
    if (currency) currencies.add(currency);
    return;
  }
  if (node.kind === 'array')
    return node.elements.forEach((entry) => collectMoneyCurrencies(entry, fields, currencies));
  if (node.kind === 'unary') return collectMoneyCurrencies(node.argument, fields, currencies);
  if (node.kind === 'binary') {
    collectMoneyCurrencies(node.left, fields, currencies);
    collectMoneyCurrencies(node.right, fields, currencies);
    return;
  }
  if (node.kind === 'call')
    node.args.forEach((entry) => collectMoneyCurrencies(entry, fields, currencies));
}

function analyzeNode(
  node: FormulaNode,
  fieldsByName: ReadonlyMap<string, FormulaEditorField>,
  functions: ReadonlyMap<string, FormulaEditorFunction>,
  diagnostics: FormulaDiagnostic[],
): void {
  if (node.kind === 'field') {
    const field = fieldsByName.get(node.widgetName);
    if (fieldsByName.size > 0 && !field) {
      diagnostics.push({
        from: node.from,
        to: node.to,
        severity: 'error',
        message: `未找到字段“${node.widgetName}”`,
      });
    } else if (field?.formulaAllowed === false) {
      diagnostics.push({
        from: node.from,
        to: node.to,
        severity: 'error',
        message: `字段“${field.label}”的类型暂不支持参与公式计算`,
      });
    }
    return;
  }
  if (node.kind === 'array') {
    node.elements.forEach((item) => analyzeNode(item, fieldsByName, functions, diagnostics));
    return;
  }
  if (node.kind === 'unary') {
    analyzeNode(node.argument, fieldsByName, functions, diagnostics);
    return;
  }
  if (node.kind === 'binary') {
    analyzeNode(node.left, fieldsByName, functions, diagnostics);
    analyzeNode(node.right, fieldsByName, functions, diagnostics);
    return;
  }
  if (node.kind !== 'call') return;

  const functionSpec = functions.get(node.name);
  if (!functionSpec) {
    diagnostics.push({
      from: node.from,
      to: node.from + node.name.length,
      severity: 'error',
      message: `未收录函数“${node.name}”`,
    });
  } else if (!isSupportedArity(node.args.length, functionSpec)) {
    diagnostics.push({
      from: node.from,
      to: node.to,
      severity: 'error',
      message: `${node.name} 参数个数不符合要求：${functionSpec.syntax}`,
    });
  }
  node.args.forEach((argument) => analyzeNode(argument, fieldsByName, functions, diagnostics));
}

function isSupportedArity(argumentCount: number, functionSpec: FormulaEditorFunction): boolean {
  if (functionSpec.arity) return functionSpec.arity.includes(argumentCount);
  if (functionSpec.minArgs !== undefined && argumentCount < functionSpec.minArgs) return false;
  return functionSpec.maxArgs === undefined || argumentCount <= functionSpec.maxArgs;
}
