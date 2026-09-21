import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DataLinkageDefinition, FormJsonValue } from '../../schema/types';
import { DataLinkageRuntime } from '../linkage/DataLinkageRuntime';

function rule(overrides: Partial<DataLinkageDefinition['runtime']> = {}): DataLinkageDefinition {
  return {
    id: 'linkage_product',
    version: 1,
    enabled: true,
    source: { type: 'form', appId: 1, sourceId: 'form_product' },
    filter: {
      logic: 'and',
      conditions: [
        {
          id: 'cond_product',
          sourceFieldId: '_widget_source_name',
          operator: 'eq',
          value: { type: 'field', fieldId: '_widget_current_name' },
        },
      ],
    },
    mappings: [{ sourceFieldId: '_widget_source_price', targetFieldId: '_widget_price' }],
    result: { mode: 'first' },
    runtime: {
      trigger: 'dependency_change',
      runOnInit: false,
      debounceMs: 20,
      emptyStrategy: 'clear',
      errorStrategy: 'keep',
      ...overrides,
    },
  };
}

afterEach(() => vi.useRealTimers());

describe('DataLinkageRuntime', () => {
  it('输入变化立即取消旧请求，并且只有最新版本可以回填', async () => {
    vi.useFakeTimers();
    let current: FormJsonValue = '产品 A';
    const requests: Array<{
      requestVersion: number;
      signal: AbortSignal;
      resolve: (value: {
        ruleId: string;
        requestVersion: number;
        matched: boolean;
        values: Record<string, FormJsonValue>;
      }) => void;
    }> = [];
    const applied: Array<Record<string, FormJsonValue>> = [];
    const runtime = new DataLinkageRuntime({
      rules: [rule()],
      formId: 'form_order',
      schemaVersion: 3,
      adapter: {
        executeLinkage(request, signal) {
          return new Promise((resolve) =>
            requests.push({ requestVersion: request.requestVersion, signal, resolve }),
          );
        },
      },
      valueOf: () => current,
      applyValues: (values) => applied.push(values),
      clearTargets: vi.fn(),
    });

    runtime.notifyFieldChange('_widget_current_name');
    await vi.advanceTimersByTimeAsync(20);
    expect(requests).toHaveLength(1);

    current = '产品 B';
    runtime.notifyFieldChange('_widget_current_name');
    expect(requests[0]!.signal.aborted).toBe(true);
    requests[0]!.resolve({
      ruleId: 'linkage_product',
      requestVersion: requests[0]!.requestVersion,
      matched: true,
      values: { _widget_price: '1.00' },
    });
    await Promise.resolve();
    expect(applied).toEqual([]);

    await vi.advanceTimersByTimeAsync(20);
    requests[1]!.resolve({
      ruleId: 'linkage_product',
      requestVersion: requests[1]!.requestVersion,
      matched: true,
      values: { _widget_price: '2.00' },
    });
    await Promise.resolve();
    expect(applied).toEqual([{ _widget_price: '2.00' }]);
  });

  it('依赖为空时不请求后端并按策略清空目标字段', async () => {
    vi.useFakeTimers();
    const executeLinkage = vi.fn();
    const clearTargets = vi.fn();
    const runtime = new DataLinkageRuntime({
      rules: [rule()],
      formId: 'form_order',
      schemaVersion: 3,
      adapter: { executeLinkage },
      valueOf: () => '',
      applyValues: vi.fn(),
      clearTargets,
    });

    runtime.notifyFieldChange('_widget_current_name');
    await vi.advanceTimersByTimeAsync(20);
    expect(executeLinkage).not.toHaveBeenCalled();
    expect(clearTargets).toHaveBeenCalledOnce();
  });
});
