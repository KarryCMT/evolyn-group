import { shallowRef } from 'vue';
import { describe, expect, it } from 'vitest';
import { createIntelligentDocument } from '../../schema';
import { useIntelligentDesigner } from '../useIntelligentDesigner';

describe('useIntelligentDesigner', () => {
  it('维护节点操作的撤销与重做历史', () => {
    const document = shallowRef(
      createIntelligentDocument({
        id: 'assistant_1',
        name: '员工同步',
        triggerFormName: '员工档案',
      }),
    );
    const designer = useIntelligentDesigner({
      getDocument: () => document.value,
      updateDocument: (nextDocument) => {
        document.value = nextDocument;
      },
    });

    designer.addNode();
    expect(document.value.nodes.map((node) => node.type)).toEqual([
      'trigger',
      'action',
      'end',
    ]);
    expect(designer.selectedAction.value).toBeUndefined();

    designer.undo();
    expect(document.value.nodes.map((node) => node.type)).toEqual(['trigger', 'end']);

    designer.redo();
    expect(document.value.nodes.map((node) => node.type)).toEqual([
      'trigger',
      'action',
      'end',
    ]);

    designer.selectedNodeId.value = document.value.nodes.find(
      (node) => node.type === 'action',
    )!.id;
    designer.deleteSelected();
    expect(document.value.nodes.map((node) => node.type)).toEqual(['trigger', 'end']);
  });

  it('把表单触发配置纳入撤销历史', () => {
    const document = shallowRef(
      createIntelligentDocument({
        id: 'assistant_2',
        name: '员工变更通知',
        triggerFormName: '员工档案',
      }),
    );
    const designer = useIntelligentDesigner({
      getDocument: () => document.value,
      updateDocument: (nextDocument) => {
        document.value = nextDocument;
      },
    });

    designer.updateTrigger({
      conditionMode: 'any',
      conditions: [
        {
          id: 'trigger_condition_department',
          fieldId: 'department',
          operator: 'equals-any',
          values: ['产品中心'],
        },
      ],
    });

    expect(document.value.trigger.conditionMode).toBe('any');
    expect(document.value.trigger.conditions).toHaveLength(1);

    designer.undo();
    expect(document.value.trigger.conditionMode).toBe('all');
    expect(document.value.trigger.conditions).toEqual([]);
  });

  it('从连线中间插入后保持画布浏览态', () => {
    const document = shallowRef(
      createIntelligentDocument({ id: 'assistant_edge', name: '连线插入' }),
    );
    const designer = useIntelligentDesigner({
      getDocument: () => document.value,
      updateDocument: (nextDocument) => {
        document.value = nextDocument;
      },
    });

    designer.addNode('update-record');
    const previousActionId = document.value.nodes.find((node) => node.type === 'action')?.id;
    const trigger = document.value.nodes.find((node) => node.type === 'trigger')!;
    const edgeBeforeAction = document.value.edges.find(
      (edge) => edge.source === trigger.id && edge.target === previousActionId,
    )!;
    designer.addNode('create-record', edgeBeforeAction.id);

    expect(document.value.nodes.filter((node) => node.type === 'action')).toHaveLength(2);
    expect(designer.selectedNodeId.value).toBeNull();
    expect(designer.selectedAction.value).toBeUndefined();
  });
});
