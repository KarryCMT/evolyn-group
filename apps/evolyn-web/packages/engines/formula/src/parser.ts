import type { FormulaDiagnostic, FormulaValueType } from './types';

type TokenKind =
  | 'number'
  | 'string'
  | 'field'
  | 'identifier'
  | 'operator'
  | 'comma'
  | 'leftParen'
  | 'rightParen'
  | 'leftBracket'
  | 'rightBracket'
  | 'eof';

interface Token {
  kind: TokenKind;
  value: string;
  from: number;
  to: number;
}

export type FormulaNode =
  | { kind: 'literal'; valueType: FormulaValueType; from: number; to: number }
  | { kind: 'field'; widgetName: string; from: number; to: number }
  | { kind: 'array'; elements: FormulaNode[]; from: number; to: number }
  | { kind: 'call'; name: string; args: FormulaNode[]; from: number; to: number }
  | { kind: 'unary'; operator: string; argument: FormulaNode; from: number; to: number }
  | {
      kind: 'binary';
      operator: string;
      left: FormulaNode;
      right: FormulaNode;
      from: number;
      to: number;
    };

export interface FormulaParseResult {
  ast: FormulaNode | undefined;
  diagnostics: FormulaDiagnostic[];
}

/**
 * 公式 DSL 的轻量语法分析器。它刻意不执行函数，只负责构建稳定 AST 与精确位置，
 * 供编辑期补全、诊断、参数检查和后续服务端同构实现使用。
 */
export function parseFormula(source: string): FormulaParseResult {
  const lexer = new FormulaLexer(source);
  const tokens = lexer.lex();
  const parser = new FormulaParser(tokens, lexer.diagnostics);
  return parser.parse();
}

class FormulaLexer {
  readonly diagnostics: FormulaDiagnostic[] = [];

  constructor(private readonly source: string) {}

  lex(): Token[] {
    const tokens: Token[] = [];
    let index = 0;

    while (index < this.source.length) {
      const character = this.source[index] ?? '';
      if (/\s/.test(character)) {
        index += 1;
        continue;
      }
      if (character === '$') {
        const match = this.source.slice(index).match(/^\$([A-Za-z_][A-Za-z0-9_]*)#/);
        if (match?.[0] && match[1]) {
          tokens.push({ kind: 'field', value: match[1], from: index, to: index + match[0].length });
          index += match[0].length;
          continue;
        }
        this.error(index, index + 1, '字段引用必须使用 $字段标识# 格式');
        index += 1;
        continue;
      }
      if (character === '"' || character === "'") {
        const quote = character;
        const start = index;
        index += 1;
        let closed = false;
        while (index < this.source.length) {
          if (this.source[index] === '\\') {
            index += 2;
            continue;
          }
          if (this.source[index] === quote) {
            index += 1;
            closed = true;
            break;
          }
          index += 1;
        }
        if (!closed) this.error(start, index, '字符串缺少结束引号');
        tokens.push({
          kind: 'string',
          value: this.source.slice(start, index),
          from: start,
          to: index,
        });
        continue;
      }
      const number = this.source.slice(index).match(/^(?:\d+\.?\d*|\.\d+)/);
      if (number?.[0]) {
        tokens.push({
          kind: 'number',
          value: number[0],
          from: index,
          to: index + number[0].length,
        });
        index += number[0].length;
        continue;
      }
      const identifier = this.source.slice(index).match(/^[A-Za-z_][A-Za-z0-9_]*/);
      if (identifier?.[0]) {
        tokens.push({
          kind: 'identifier',
          value: identifier[0],
          from: index,
          to: index + identifier[0].length,
        });
        index += identifier[0].length;
        continue;
      }
      const pairOperator = this.source.slice(index, index + 2);
      if (['>=', '<=', '==', '!='].includes(pairOperator)) {
        tokens.push({ kind: 'operator', value: pairOperator, from: index, to: index + 2 });
        index += 2;
        continue;
      }
      if ('+-*/%^><'.includes(character)) {
        tokens.push({ kind: 'operator', value: character, from: index, to: index + 1 });
        index += 1;
        continue;
      }
      const punctuation: Partial<Record<string, TokenKind>> = {
        ',': 'comma',
        '(': 'leftParen',
        ')': 'rightParen',
        '[': 'leftBracket',
        ']': 'rightBracket',
      };
      const kind = punctuation[character];
      if (kind) {
        tokens.push({ kind, value: character, from: index, to: index + 1 });
        index += 1;
        continue;
      }
      this.error(index, index + 1, `不支持的字符“${character}”`);
      index += 1;
    }
    tokens.push({ kind: 'eof', value: '', from: this.source.length, to: this.source.length });
    return tokens;
  }

  private error(from: number, to: number, message: string): void {
    this.diagnostics.push({ from, to, severity: 'error', message });
  }
}

class FormulaParser {
  private position = 0;

  constructor(
    private readonly tokens: readonly Token[],
    private readonly diagnostics: FormulaDiagnostic[],
  ) {}

  parse(): FormulaParseResult {
    if (this.currentKind() === 'eof') return { ast: undefined, diagnostics: this.diagnostics };
    const ast = this.parseExpression();
    if (this.currentKind() !== 'eof') this.error(this.current, '此处不应继续出现公式内容');
    return { ast, diagnostics: this.diagnostics };
  }

  private parseExpression(minPrecedence = 0): FormulaNode {
    let left = this.parseUnary();
    while (this.currentKind() === 'operator' && precedence(this.current.value) >= minPrecedence) {
      const operator = this.advance();
      const right = this.parseExpression(precedence(operator.value) + 1);
      left = {
        kind: 'binary',
        operator: operator.value,
        left,
        right,
        from: left.from,
        to: right.to,
      };
    }
    return left;
  }

  private parseUnary(): FormulaNode {
    if (this.currentKind() === 'operator' && ['+', '-'].includes(this.current.value)) {
      const operator = this.advance();
      const argument = this.parseUnary();
      return {
        kind: 'unary',
        operator: operator.value,
        argument,
        from: operator.from,
        to: argument.to,
      };
    }
    return this.parsePrimary();
  }

  private parsePrimary(): FormulaNode {
    const token = this.current;
    if (token.kind === 'number') {
      this.advance();
      return { kind: 'literal', valueType: 'number', from: token.from, to: token.to };
    }
    if (token.kind === 'string') {
      this.advance();
      return { kind: 'literal', valueType: 'text', from: token.from, to: token.to };
    }
    if (token.kind === 'field') {
      this.advance();
      return { kind: 'field', widgetName: token.value, from: token.from, to: token.to };
    }
    if (token.kind === 'identifier') return this.parseIdentifier();
    if (token.kind === 'leftParen') {
      this.advance();
      const expression = this.parseExpression();
      this.expect('rightParen', '缺少与此左括号匹配的右括号');
      return expression;
    }
    if (token.kind === 'leftBracket') return this.parseArray();

    this.error(token, '此处需要字段、函数或值');
    this.advance();
    return { kind: 'literal', valueType: 'unknown', from: token.from, to: token.to };
  }

  private parseIdentifier(): FormulaNode {
    const identifier = this.advance();
    if (this.currentKind() !== 'leftParen') {
      this.error(identifier, '字段必须从字段列表插入，函数调用后必须跟随括号');
      return { kind: 'literal', valueType: 'unknown', from: identifier.from, to: identifier.to };
    }
    this.advance();
    const args: FormulaNode[] = [];
    if (this.currentKind() !== 'rightParen') {
      do {
        args.push(this.parseExpression());
        if (this.currentKind() !== 'comma') break;
        this.advance();
        if (this.currentKind() === 'rightParen') this.error(this.current, '逗号后缺少函数参数');
      } while (this.currentKind() !== 'rightParen' && this.currentKind() !== 'eof');
    }
    const end = this.expect('rightParen', `函数 ${identifier.value} 缺少右括号`);
    return {
      kind: 'call',
      name: identifier.value.toUpperCase(),
      args,
      from: identifier.from,
      to: end?.to ?? this.current.to,
    };
  }

  private parseArray(): FormulaNode {
    const opening = this.advance();
    const elements: FormulaNode[] = [];
    if (this.currentKind() !== 'rightBracket') {
      do {
        elements.push(this.parseExpression());
        if (this.currentKind() !== 'comma') break;
        this.advance();
      } while (this.currentKind() !== 'rightBracket' && this.currentKind() !== 'eof');
    }
    const end = this.expect('rightBracket', '数组缺少右方括号');
    return { kind: 'array', elements, from: opening.from, to: end?.to ?? this.current.to };
  }

  private expect(kind: TokenKind, message: string): Token | undefined {
    if (this.currentKind() === kind) return this.advance();
    this.error(this.current, message);
    return undefined;
  }

  private advance(): Token {
    const token = this.current;
    this.position = Math.min(this.position + 1, this.tokens.length - 1);
    return token;
  }

  private get current(): Token {
    return this.tokens[this.position] ?? this.tokens[this.tokens.length - 1]!;
  }

  private currentKind(): TokenKind {
    return this.current.kind;
  }

  private error(token: Token, message: string): void {
    this.diagnostics.push({
      from: token.from,
      to: Math.max(token.to, token.from + 1),
      severity: 'error',
      message,
    });
  }
}

function precedence(operator: string): number {
  if (['==', '!=', '>', '<', '>=', '<='].includes(operator)) return 1;
  if (['+', '-'].includes(operator)) return 2;
  if (['*', '/', '%'].includes(operator)) return 3;
  if (operator === '^') return 4;
  return -1;
}
