import { parseFormula, type FormulaNode } from '@evolyn.do/formula';
import type { FormContent, FormJsonValue, FormItem, SubmitValidatorFailAction } from './types';

/**
 * v7 表单提交校验的框架无关执行器。它只解释已校验的受控 AST，不接触 Vue、HTTP
 * 或任何动态代码执行能力；服务端应以同一 fixture 复现这里的空值与函数语义。
 */
export interface SubmitValidatorFailure {
  index: number;
  remind: string;
  fields: readonly string[];
  failAction: SubmitValidatorFailAction;
}

export interface SubmitValidationContext {
  values: Readonly<Record<string, FormJsonValue>>;
  /** 有效不可见字段在公式和模板中一律视为空，不能泄露保留的会话旧值。 */
  isVisible: (field: string) => boolean;
  /**
   * 宿主可为成员等 ID 型字段注入当前会话已知的展示名；未解析的值返回 undefined，
   * 仍由纯协议层使用默认格式化，避免把应用数据源耦合进 Schema 包。
   */
  formatTemplateValue?: (
    field: string,
    value: FormJsonValue | undefined,
    item: FormItem | undefined,
  ) => string | undefined;
}

/** 发布快照/运行时会话持有的只读编译结果，避免每次输入重新解析公式源码。 */
export interface CompiledSubmitValidation {
  readonly items: readonly FormItem[];
  readonly validators: readonly CompiledSubmitValidator[];
}

interface CompiledSubmitValidator {
  readonly index: number;
  readonly formula: string;
  readonly remind: string;
  readonly failAction: SubmitValidatorFailAction;
  readonly fields: readonly string[];
  readonly ast?: FormulaNode;
  readonly valid: boolean;
}

export function compileSubmitValidators(content: {
  validators: readonly FormContent['validators'][number][];
  items: readonly FormItem[];
}): CompiledSubmitValidation {
  return {
    items: content.items,
    validators: content.validators.map((validator, index) => {
      const parsed = parseFormula(validator.formula);
      const valid =
        Boolean(parsed.ast) && !parsed.diagnostics.some((entry) => entry.severity === 'error');
      return {
        index,
        formula: validator.formula,
        remind: validator.remind,
        failAction: validator.failAction,
        fields: parsed.ast ? [...collectFormulaFields(parsed.ast)] : [],
        ast: parsed.ast,
        valid,
      };
    }),
  };
}

/** 每条规则的字段依赖，供运行时构建反向索引；顺序严格对应 validators 数组。 */
export function collectSubmitValidatorDependencies(
  validators: readonly FormContent['validators'][number][],
): readonly (readonly string[])[] {
  return compileSubmitValidators({ validators, items: [] }).validators.map(
    (validator) => validator.fields,
  );
}

/** 执行全部规则并保持协议数组顺序。公式异常按不通过处理，避免运行时崩溃放行提交。 */
export function evaluateSubmitValidators(
  content: Pick<FormContent, 'validators' | 'items'>,
  context: SubmitValidationContext,
  indexes?: ReadonlySet<number>,
): SubmitValidatorFailure[] {
  return evaluateCompiledSubmitValidators(compileSubmitValidators(content), context, indexes);
}

/** 执行会话缓存的 AST；调用方可按反向依赖索引传入局部规则集合。 */
export function evaluateCompiledSubmitValidators(
  compiled: CompiledSubmitValidation,
  context: SubmitValidationContext,
  indexes?: ReadonlySet<number>,
): SubmitValidatorFailure[] {
  const failures: SubmitValidatorFailure[] = [];
  compiled.validators.forEach((validator) => {
    if (indexes && !indexes.has(validator.index)) return;
    if (!validator.ast || !validator.valid) {
      failures.push({
        index: validator.index,
        remind: renderSubmitTemplate(validator.remind, compiled.items, context),
        fields: [],
        failAction: validator.failAction,
      });
      return;
    }
    try {
      const value = evaluateFormulaNode(validator.ast, validator.formula, context);
      if (value === true) return;
    } catch {
      // 受控公式即使遇到用户输入造成的类型不匹配，也必须以可预期的失败收口。
    }
    failures.push({
      index: validator.index,
      remind: renderSubmitTemplate(validator.remind, compiled.items, context),
      fields: validator.fields,
      failAction: validator.failAction,
    });
  });
  return failures;
}

/** 仅在字段有效可见时插值，选项值转换为标签，数组以“、”连接。 */
export function renderSubmitTemplate(
  template: string,
  items: readonly FormItem[],
  context: SubmitValidationContext,
): string {
  const itemMap = new Map(items.map((item) => [item.widget.widgetName, item]));
  return template.replace(/\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g, (_, field: string) => {
    if (!context.isVisible(field)) return '';
    const item = itemMap.get(field);
    return (
      context.formatTemplateValue?.(field, context.values[field], item) ??
      formatTemplateValue(context.values[field], item)
    );
  });
}

function collectFormulaFields(node: FormulaNode): Set<string> {
  const fields = new Set<string>();
  visitFormulaNode(node, (current) => {
    if (current.kind === 'field') fields.add(current.widgetName);
  });
  return fields;
}

function visitFormulaNode(node: FormulaNode, visit: (entry: FormulaNode) => void): void {
  visit(node);
  if (node.kind === 'array') node.elements.forEach((entry) => visitFormulaNode(entry, visit));
  if (node.kind === 'unary') visitFormulaNode(node.argument, visit);
  if (node.kind === 'binary') {
    visitFormulaNode(node.left, visit);
    visitFormulaNode(node.right, visit);
  }
  if (node.kind === 'call') node.args.forEach((entry) => visitFormulaNode(entry, visit));
}

type FormulaRuntimeValue = string | number | boolean | null | Date | FormulaRuntimeValue[];

function toFormulaValue(value: FormJsonValue | undefined): FormulaRuntimeValue {
  if (value === undefined || value === null) return null;
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean')
    return value;
  if (Array.isArray(value)) return value.map(toFormulaValue);
  // 对不开放的结构值保持空值语义；Schema 校验会在保存期拦截其字段引用。
  return null;
}

function evaluateFormulaNode(
  node: FormulaNode,
  source: string,
  context: SubmitValidationContext,
): FormulaRuntimeValue {
  if (node.kind === 'literal') return readLiteral(source.slice(node.from, node.to), node.valueType);
  if (node.kind === 'field') {
    return context.isVisible(node.widgetName)
      ? toFormulaValue(context.values[node.widgetName])
      : null;
  }
  if (node.kind === 'array')
    return node.elements.map((entry) => evaluateFormulaNode(entry, source, context));
  if (node.kind === 'unary') {
    const value = toNumber(evaluateFormulaNode(node.argument, source, context));
    return node.operator === '-' ? -value : value;
  }
  if (node.kind === 'binary') {
    const left = evaluateFormulaNode(node.left, source, context);
    const right = evaluateFormulaNode(node.right, source, context);
    switch (node.operator) {
      case '+':
        return toNumber(left) + toNumber(right);
      case '-':
        return toNumber(left) - toNumber(right);
      case '*':
        return toNumber(left) * toNumber(right);
      case '/': {
        const divisor = toNumber(right);
        if (divisor === 0) throw new Error('division by zero');
        return toNumber(left) / divisor;
      }
      case '%':
        return toNumber(left) % toNumber(right);
      case '^':
        return toNumber(left) ** toNumber(right);
      case '==':
        return compare(left, right) === 0;
      case '!=':
        return compare(left, right) !== 0;
      case '>':
        return compare(left, right) > 0;
      case '>=':
        return compare(left, right) >= 0;
      case '<':
        return compare(left, right) < 0;
      case '<=':
        return compare(left, right) <= 0;
      default:
        throw new Error('unsupported operator');
    }
  }
  const args = node.args.map((entry) => evaluateFormulaNode(entry, source, context));
  return evaluateFormulaFunction(node.name, args);
}

function readLiteral(raw: string, type: string): FormulaRuntimeValue {
  if (type === 'number') return Number(raw);
  if (type !== 'text') return null;
  // 解析器已保证引号闭合；这里仅还原常见转义，绝不将文本交给动态执行器。
  const body = raw.slice(1, -1);
  return body.replace(/\\(['"\\])/g, '$1');
}

function evaluateFormulaFunction(name: string, args: FormulaRuntimeValue[]): FormulaRuntimeValue {
  switch (name) {
    case 'AND':
      return args.every((value) => value === true);
    case 'OR':
      return args.some((value) => value === true);
    case 'NOT':
      return args[0] === false;
    case 'IF':
      return args[0] === true ? (args[1] ?? null) : (args[2] ?? null);
    case 'ISBLANK':
      return isBlank(args[0]);
    case 'LEN':
      return toText(args[0]).length;
    case 'CONCATENATE':
      return args.map(toText).join('');
    case 'LOWER':
      return toText(args[0]).toLowerCase();
    case 'UPPER':
      return toText(args[0]).toUpperCase();
    case 'TRIM':
      return toText(args[0]).trim().replace(/\s+/g, ' ');
    case 'ABS':
      return Math.abs(toNumber(args[0]));
    case 'ROUND': {
      const precision = args.length > 1 ? toNumber(args[1]) : 0;
      const factor = 10 ** precision;
      return Math.round((toNumber(args[0]) + Number.EPSILON) * factor) / factor;
    }
    case 'DATE':
      return createDate(args);
    case 'DATEDIF':
      return dateDifference(args);
    default:
      throw new Error(`unsupported function ${name}`);
  }
}

function isBlank(value: FormulaRuntimeValue | undefined): boolean {
  return (
    value === undefined ||
    value === null ||
    value === '' ||
    (Array.isArray(value) && value.length === 0)
  );
}

function toNumber(value: FormulaRuntimeValue | undefined): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error('number required');
  return value;
}

function toText(value: FormulaRuntimeValue | undefined): string {
  if (value === undefined || value === null) return '';
  if (value instanceof Date) return value.toISOString();
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean')
    return String(value);
  if (Array.isArray(value)) return value.map(toText).join('、');
  throw new Error('text required');
}

function compare(left: FormulaRuntimeValue, right: FormulaRuntimeValue): number {
  if (left === null || right === null) throw new Error('null cannot be compared');
  const comparableLeft = left instanceof Date ? left.getTime() : left;
  const comparableRight = right instanceof Date ? right.getTime() : right;
  if (
    (typeof comparableLeft !== 'string' &&
      typeof comparableLeft !== 'number' &&
      typeof comparableLeft !== 'boolean') ||
    typeof comparableLeft !== typeof comparableRight
  ) {
    throw new Error('incompatible values');
  }
  if (comparableLeft === comparableRight) return 0;
  return comparableLeft > (comparableRight as string | number | boolean) ? 1 : -1;
}

function createDate(args: FormulaRuntimeValue[]): Date {
  if (args.length === 1) {
    const value = new Date(toNumber(args[0]));
    if (Number.isNaN(value.getTime())) throw new Error('invalid date');
    return value;
  }
  const year = toNumber(args[0]);
  const month = toNumber(args[1]);
  const day = toNumber(args[2]);
  const hour = args.length === 6 ? toNumber(args[3]) : 0;
  const minute = args.length === 6 ? toNumber(args[4]) : 0;
  const second = args.length === 6 ? toNumber(args[5]) : 0;
  const value = new Date(Date.UTC(year, month - 1, day, hour, minute, second));
  if (
    value.getUTCFullYear() !== year ||
    value.getUTCMonth() !== month - 1 ||
    value.getUTCDate() !== day
  ) {
    throw new Error('invalid date');
  }
  return value;
}

function dateDifference(args: FormulaRuntimeValue[]): number {
  const start = toDate(args[0]);
  const end = toDate(args[1]);
  const unit = toText(args[2]).toUpperCase();
  const milliseconds = end.getTime() - start.getTime();
  if (unit === 'D') return Math.trunc(milliseconds / 86_400_000);
  if (unit === 'M')
    return (
      (end.getUTCFullYear() - start.getUTCFullYear()) * 12 + end.getUTCMonth() - start.getUTCMonth()
    );
  if (unit === 'Y') return end.getUTCFullYear() - start.getUTCFullYear();
  throw new Error('unsupported date unit');
}

function toDate(value: FormulaRuntimeValue | undefined): Date {
  if (value instanceof Date) return value;
  if (typeof value !== 'string') throw new Error('date required');
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) throw new Error('invalid date');
  return parsed;
}

function formatTemplateValue(value: FormJsonValue | undefined, item?: FormItem): string {
  if (value === undefined || value === null || value === '') return '';
  const widget = item?.widget;
  const options = widget && 'options' in widget ? widget.options : undefined;
  const labelFor = (entry: FormJsonValue): string => {
    if (typeof entry !== 'string') return String(entry);
    return options?.find((option) => option.value === entry)?.label ?? entry;
  };
  return Array.isArray(value) ? value.map(labelFor).join('、') : labelFor(value);
}
