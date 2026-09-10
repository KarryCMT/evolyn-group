import { Numeric, NumericError, numeric } from '@evolyn.do/numeric'
import type { NumericInput, NumericRuntime } from '@evolyn.do/numeric'
import type { FormulaNode } from '@evolyn.do/formula'

import { FormulaError } from './errors'
import { FORMULA_RUNTIME_FUNCTIONS } from './functions'
import type { FunctionContext } from './functions'
import { describe as describeValue, tryAsNumeric } from './values'
import type { RuntimeValue } from './values'

/**
 * 字段解析器：widgetName → 运行时值。表单记录数据（string/number/null/
 * 数组）由调用方提供；未提供视为 null（平台空值策略交由 NumericRuntime）。
 */
export type FieldResolver = (widgetName: string) => RuntimeValue | undefined

export interface EvaluateOptions {
  /** 公式源文本（literal 原文还原依赖 from/to 切片，设计 §20） */
  source: string
  /** 字段取值器；缺省恒 null（空值传播） */
  resolveField?: FieldResolver
  /** 数值运行时（平台默认/字段级上下文） */
  runtime?: NumericRuntime
}

/** evaluateFormula 求值入口：AST + 源文本 + 字段解析器 → 运行时值 */
export function evaluateFormula(ast: FormulaNode, options: EvaluateOptions): RuntimeValue {
  return new Evaluator(options).evaluate(ast)
}

class Evaluator {
  readonly #source: string
  readonly #resolve: FieldResolver
  readonly #rt: NumericRuntime
  readonly #ctx: FunctionContext

  constructor(options: EvaluateOptions) {
    this.#source = options.source
    this.#resolve = options.resolveField ?? (() => undefined)
    this.#rt = options.runtime ?? numeric
    this.#ctx = { runtime: this.#rt }
  }

  evaluate(node: FormulaNode): RuntimeValue {
    switch (node.kind) {
      case 'literal':
        return this.literal(node)
      case 'field':
        return this.field(node)
      case 'unary':
        return this.unary(node)
      case 'binary':
        return this.binary(node)
      case 'call':
        return this.call(node)
      case 'array':
        return node.elements.map((el) => this.evaluate(el))
    }
  }

  /** literal 原文还原（设计 §20）：number 保原始字符串进 Numeric；串去引号 */
  private literal(node: FormulaNode & { kind: 'literal' }): RuntimeValue {
    const raw = this.#source.slice(node.from, node.to)
    switch (node.valueType) {
      case 'number':
        return this.of(raw)
      case 'text': {
        const inner = raw.slice(1, -1)
        return inner.replace(/\\(.)/g, '$1')
      }
      case 'boolean':
        return /^true$/i.test(raw)
      default:
        // identifier 形态的 TRUE/FALSE 由 parser 标为 unknown literal
        if (/^(true|false)$/i.test(raw)) {
          return /^true$/i.test(raw)
        }
        throw new FormulaError('INVALID_LITERAL', `无法识别的字面量: ${raw}`)
    }
  }

  private field(node: FormulaNode & { kind: 'field' }): RuntimeValue {
    const value = this.#resolve(node.widgetName)
    if (value === undefined) {
      // 字段未提供：空值语义（平台默认 NULL 传播），与 null 同口径
      return null
    }
    return value
  }

  private unary(node: FormulaNode & { kind: 'unary' }): RuntimeValue {
    const value = this.evaluate(node.argument)
    if (node.operator === '+') {
      return tryAsNumeric(value, this.#rt) ?? value
    }
    if (node.operator === '-') {
      const n = tryAsNumeric(value, this.#rt)
      return n === null ? null : n.negate()
    }
    throw new FormulaError('UNSUPPORTED_OPERATOR', `不支持的一元运算符: ${node.operator}`)
  }

  private binary(node: FormulaNode & { kind: 'binary' }): RuntimeValue {
    const left = this.evaluate(node.left)
    const right = this.evaluate(node.right)

    switch (node.operator) {
      case '+':
      case '-':
      case '*':
      case '/':
      case '%':
      case '^':
        return this.arithmetic(node.operator, left, right)
      case '==':
      case '!=':
      case '>':
      case '<':
      case '>=':
      case '<=':
        return this.compare(node.operator, left, right)
      default:
        throw new FormulaError('UNSUPPORTED_OPERATOR', `不支持的二元运算符: ${node.operator}`)
    }
  }

  /** 数值运算（设计 §19）：两侧经 Numeric 桥接，禁止原生 JS 数值运算。
   *  空操作数经 of(null) 进入空值策略（平台默认 NULL 传播），不在本层抛错 */
  private arithmetic(op: string, left: RuntimeValue, right: RuntimeValue): RuntimeValue {
    const a = this.ofNullable(left)
    const bInput = this.toInput(right)
    try {
      switch (op) {
        case '+':
          return a.add(bInput)
        case '-':
          return a.subtract(bInput)
        case '*':
          return a.multiply(bInput)
        case '/':
          return a.divide(bInput)
        case '%':
          return a.mod(bInput)
        case '^':
          return a.pow(bInput)
        default:
          throw new FormulaError('UNSUPPORTED_OPERATOR', `不支持的算术运算符: ${op}`)
      }
    } catch (err) {
      if (err instanceof FormulaError) {
        throw err
      }
      throw FormulaError.fromNumeric(err as NumericError)
    }
  }

  /** 数值样判定：Numeric/number/十进制文本；纯文本（"a"）不得进入数值化 */
  private isNumericLike(value: RuntimeValue): boolean {
    if (value instanceof Numeric || typeof value === 'number') {
      return true
    }
    return typeof value === 'string' && /^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$/.test(value.trim())
  }

  /** 比较：两侧数值样走数值谓词（SQL 空值假语义）；否则文本/布尔按 JS 语义 */
  private compare(op: string, left: RuntimeValue, right: RuntimeValue): boolean {
    if (this.isNumericLike(left) && this.isNumericLike(right)) {
      const ln = tryAsNumeric(left, this.#rt)
      const rn = tryAsNumeric(right, this.#rt)
      if (ln !== null && rn !== null) {
        return this.numericCompare(op, ln, rn)
      }
      return false // 空值比较恒假（SQL 语义）
    }
    return this.plainCompare(op, left, right)
  }

  private numericCompare(op: string, ln: Numeric, rn: Numeric): boolean {
    switch (op) {
      case '==':
        return ln.eq(rn)
      case '!=':
        return !ln.eq(rn)
      case '>':
        return ln.gt(rn)
      case '<':
        return ln.lt(rn)
      case '>=':
        return ln.gte(rn)
      case '<=':
        return ln.lte(rn)
      default:
        return false
    }
  }

  /** 非数值比较：文本字典序、布尔/空值仅等值判定，其余关系运算恒假 */
  private plainCompare(op: string, left: RuntimeValue, right: RuntimeValue): boolean {
    const textCompare =
      typeof left === 'string' && typeof right === 'string' ? left.localeCompare(right) : null
    switch (op) {
      case '==':
        return left === right
      case '!=':
        return left !== right
      case '>':
        return textCompare !== null && textCompare > 0
      case '<':
        return textCompare !== null && textCompare < 0
      case '>=':
        return textCompare !== null && textCompare >= 0
      case '<=':
        return textCompare !== null && textCompare <= 0
      default:
        return false
    }
  }

  private call(node: FormulaNode & { kind: 'call' }): RuntimeValue {
    const impl = FORMULA_RUNTIME_FUNCTIONS[node.name.toUpperCase()]
    if (!impl) {
      throw new FormulaError('FUNCTION_NOT_IMPLEMENTED', `函数未实现: ${node.name}`)
    }
    const args = node.args.map((arg) => this.evaluate(arg))
    return impl(args, this.#ctx)
  }

  /** ofNullable 操作数构造：null/undefined 走策略（NULL→空 Numeric 传播） */
  private ofNullable(value: RuntimeValue): Numeric {
    if (value === null || value === undefined) {
      return this.of(null)
    }
    if (value instanceof Numeric) {
      return value
    }
    if (typeof value === 'string' || typeof value === 'number') {
      return this.of(value)
    }
    throw new FormulaError('INVALID_NUMBER', `操作数不是数值: ${describeValue(value)}`)
  }

  private toInput(value: RuntimeValue): NumericInput {
    const n = tryAsNumeric(value, this.#rt)
    return n === null ? null : n
  }

  private of(input: NumericInput): Numeric {
    try {
      return this.#rt.of(input)
    } catch (err) {
      throw FormulaError.fromNumeric(err as NumericError)
    }
  }

}
