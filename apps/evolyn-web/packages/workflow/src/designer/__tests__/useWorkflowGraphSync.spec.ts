import type LogicFlow from '@logicflow/core';
import { effectScope, nextTick, shallowRef } from 'vue';
import { describe, expect, it } from 'vitest';
import { addNode, createWorkflowDocument, setNodePosition } from '../../schema';
import { useWorkflowGraphSync } from '../useWorkflowGraphSync';

interface FakeElementModel {
  id: string;
  x: number;
  y: number;
  text?: string;
  properties: Record<string, unknown>;
}

/** 只实现同步层使用的 LogicFlow 窄接口，避免测试依赖 DOM。 */
class FakeLogicFlow {
  renderCount = 0;
  selectedKey: string | null = null;
  editConfig: Record<string, unknown> = {};
  graphModel = {
    nodes: [] as FakeElementModel[],
    edges: [] as FakeElementModel[],
    moveNode2Coordinate: (id: string, x: number, y: number) => {
      const model = this.getNodeModelById(id);
      if (model) Object.assign(model, { x, y });
    },
  };

  render(graph: LogicFlow.GraphConfigData) {
    this.renderCount += 1;
    this.graphModel.nodes = (graph.nodes ?? []).map((node) => ({
      id: String(node.id),
      x: node.x,
      y: node.y,
      properties: { ...(node.properties ?? {}) },
    }));
    this.graphModel.edges = (graph.edges ?? []).map((edge) => ({
      id: String(edge.id),
      x: 0,
      y: 0,
      text: typeof edge.text === 'string' ? edge.text : '',
      properties: { ...(edge.properties ?? {}) },
    }));
  }

  setProperties(id: string, properties: Record<string, unknown>) {
    const model = this.getModelById(id);
    if (model) model.properties = { ...model.properties, ...properties };
  }

  clearSelectElements() {
    this.selectedKey = null;
  }

  selectElementById(id: string) {
    this.selectedKey = id;
  }

  getModelById(id: string) {
    return this.getNodeModelById(id) ?? this.getEdgeModelById(id);
  }

  getNodeModelById(id: string) {
    return this.graphModel.nodes.find((node) => node.id === id);
  }

  getEdgeModelById(id: string) {
    return this.graphModel.edges.find((edge) => edge.id === id);
  }

  updateText(id: string, text: string) {
    const model = this.getEdgeModelById(id);
    if (model) model.text = text;
  }

  updateEditConfig(config: Record<string, unknown>) {
    this.editConfig = { ...this.editConfig, ...config };
  }
}

describe('workflow graph synchronization', () => {
  it('does not rebuild or move the graph when selection changes after a drag', async () => {
    const document = shallowRef(createWorkflowDocument());
    const selectedNodeKey = shallowRef<string | null>('start');
    const selectedEdgeKey = shallowRef<string | null>(null);
    const readonly = shallowRef(false);
    const fake = new FakeLogicFlow();
    const scope = effectScope();

    const sync = scope.run(() =>
      useWorkflowGraphSync({
        document: () => document.value,
        selectedNodeKey: () => selectedNodeKey.value,
        selectedEdgeKey: () => selectedEdgeKey.value,
        errorNodeKeys: () => undefined,
        errorEdgeKeys: () => undefined,
        viewMode: () => 'compact',
        readonly: () => readonly.value,
      }),
    )!;
    sync.initialize(fake as unknown as LogicFlow);

    sync.beginNodeDrag('start');
    fake.graphModel.moveNode2Coordinate('start', 760, 310);
    sync.finishNodeDrag('start', { x: 760, y: 310 });
    document.value = setNodePosition(document.value, 'start', { x: 760, y: 310 });
    selectedNodeKey.value = 'end';
    await nextTick();

    expect(fake.renderCount).toBe(1);
    expect(fake.getNodeModelById('start')).toMatchObject({ x: 760, y: 310 });
    expect(fake.selectedKey).toBe('end');
    expect(fake.getNodeModelById('end')?.properties.selected).toBe(true);
    scope.stop();
  });

  it('rebuilds only for topology changes and updates readonly mode incrementally', async () => {
    const document = shallowRef(createWorkflowDocument());
    const readonly = shallowRef(false);
    const fake = new FakeLogicFlow();
    const scope = effectScope();

    const sync = scope.run(() =>
      useWorkflowGraphSync({
        document: () => document.value,
        selectedNodeKey: () => 'start',
        selectedEdgeKey: () => null,
        errorNodeKeys: () => undefined,
        errorEdgeKeys: () => undefined,
        viewMode: () => 'compact',
        readonly: () => readonly.value,
      }),
    )!;
    sync.initialize(fake as unknown as LogicFlow);

    readonly.value = true;
    await nextTick();
    expect(fake.renderCount).toBe(1);
    expect(fake.editConfig).toMatchObject({ adjustNodePosition: false, hideAnchors: true });
    expect(fake.selectedKey).toBeNull();
    expect(fake.getNodeModelById('start')?.properties).toMatchObject({
      readonly: true,
      selected: false,
    });

    document.value = addNode(document.value, 'approval', { x: 650, y: 320 }).document;
    await nextTick();
    expect(fake.renderCount).toBe(2);
    expect(fake.getNodeModelById('approval_1')).toMatchObject({ x: 650, y: 320 });
    scope.stop();
  });
});
