import { describe, expect, it } from 'vitest';
import {
  addActionNode,
  createIntelligentDocument,
  moveIntelligentNode,
  removeActionNode,
  updateIntelligentNode,
  updateIntelligentTrigger,
} from '../document';

describe('intelligent document', () => {
  it('创建最小画布并支持执行节点的增删与移动', () => {
    const initial = createIntelligentDocument({
      id: 'assistant_1',
      name: '员工同步',
      triggerFormCode: 'form_employee',
      triggerFormName: '员工档案',
    });

    expect(initial.nodes.map((node) => node.type)).toEqual(['trigger', 'end']);
    expect(initial.edges).toHaveLength(1);

    const withAction = addActionNode(initial);
    const action = withAction.nodes.find((node) => node.type === 'action')!;
    expect(withAction.nodes.map((node) => node.type)).toEqual(['trigger', 'action', 'end']);
    expect(withAction.edges).toHaveLength(2);

    const moved = moveIntelligentNode(withAction, action.id, { x: 720, y: 360 });
    expect(moved.nodes.find((node) => node.id === action.id)?.position).toEqual({ x: 720, y: 360 });

    const removed = removeActionNode(moved, action.id);
    expect(removed.nodes.map((node) => node.type)).toEqual(['trigger', 'end']);
    expect(removed.edges).toHaveLength(1);
    expect(removed.nodes.find((node) => node.type === 'end')?.position.y).toBe(
      initial.nodes.find((node) => node.type === 'end')?.position.y,
    );
  });

  it('保留执行节点类型与属性面板配置', () => {
    const initial = createIntelligentDocument({ id: 'assistant_2', name: '接口同步' });
    const withAction = addActionNode(initial, {
      actionType: 'http-request',
      name: '发送 HTTP 请求',
      config: { requestMethod: 'POST' },
    });
    const action = withAction.nodes.find((node) => node.type === 'action')!;
    const updated = updateIntelligentNode(withAction, action.id, {
      config: { ...action.config, requestUrl: 'https://api.lingyanyun.com/sync' },
    });

    expect(updated.nodes.find((node) => node.id === action.id)).toMatchObject({
      actionType: 'http-request',
      name: '发送 HTTP 请求',
      config: {
        requestMethod: 'POST',
        requestUrl: 'https://api.lingyanyun.com/sync',
      },
    });
  });

  it('更新表单触发动作与条件并同步画布节点摘要', () => {
    const initial = createIntelligentDocument({
      id: 'assistant_3',
      name: '员工变更通知',
      triggerFormName: '员工档案',
    });
    const updated = updateIntelligentTrigger(initial, {
      actions: [
        {
          id: 'trigger_action_update',
          type: 'update',
          updateScope: 'any-field',
          fieldIds: [],
        },
      ],
      conditionMode: 'any',
      conditions: [
        {
          id: 'trigger_condition_gender',
          fieldId: 'gender',
          operator: 'equals-any',
          values: ['女'],
        },
      ],
    });

    expect(initial.trigger.actions.map((action) => action.type)).toEqual([
      'create',
      'update',
      'delete',
    ]);
    expect(updated.trigger).toMatchObject({
      conditionMode: 'any',
      conditions: [
        {
          fieldId: 'gender',
          operator: 'equals-any',
          values: ['女'],
        },
      ],
    });
    expect(updated.nodes.find((node) => node.type === 'trigger')?.description).toBe('修改数据时');
  });
});
