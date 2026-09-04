import type { QueryConjunction, QueryExpression } from './types.js';

/**
 * 以结合律合成筛选表达式，并扁平化同类分组，避免设计器多次编辑后产生无意义嵌套。
 */
export function composeQueryFilters(
  conjunction: QueryConjunction,
  expressions: readonly (QueryExpression | undefined)[],
): QueryExpression | undefined {
  const children = expressions.filter((expression): expression is QueryExpression => Boolean(expression));
  if (!children.length) return undefined;
  if (children.length === 1) return children[0];

  return {
    type: 'group',
    conjunction,
    children: children.flatMap((child) =>
      child.type === 'group' && child.conjunction === conjunction ? child.children : [child],
    ),
  };
}
