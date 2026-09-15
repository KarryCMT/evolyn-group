import { Numeric } from '@evolyn.do/numeric';

export { evaluateFormula, type EvaluateOptions, type FieldResolver } from './evaluator';
export {
  FORMULA_RUNTIME_FUNCTIONS,
  type FormulaFunctionImpl,
  type FunctionContext,
} from './functions';
export { FormulaError, type FormulaErrorCode } from './errors';
export type { RuntimeValue } from './values';

/**
 * 便捷求值：公式源文本 → 解析（诊断即错）→ 字段求值。
 * 结果序列化：数值返回 canonical decimal string（null 空值原样返回 null），
 * 布尔/文本/数组按 JSON 语义返回。
 */
export async function evaluate(source: string): Promise<string | boolean | null> {
  const { parseFormula } = await import('@evolyn.do/formula');
  const { diagnostics, ast } = parseFormula(source);
  if (diagnostics.length > 0 || !ast) {
    const first = diagnostics[0];
    throw new Error(first ? `公式解析失败: ${first.message}` : '公式为空');
  }
  // 便捷入口无字段上下文：所有字段解析为 null（空值传播语义）
  const { evaluateFormula } = await import('./evaluator');
  const value = evaluateFormula(ast, { source });
  if (value instanceof Numeric) {
    return value.serialize();
  }
  if (Array.isArray(value)) {
    return JSON.stringify(value.map((v) => (v instanceof Numeric ? v.serialize() : v)));
  }
  return value;
}
