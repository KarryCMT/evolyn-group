import type LogicFlow from '@logicflow/core';
import { watch } from 'vue';
import { toGraphData } from '../adapters/graph';
import type { WorkflowDocument, WorkflowPosition } from '../schema';

type ViewMode = 'compact' | 'detailed';

interface WorkflowGraphSyncOptions {
  document: () => WorkflowDocument;
  selectedNodeKey: () => string | null;
  selectedEdgeKey: () => string | null;
  errorNodeKeys: () => ReadonlySet<string> | undefined;
  errorEdgeKeys: () => ReadonlySet<string> | undefined;
  viewMode: () => ViewMode;
  readonly: () => boolean;
}

/**
 * LogicFlow 是命令式外部状态，DSL 文档仍是业务事实源。该同步层将结构变更、
 * 视觉属性、选择、坐标和只读状态拆开处理，避免任意 prop 变化都销毁整张图。
 */
export function useWorkflowGraphSync(options: WorkflowGraphSyncOptions) {
  let instance: LogicFlow | null = null;
  let renderedStructure = '';
  let draggingNodeKey: string | null = null;
  let structuralSyncPending = false;
  const stablePositions = new Map<string, WorkflowPosition>();

  function graphState() {
    return {
      selectedNodeKey: options.selectedNodeKey(),
      selectedEdgeKey: options.selectedEdgeKey(),
      errorNodeKeys: options.errorNodeKeys(),
      errorEdgeKeys: options.errorEdgeKeys(),
      viewMode: options.viewMode(),
    };
  }

  /** 节点/边拓扑签名：只有真正的结构变化才允许执行 LogicFlow.render。 */
  function structureSignature(document: WorkflowDocument): string {
    return JSON.stringify({
      nodes: document.nodes.map((node) => [node.key, node.type]),
      edges: document.edges.map((edge) => [edge.key, edge.source, edge.target]),
    });
  }

  /**
   * 缺少持久化坐标的历史节点在当前画布会话中只计算一次位置；后续结构或选择
   * 变化继续复用，直至用户发生实际编辑并由上层补齐完整 layout。
   */
  function projectGraph() {
    const document = options.document();
    const graph = toGraphData(document, graphState());
    const savedLayout = document.settings.designer?.layout ?? {};
    const activeNodeKeys = new Set(document.nodes.map((node) => node.key));

    for (const key of stablePositions.keys()) {
      if (!activeNodeKeys.has(key)) stablePositions.delete(key);
    }
    for (const node of graph.nodes ?? []) {
      const nodeKey = String(node.id);
      const saved = savedLayout[nodeKey];
      if (saved && Number.isFinite(saved.x) && Number.isFinite(saved.y)) {
        stablePositions.set(nodeKey, { x: saved.x, y: saved.y });
      } else {
        const stable = stablePositions.get(nodeKey);
        if (stable) {
          node.x = stable.x;
          node.y = stable.y;
        } else {
          stablePositions.set(nodeKey, { x: node.x, y: node.y });
        }
      }
    }
    return graph;
  }

  function renderFullGraph() {
    if (!instance) return;
    const document = options.document();
    instance.render(projectGraph());
    renderedStructure = structureSignature(document);
    structuralSyncPending = false;
    syncSelection();
    syncReadonly();
  }

  function syncSelection() {
    if (!instance) return;
    const nodeKey = options.selectedNodeKey();
    const edgeKey = options.selectedEdgeKey();

    for (const node of instance.graphModel.nodes) {
      instance.setProperties(node.id, { selected: node.id === nodeKey });
    }
    for (const edge of instance.graphModel.edges) {
      instance.setProperties(edge.id, { selected: edge.id === edgeKey });
    }

    instance.clearSelectElements();
    const selectedKey = nodeKey ?? edgeKey;
    if (selectedKey && instance.getModelById(selectedKey)) {
      // 业务只支持单选，且选择不应改变节点层级。
      instance.selectElementById(selectedKey, false, false);
    }
  }

  /** 拖拽开始时立即收敛引擎内部选区，但绝不重建画布。 */
  function beginNodeDrag(nodeKey: string) {
    if (!instance) return;
    draggingNodeKey = nodeKey;
    instance.clearSelectElements();
    instance.selectElementById(nodeKey, false, false);
    for (const node of instance.graphModel.nodes) {
      instance.setProperties(node.id, { selected: node.id === nodeKey });
    }
    for (const edge of instance.graphModel.edges) {
      instance.setProperties(edge.id, { selected: false });
    }
  }

  /** 节点即时坐标进入会话缓存，等待上层不可变写回 DSL。 */
  function finishNodeDrag(nodeKey: string, position: WorkflowPosition) {
    stablePositions.set(nodeKey, { ...position });
    draggingNodeKey = null;
    if (structuralSyncPending) syncDocument();
  }

  function syncElementProperties(movePositions: boolean) {
    if (!instance) return;
    const graph = projectGraph();
    for (const node of graph.nodes ?? []) {
      const nodeKey = String(node.id);
      const model = instance.getNodeModelById(nodeKey);
      if (!model) continue;
      instance.setProperties(nodeKey, node.properties ?? {});
      if (
        movePositions &&
        draggingNodeKey !== nodeKey &&
        (model.x !== node.x || model.y !== node.y)
      ) {
        instance.graphModel.moveNode2Coordinate(nodeKey, node.x, node.y);
      }
    }
    for (const edge of graph.edges ?? []) {
      const edgeKey = String(edge.id);
      if (!instance.getEdgeModelById(edgeKey)) continue;
      instance.setProperties(edgeKey, edge.properties ?? {});
      const text = typeof edge.text === 'string' ? edge.text : '';
      instance.updateText(edgeKey, text);
    }
  }

  function syncDocument() {
    if (!instance) return;
    const nextStructure = structureSignature(options.document());
    if (nextStructure !== renderedStructure) {
      if (draggingNodeKey) {
        structuralSyncPending = true;
        return;
      }
      renderFullGraph();
      return;
    }
    syncElementProperties(true);
    syncSelection();
  }

  function syncVisualState() {
    if (!instance) return;
    syncElementProperties(false);
    syncSelection();
  }

  function syncReadonly() {
    if (!instance) return;
    const readonly = options.readonly();
    instance.updateEditConfig({
      adjustNodePosition: !readonly,
      hideAnchors: readonly,
    });
  }

  function initialize(nextInstance: LogicFlow) {
    instance = nextInstance;
    renderFullGraph();
  }

  function destroy() {
    instance = null;
    renderedStructure = '';
    draggingNodeKey = null;
    structuralSyncPending = false;
    stablePositions.clear();
  }

  watch(options.document, syncDocument, { deep: false });
  watch([options.selectedNodeKey, options.selectedEdgeKey], syncSelection);
  watch(
    [options.errorNodeKeys, options.errorEdgeKeys, options.viewMode],
    syncVisualState,
    { deep: false },
  );
  watch(options.readonly, syncReadonly);

  return {
    beginNodeDrag,
    destroy,
    finishNodeDrag,
    initialize,
    syncVisualState,
  };
}

