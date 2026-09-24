import { describe, expect, it, vi } from 'vitest';
import { createIntelligentDocument } from '../../schema';
import { IntelligentContext } from '../IntelligentContext';

describe('IntelligentContext', () => {
  it('集中维护设计器事件和可撤销文档历史', () => {
    const initial = createIntelligentDocument({ id: 'assistant_1', name: '员工同步' });
    const next = { ...initial, name: '员工入职同步' };
    const context = new IntelligentContext(initial);
    const listener = vi.fn();

    context.events.on('document:change', listener);
    context.events.emit('document:change', next);
    context.history.add(next);

    expect(listener).toHaveBeenCalledWith(next);
    expect(context.history.undo().current?.name).toBe('员工同步');
    expect(context.history.redo().current?.name).toBe('员工入职同步');

    context.destroy();
    context.events.emit('document:change', initial);
    expect(listener).toHaveBeenCalledTimes(1);
  });
});
