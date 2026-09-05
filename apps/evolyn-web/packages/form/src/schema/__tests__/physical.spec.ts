import { describe, expect, it } from 'vitest';
import { projectPhysicalStorageFields } from '../physical';
import type { FormItem } from '../types';

describe('form physical projection', () => {
  it('uses widgetName as the stable logical storage identifier', () => {
    const items = [
      {
        label: '合同金额',
        widget: { type: 'number', widgetName: '_widget_amount', allowBlank: false },
      },
      { label: '', widget: { type: 'separator', widgetName: '_widget_divider', allowBlank: true } },
    ] as FormItem[];

    expect(projectPhysicalStorageFields(items)).toEqual([
      { id: '_widget_amount', type: 'decimal', required: true },
    ]);
  });
});
