import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import type { QueryDiagnostic, QueryDocument, QueryFieldCapability } from '../types.js';
import {
  buildQueryLogicalPlan,
  composeQueryFilters,
  normalizeQuery,
  serializeQuery,
  validateQuery,
} from '../index.js';

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

    expect(filter).toMatchObject({
      type: 'group',
      children: [{ operator: 'gt' }, { operator: 'lt' }],
    });
    expect(result.diagnostics).toEqual([]);
    expect(result.document).not.toBeNull();
  });

  it('builds a storage-independent aggregate plan', () => {
    const document = normalizeQuery({
      groupBy: ['region'],
      aggregates: [{ field: 'amount', operator: 'sum', alias: 'totalAmount' }],
      sorts: [{ field: 'totalAmount', direction: 'desc' }],
    });
    expect(buildQueryLogicalPlan(document)).toMatchObject({
      aggregate: true,
      groupBy: ['region'],
      aggregates: [{ alias: 'totalAmount' }],
      complexity: { depth: 0, conditions: 0 },
    });
  });
});

interface QueryContractVector {
  name: string;
  query: unknown;
  expectedIssues: Array<Pick<QueryDiagnostic, 'path' | 'code'>>;
  expectedNormalized?: QueryDocument;
}

const vectorsFile = join(
  fileURLToPath(import.meta.url),
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  'docs',
  'contracts',
  'dashboard-query-test-vectors.json',
);
const contract = JSON.parse(readFileSync(vectorsFile, 'utf-8')) as {
  fields: Record<string, QueryFieldCapability>;
  vectors: QueryContractVector[];
};

describe('dashboard query cross-runtime contract', () => {
  for (const vector of contract.vectors) {
    it(vector.name, () => {
      const result = validateQuery(vector.query, { fields: contract.fields });
      expect(result.diagnostics.map(({ path, code }) => ({ path, code }))).toEqual(
        vector.expectedIssues,
      );
      expect(result.document ?? undefined).toEqual(vector.expectedNormalized);
    });
  }
});
