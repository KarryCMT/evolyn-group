import type { IntelligentDocument } from '../schema';

const CENTER_X = 520;
const TOP_Y = 130;
const NODE_GAP = 154;

/** 将当前节点按执行顺序恢复为稳定的纵向布局。 */
export function layoutIntelligentDocument(document: IntelligentDocument): IntelligentDocument {
  const ordered = [...document.nodes].sort((left, right) => {
    const rank = { trigger: 0, action: 1, end: 2 } as const;
    if (rank[left.type] !== rank[right.type]) return rank[left.type] - rank[right.type];
    return left.position.y - right.position.y;
  });

  return {
    ...document,
    nodes: ordered.map((node, index) => ({
      ...node,
      position: { x: CENTER_X, y: TOP_Y + index * NODE_GAP },
    })),
  };
}
