import { describe, expect, it } from 'vitest';
import { createPhysicalModel, derivePhysicalColumnName, planPhysicalMigration } from '../index.js';

describe('Physical Engine', () => {
  it('derives a stable column from field id instead of a mutable field label', () => {
    expect(derivePhysicalColumnName('field_01J82')).toBe('f_01j82');

    const model = createPhysicalModel('app_order', [
      { id: 'field_01J82', type: 'decimal', required: true, indexed: true },
    ]);
    expect(model.columns[0]).toMatchObject({ column: 'f_01j82', storageType: 'numeric', nullable: false });
    expect(model.indexes[0]?.name).toBe('idx_app_order_f_01j82');
  });

  it('marks removed columns as deprecated instead of producing a destructive operation', () => {
    const previous = createPhysicalModel('app_order', [
      { id: 'field_amount', type: 'decimal' },
      { id: 'field_status', type: 'shortText' },
    ]);
    const next = createPhysicalModel('app_order', [{ id: 'field_amount', type: 'decimal', required: true }]);

    expect(planPhysicalMigration(previous, next).operations).toEqual([
      expect.objectContaining({ type: 'alterColumn', after: expect.objectContaining({ nullable: false }) }),
      expect.objectContaining({ type: 'deprecateColumn', column: expect.objectContaining({ column: 'f_status' }) }),
    ]);
  });
});
