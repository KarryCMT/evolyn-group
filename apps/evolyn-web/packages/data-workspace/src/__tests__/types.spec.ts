import { describe, expect, it } from 'vitest';
import { flattenDataColumns } from '../types';

describe('data workspace column hierarchy', () => {
  it('exposes grouped-header children individually in column settings', () => {
    expect(
      flattenDataColumns([
        { field: 'orderNo', title: '订单号' },
        {
          title: '订单明细',
          columns: [
            { field: '__subform:lines:product', title: '商品' },
            { field: '__subform:lines:owner', title: '负责人' },
          ],
        },
      ]),
    ).toEqual([
      expect.objectContaining({ titlePath: '订单号' }),
      expect.objectContaining({ titlePath: '订单明细 / 商品' }),
      expect.objectContaining({ titlePath: '订单明细 / 负责人' }),
    ]);
  });
});
