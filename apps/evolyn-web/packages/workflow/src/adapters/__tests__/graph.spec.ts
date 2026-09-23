import { describe, expect, it } from 'vitest';
import {
  type WorkflowDocument,
  createWorkflowDocument,
  setNodePositions,
} from '../../schema';
import {
  computeAutoLayout,
  ensureDesignerLayout,
  resolveNodePositions,
  toGraphData,
} from '../graph';

function documentWithCycle(): WorkflowDocument {
  return {
    schemaVersion: '1.0',
    nodes: [
      { key: 'start', type: 'start', name: '发起', config: {} },
      { key: 'approval_1', type: 'approval', name: '审批 1', config: {} },
      { key: 'approval_2', type: 'approval', name: '审批 2', config: {} },
      { key: 'end', type: 'end', name: '结束', config: {} },
    ],
    edges: [
      { key: 'e_1', source: 'start', target: 'approval_1' },
      { key: 'e_2', source: 'approval_1', target: 'approval_2' },
      { key: 'e_3', source: 'approval_2', target: 'approval_1' },
      { key: 'e_4', source: 'approval_2', target: 'end' },
    ],
    settings: {},
  };
}

describe('workflow graph layout', () => {
  it('keeps cyclic or unreachable nodes in a finite fallback region', () => {
    const positions = computeAutoLayout(documentWithCycle());

    for (const position of Object.values(positions)) {
      expect(Number.isFinite(position.x)).toBe(true);
      expect(Number.isFinite(position.y)).toBe(true);
      expect(Math.abs(position.x)).toBeLessThan(2_000);
      expect(Math.abs(position.y)).toBeLessThan(2_000);
    }
    expect(positions.start).toEqual({ x: 420, y: 90 });
  });

  it('fills missing coordinates once without overwriting saved coordinates', () => {
    const document = createWorkflowDocument();
    document.settings.designer = { layout: { start: { x: 150, y: 180 } } };

    const normalized = ensureDesignerLayout(document);

    expect(normalized).not.toBe(document);
    expect(normalized.settings.designer?.layout?.start).toEqual({ x: 150, y: 180 });
    expect(normalized.settings.designer?.layout?.end).toEqual({ x: 420, y: 240 });
    expect(ensureDesignerLayout(normalized)).toBe(normalized);
  });

  it('applies batch positions immutably and ignores invalid or unknown entries', () => {
    const document = createWorkflowDocument();
    const next = setNodePositions(document, {
      start: { x: 25, y: 35 },
      end: { x: Number.NaN, y: 50 },
      missing: { x: 80, y: 90 },
    });

    expect(next).not.toBe(document);
    expect(document.settings.designer).toBeUndefined();
    expect(next.settings.designer?.layout).toEqual({ start: { x: 25, y: 35 } });
    expect(resolveNodePositions(next).start).toEqual({ x: 25, y: 35 });
  });

  it('projects readonly state into every node without preserving selection', () => {
    const graph = toGraphData(createWorkflowDocument(), {
      readonly: true,
      selectedNodeKey: null,
    });

    expect(graph.nodes).toHaveLength(2);
    expect(graph.nodes?.every((node) => node.properties?.readonly === true)).toBe(true);
    expect(graph.nodes?.every((node) => node.properties?.selected === false)).toBe(true);
  });
});
