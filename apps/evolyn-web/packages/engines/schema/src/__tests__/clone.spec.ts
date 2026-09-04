import { describe, expect, it } from 'vitest';
import { cloneJsonValue } from '../clone';

describe('Schema Engine clone', () => {
  it('returns a detached copy of JSON-safe schema data', () => {
    const source = { content: { fields: [{ name: 'amount' }] } };
    const clone = cloneJsonValue(source);
    clone.content.fields[0]!.name = 'total';

    expect(source.content.fields[0]!.name).toBe('amount');
  });
});
