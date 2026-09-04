import { describe, expect, it } from 'vitest';
import { createValidationResult } from '../result';

describe('Validator Engine result', () => {
  it('returns a null value whenever diagnostics exist', () => {
    const result = createValidationResult([{ path: 'content.type', message: '类型错误' }], {
      type: 'form',
    });

    expect(result).toEqual({
      valid: false,
      value: null,
      issues: [{ path: 'content.type', message: '类型错误' }],
    });
  });
});
