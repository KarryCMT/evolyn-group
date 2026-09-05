import { describe, expect, it } from 'vitest';
import { compileRuleGraph, downstreamRuleTargets, evaluateRuleGraph } from '../graph';

const graph = compileRuleGraph([
  { id: 'rule-a', rel: 'and', conditions: [{ field: 'source' }], targets: ['middle'] },
  { id: 'rule-b', rel: 'and', conditions: [{ field: 'middle' }], targets: ['target'] },
]);

describe('Rule Engine dependency graph', () => {
  it('maintains deterministic downstream topological order', () => {
    expect(downstreamRuleTargets(graph, 'source')).toEqual(['middle', 'target']);
  });

  it('evaluates only rules whose condition sources are visible', () => {
    const visibility = evaluateRuleGraph(graph, {
      isBaseVisible: () => true,
      matchRule: (rule, context) =>
        rule.conditions.every((condition) => context.isFieldVisible(condition.field)),
    });

    expect([...visibility]).toEqual([
      ['middle', true],
      ['target', true],
    ]);
  });
});
