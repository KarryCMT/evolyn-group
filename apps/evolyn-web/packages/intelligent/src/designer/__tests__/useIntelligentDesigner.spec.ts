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
    expect(designer.selectedAction.value?.type).toBe('action');

    designer.undo();
    expect(document.value.nodes.map((node) => node.type)).toEqual(['trigger', 'end']);

    designer.redo();
    expect(document.value.nodes.map((node) => node.type)).toEqual([
      'trigger',
      'action',
      'end',
    ]);

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
});
