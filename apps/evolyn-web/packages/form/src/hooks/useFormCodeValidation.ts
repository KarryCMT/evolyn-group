import type { PluginCodeDiagnostic, PluginDesignFunction } from '../types';
import { getFormRuntimeLanguage } from './useFormRuntime';

type BracketState = {
  bracket: string;
  line: number;
  column: number;
};

const closingBracketMap: Record<string, string> = {
  ')': '(',
  ']': '[',
  '}': '{',
};

const openingBracketMap: Record<string, string> = {
  '(': ')',
  '[': ']',
  '{': '}',
};

const operatorBeforeCallRegExp = /(?:[=+\-*/%,:[{(]|return|yield|and|or|not|if|elif|while)\s*$/u;
const allowedPythonKeywordAfterClosingParen = new Set([
  'and',
  'as',
  'else',
  'for',
  'from',
  'if',
  'in',
  'is',
  'not',
  'or',
]);

const createDiagnostic = (
  line: number,
  column: number,
  message: string,
  endColumn = column + 1,
): PluginCodeDiagnostic => ({
  line,
  column,
  endLine: line,
  endColumn,
  message,
  severity: 'error',
});

const isEscaped = (line: string, index: number) => {
  let slashCount = 0;
  for (let current = index - 1; current >= 0 && line[current] === '\\'; current -= 1) {
    slashCount += 1;
  }
  return slashCount % 2 === 1;
};

const validateBalancedTokens = (code: string) => {
  const diagnostics: PluginCodeDiagnostic[] = [];
  const brackets: BracketState[] = [];
  let stringQuote = '';
  let stringLine = 0;
  let stringColumn = 0;

  code.split(/\r?\n/u).forEach((line, lineIndex) => {
    let columnIndex = 0;
    while (columnIndex < line.length) {
      const char = line[columnIndex];
      const nextThree = line.slice(columnIndex, columnIndex + 3);
      const isTripleQuote = nextThree === "'''" || nextThree === '"""';

      if (stringQuote) {
        if (
          (stringQuote.length === 3 && line.slice(columnIndex, columnIndex + 3) === stringQuote) ||
          (stringQuote.length === 1 && char === stringQuote && !isEscaped(line, columnIndex))
        ) {
          columnIndex += stringQuote.length;
          stringQuote = '';
          continue;
        }
        columnIndex += 1;
        continue;
      }

      if (char === '#') break;

      if (isTripleQuote || char === "'" || char === '"') {
        stringQuote = isTripleQuote ? nextThree : char;
        stringLine = lineIndex + 1;
        stringColumn = columnIndex + 1;
        columnIndex += stringQuote.length;
        continue;
      }

      if (openingBracketMap[char]) {
        brackets.push({ bracket: char, line: lineIndex + 1, column: columnIndex + 1 });
      } else if (closingBracketMap[char]) {
        const lastBracket = brackets.pop();
        if (!lastBracket || lastBracket.bracket !== closingBracketMap[char]) {
          diagnostics.push(createDiagnostic(lineIndex + 1, columnIndex + 1, '括号不匹配'));
        }
      }
      columnIndex += 1;
    }
    if (stringQuote.length === 1) {
      diagnostics.push(
        createDiagnostic(stringLine, stringColumn, '字符串没有正确闭合', line.length + 1),
      );
      stringQuote = '';
    }
  });

  if (!diagnostics.length && stringQuote) {
    diagnostics.push(createDiagnostic(stringLine, stringColumn, '字符串没有正确闭合'));
  }
  if (brackets.length) {
    brackets.forEach((lastBracket) => {
      // 未闭合括号可能来自更早的行，逐个标出，避免只提示最后一个错误。
      diagnostics.push(
        createDiagnostic(
          lastBracket.line,
          lastBracket.column,
          `${'括号不匹配'}，${'缺少'} ${openingBracketMap[lastBracket.bracket]}`,
        ),
      );
    });
  }

  return diagnostics;
};

const validatePythonIndent = (lines: string[]) => {
  const diagnostics: PluginCodeDiagnostic[] = [];
  lines.forEach((line, index) => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#') || !trimmed.endsWith(':')) return;

    const currentIndent = line.match(/^\s*/u)?.[0].length || 0;
    const nextIndex = lines.findIndex((nextLine, nextLineIndex) => {
      return nextLineIndex > index && nextLine.trim() && !nextLine.trim().startsWith('#');
    });
    if (nextIndex === -1) {
      diagnostics.push(createDiagnostic(index + 1, line.length, '冒号后缺少缩进代码块'));
      return;
    }

    const nextIndent = lines[nextIndex].match(/^\s*/u)?.[0].length || 0;
    if (nextIndent <= currentIndent) {
      diagnostics.push(createDiagnostic(nextIndex + 1, nextIndent + 1, '冒号后缺少缩进代码块'));
    }
  });
  return diagnostics;
};

const validatePythonAdjacentCall = (lines: string[]) => {
  const diagnostics: PluginCodeDiagnostic[] = [];
  lines.forEach((line, index) => {
    const codePart = line.split('#')[0];
    const adjacentCallMatch = codePart.match(/\)\s*([A-Za-z_][\w]*)/u);
    if (adjacentCallMatch?.index === undefined) return;

    const closingParenWithKeyword = adjacentCallMatch[0];
    const hasKeywordSpacing = /\)\s+[A-Za-z_]/u.test(closingParenWithKeyword);
    // Python 允许括号表达式后接关键字，如三元表达式 `if ... else ...`、异常链 `raise Error() from exc`。
    if (hasKeywordSpacing && allowedPythonKeywordAfterClosingParen.has(adjacentCallMatch[1]))
      return;

    const beforeCall = codePart.slice(0, adjacentCallMatch.index + 1);
    const previousText = beforeCall.slice(0, Math.max(0, beforeCall.lastIndexOf('('))).trim();
    if (operatorBeforeCallRegExp.test(previousText)) return;

    diagnostics.push(
      createDiagnostic(
        index + 1,
        adjacentCallMatch.index + 2,
        '函数调用后缺少换行、赋值或运算符',
        adjacentCallMatch.index + adjacentCallMatch[0].length + 1,
      ),
    );
  });
  return diagnostics;
};

const validatePythonCode = (code: string) => {
  const lines = code.split(/\r?\n/u);
  return [
    ...validateBalancedTokens(code),
    ...validatePythonAdjacentCall(lines),
    ...validatePythonIndent(lines),
  ];
};

const validateGenericCode = () => {
  // JS/Java 代码可能包含多种注释和模板字符串，前端轻量规则容易误伤；先交给 Monaco/运行时处理。
  return [];
};

export const validateFormCode = (functionData: PluginDesignFunction): PluginCodeDiagnostic[] => {
  if (!functionData.code.trim()) {
    return [createDiagnostic(1, 1, '代码不能为空')];
  }

  const language = getFormRuntimeLanguage(functionData.runtime);
  if (language === 'python') return validatePythonCode(functionData.code);
  return validateGenericCode();
};

/**
 * 校验插件设计中所有函数代码，返回第一个出错函数，保存前用它阻断明显的语法问题。
 */
export const validateFormDesignCode = (functions: PluginDesignFunction[]) => {
  for (const functionData of functions) {
    const diagnostics = validateFormCode(functionData);
    if (diagnostics.length) {
      return {
        functionData,
        diagnostics,
      };
    }
  }
  return undefined;
};
