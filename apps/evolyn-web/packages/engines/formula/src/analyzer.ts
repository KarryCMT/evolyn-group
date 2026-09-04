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
  return diagnostics;
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
