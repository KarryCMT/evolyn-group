import { describe, expect, it } from 'vitest';
import { createRuntimeSessionState } from '../session';

describe('Runtime Engine session', () => {
  it('creates isolated, mutable state for a domain runtime to own', () => {
    const first = createRuntimeSessionState<string, { visible: boolean }, string, 'ready', 'save'>(
      'ready',
    );
    const second = createRuntimeSessionState<string, { visible: boolean }, string, 'ready', 'save'>(
      'ready',
    );
    first.dirtyKeys.add('amount');

    expect(first.lifecycle).toBe('ready');
    expect(second.dirtyKeys.size).toBe(0);
  });
});
