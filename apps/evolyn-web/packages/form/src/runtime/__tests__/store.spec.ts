import { describe, expect, it } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import { createFormRuntime } from '../store/createFormRuntime';
import type { FormSubmitResult } from '../types';

/** 构造字段项（默认必填可选）。 */
function item(
  widget: Record<string, unknown>,
  extras: Partial<Pick<FormItem, 'label' | 'lineWidth'>> = {},
): FormItem {
  return {
    widget: {
      enable: true,
      visible: true,
      allowBlank: true,
      ...widget,
    } as FormItem['widget'],
    label: extras.label ?? '字段',
    description: '',
    labelHidden: false,
    lineWidth: extras.lineWidth ?? 12,
  };
}

function documentOf(items: FormItem[]): FormSchemaDocument {
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items,
      layout_fields: [],
      field_layout: items.map((item) => item.widget.widgetName),
    },
  };
}

describe('createFormRuntime 初始化', () => {
  it('布局项不进入值表；多选空值为 []、其余为 null', () => {
    const runtime = createFormRuntime({
      schema: documentOf([
        item({ type: 'separator', widgetName: '_widget_sep' }),
        item({ type: 'text', widgetName: '_widget_t' }),
        item({
          type: 'combocheck',
          widgetName: '_widget_m',
          options: [{ label: 'A', value: 'a' }],
        }),
      ]),
    });
    expect(runtime.state.values).toEqual({ _widget_t: null, _widget_m: [] });
    expect(runtime.state.fieldStates['_widget_sep']).toBeUndefined();
  });

  it('初始化优先级：已保存值 → 上下文默认值 → 类型化空值（Schema defaultValue 随 P5 执行）', () => {
    const schema = documentOf([
      item({ type: 'text', widgetName: '_widget_t', defaultValue: 'schema' }),
    ]);
    const runtime = createFormRuntime({
      schema,
      initialValues: { _widget_t: 'saved' },
      contextDefaults: { _widget_t: 'context' },
    });
    expect(runtime.state.values._widget_t).toBe('saved');

    const runtime2 = createFormRuntime({ schema, contextDefaults: { _widget_t: 'context' } });
    expect(runtime2.state.values._widget_t).toBe('context');

    const runtime3 = createFormRuntime({ schema });
    expect(runtime3.state.values._widget_t).toBeNull();
  });

  it('enable=false 映射禁用；visible=false 不渲染不收集', () => {
    const runtime = createFormRuntime({
      schema: documentOf([
        item({ type: 'text', widgetName: '_widget_disabled', enable: false }),
        item({ type: 'text', widgetName: '_widget_hidden', visible: false }),
      ]),
    });
    expect(runtime.state.fieldStates._widget_disabled.disabled).toBe(true);
    expect(runtime.state.fieldStates._widget_hidden.visible).toBe(false);
    const payload = runtime.buildSubmitPayload();
    expect(payload.values).toEqual({
      _widget_disabled: { visible: true },
      _widget_hidden: { visible: false },
    });
  });
});

describe('setValue 与校验', () => {
  it('写入归一化值并计入脏状态；已出错字段输入时即时重校验', () => {
    const runtime = createFormRuntime({
      schema: documentOf([item({ type: 'text', widgetName: '_widget_t', allowBlank: false })]),
    });
    // 触发校验产生必填错误，随后用户输入立即清除错误（交互即时反馈）。
    expect(runtime.validateField('_widget_t')).toEqual(['请输入字段']);
    runtime.setValue('_widget_t', 'x');
    expect(runtime.state.values._widget_t).toBe('x');
    expect(runtime.isDirty()).toBe(true);
    expect(runtime.state.fieldStates._widget_t.errors).toEqual([]);
  });

  it('隐藏字段不校验、不产生错误', () => {
    const runtime = createFormRuntime({
      schema: documentOf([
        item({ type: 'text', widgetName: '_widget_h', allowBlank: false, visible: false }),
      ]),
    });
    expect(runtime.validateField('_widget_h')).toEqual([]);
    expect(runtime.validateVisibleFields()).toBe(true);
  });

  it('validateVisibleFields 聚合本地字段错误并保留服务端非字段错误', () => {
    const runtime = createFormRuntime({
      schema: documentOf([
        item({ type: 'text', widgetName: '_widget_a', allowBlank: false, label: '姓名' }),
        item({ type: 'number', widgetName: '_widget_b', allowBlank: false, label: '年龄' }),
      ]),
    });
    expect(runtime.validateVisibleFields()).toBe(false);
    expect(runtime.state.issues.map((issue) => issue.fieldKey)).toEqual(['_widget_a', '_widget_b']);
    runtime.addServerIssue('服务端错误');
    runtime.validateVisibleFields();
    expect(runtime.state.issues.some((issue) => issue.message === '服务端错误')).toBe(true);
  });
});

describe('提交', () => {
  const schema = documentOf([
    item({ type: 'text', widgetName: '_widget_t', allowBlank: false }),
    item({
      type: 'combocheck',
      widgetName: '_widget_m',
      options: [{ label: 'A', value: 'a' }],
    }),
    item({ type: 'separator', widgetName: '_widget_sep' }),
  ]);

  it('无 adapter 时仅产出载荷；载荷含双口令与按 widgetName 的值快照', async () => {
    const runtime = createFormRuntime({
      schema,
      formId: '9',
      publishedVersion: 3,
      schemaRevision: '77',
    });
    runtime.setValue('_widget_t', 'ok');
    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(true);
    if (!outcome.ok) return;
    expect(outcome.submitted).toBe(false);
    expect(outcome.payload).toEqual({
      formId: '9',
      publishedVersion: 3,
      schemaRevision: '77',
      values: {
        _widget_t: { data: 'ok', visible: true },
        _widget_m: { data: [], visible: true },
      },
    });
  });

  it('本地校验失败拒绝提交（invalid）', async () => {
    const runtime = createFormRuntime({
      schema,
      adapter: { submit: async () => ({ accepted: true }) },
    });
    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(false);
    if (!outcome.ok) expect(outcome.reason).toBe('invalid');
  });

  it('服务端字段错误按 widgetName 回填并恢复 ready 生命周期', async () => {
    const rejected: FormSubmitResult = {
      accepted: false,
      fieldErrors: { _widget_m: ['选项已失效'] },
      message: '请修正后重试',
    };
    const runtime = createFormRuntime({
      schema,
      adapter: { submit: async () => rejected },
    });
    runtime.setValue('_widget_t', 'ok');
    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(false);
    if (outcome.ok) return;
    expect(outcome.reason).toBe('server');
    expect(runtime.state.fieldStates._widget_m.errors).toEqual(['选项已失效']);
    expect(runtime.state.lifecycle).toBe('ready');
    expect(runtime.state.activeOperation).toBeNull();
  });

  it('adapter 抛 AbortError 恢复 ready 态（取消不是业务错误）', async () => {
    const runtime = createFormRuntime({
      schema,
      adapter: {
        submit: () => {
          const abort = new Error('aborted');
          abort.name = 'AbortError';
          return Promise.reject(abort);
        },
      },
    });
    runtime.setValue('_widget_t', 'ok');
    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(false);
    if (!outcome.ok) expect(outcome.reason).toBe('cancelled');
    expect(runtime.state.lifecycle).toBe('ready');
    expect(runtime.state.activeOperation).toBeNull();
  });

  it('兼容 Axios 的 ERR_CANCELED 取消错误', async () => {
    const runtime = createFormRuntime({
      schema,
      adapter: {
        submit: async () => Promise.reject({ code: 'ERR_CANCELED' }),
      },
    });
    runtime.setValue('_widget_t', 'ok');

    const outcome = await runtime.submit();

    expect(outcome.ok).toBe(false);
    if (!outcome.ok) expect(outcome.reason).toBe('cancelled');
    expect(runtime.state.issues).toEqual([]);
    expect(runtime.state.lifecycle).toBe('ready');
  });
});

describe('填写草稿', () => {
  const schema = documentOf([
    item({ type: 'text', widgetName: '_widget_visible', allowBlank: false }),
    item({ type: 'text', widgetName: '_widget_hidden', visible: false }),
  ]);

  it('不执行必填校验并保存全部字段及版本口令', async () => {
    let received: unknown;
    const runtime = createFormRuntime({
      schema,
      formId: 'form_a',
      publishedVersion: 2,
      schemaRevision: 'revision-2',
      initialValues: { _widget_hidden: '保留值' },
      adapter: {
        saveDraft: async (payload) => {
          received = payload;
        },
      },
    });
    runtime.setValue('_widget_visible', '未完成');

    const outcome = await runtime.saveDraft();

    expect(outcome.ok).toBe(true);
    expect(received).toEqual({
      formId: 'form_a',
      publishedVersion: 2,
      schemaRevision: 'revision-2',
      values: { _widget_visible: '未完成', _widget_hidden: '保留值' },
    });
    expect(runtime.isDirty()).toBe(false);
  });

  it('adapter 缺失时返回 unavailable，不伪装保存成功', async () => {
    const runtime = createFormRuntime({ schema });
    const outcome = await runtime.saveDraft();
    expect(outcome.ok).toBe(false);
    if (!outcome.ok) expect(outcome.reason).toBe('unavailable');
  });

  it('提交与保存草稿互斥，避免并发持久化', async () => {
    let release: (() => void) | undefined;
    const runtime = createFormRuntime({
      schema: documentOf([item({ type: 'text', widgetName: '_widget_t' })]),
      adapter: {
        saveDraft: () =>
          new Promise<void>((resolve) => {
            release = resolve;
          }),
        submit: async () => ({ accepted: true }),
      },
    });

    const saving = runtime.saveDraft();
    expect(runtime.state.activeOperation).toBe('save-draft');
    const submitOutcome = await runtime.submit();
    expect(submitOutcome.ok).toBe(false);
    if (!submitOutcome.ok) expect(submitOutcome.reason).toBe('busy');

    release?.();
    await saving;
    expect(runtime.state.activeOperation).toBeNull();
  });
});
