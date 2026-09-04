import { describe, expect, it } from 'vitest';
import { createDataContext } from '../context.js';
import { normalizeDataQuery } from '../query.js';

describe('Data Engine', () => {
  it('normalizes partial queries without any UI runtime', () => {
    expect(normalizeDataQuery({ keyword: '  contract  ', page: 0, pageSize: 8.9 })).toEqual({
      keyword: 'contract',
      page: 1,
      pageSize: 8,
    });
  });

  it('creates an immutable, resource-scoped context', () => {
    const context = createDataContext(' form-records ', { tenantId: 'tenant_1' });

    expect(context).toEqual({ resource: 'form-records', metadata: { tenantId: 'tenant_1' } });
    expect(Object.isFrozen(context)).toBe(true);
  });
});
