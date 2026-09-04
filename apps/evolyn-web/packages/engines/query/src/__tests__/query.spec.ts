import { describe, expect, it } from 'vitest';
import { composeQueryFilters, normalizeQuery, serializeQuery, validateQuery } from '../index.js';

describe('Query Engine', () => {
  it('normalizes a serializable query document', () => {
    const query = normalizeQuery({
      filter: { type: 'condition', field: ' status ', operator: 'eq', value: 'active' },
      sorts: [{ field: ' createdAt ', direction: 'desc' }],
      paging: { page: 1.8, pageSize: 0 },
      projection: [' id ', 'id', 'name'],
    });

    expect(query).toMatchObject({
      version: 1,
      filter: { type: 'condition', field: 'status', operator: 'eq', value: 'active' },
      sorts: [{ field: 'createdAt', direction: 'desc' }],
      paging: { page: 1, pageSize: 20 },
      projection: ['id', 'name'],
    });
    expect(serializeQuery(query)).toBe(serializeQuery(query));
  });

  it('flattens matching condition groups and validates field capabilities', () => {
    const filter = composeQueryFilters('and', [
      { type: 'condition', field: 'amount', operator: 'gt', value: 100 },
      {
        type: 'group',
        conjunction: 'and',
        children: [{ type: 'condition', field: 'amount', operator: 'lt', value: 500 }],
      },
    ]);
    const result = validateQuery(
      { version: 1, filter, sorts: [], paging: { page: 1, pageSize: 20 } },
      { fieldTypes: { amount: 'number' } },
    );

    expect(filter).toMatchObject({ type: 'group', children: [{ operator: 'gt' }, { operator: 'lt' }] });
    expect(result.diagnostics).toEqual([]);
    expect(result.document).not.toBeNull();
  });
});
