import { describe, expect, it } from 'vitest';
import { createFieldRegistry } from '../registry';

describe('Field Engine registry', () => {
  it('exposes a stable, read-only field definition lookup', () => {
    const registry = createFieldRegistry({ text: { label: '文本' }, number: { label: '数字' } });

    expect(registry.types).toEqual(['text', 'number']);
    expect(registry.has('text')).toBe(true);
    expect(registry.has('missing')).toBe(false);
    expect(registry.get('number')).toEqual({ label: '数字' });
    expect(registry.find('missing')).toBeUndefined();
  });
});
