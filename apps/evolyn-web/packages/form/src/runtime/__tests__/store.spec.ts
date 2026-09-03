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
      fieldShowRules: [],
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
    expect(runtime.state.fieldStates._widget_disabled!.disabled).toBe(true);
    expect(runtime.state.fieldStates._widget_hidden!.visible).toBe(false);
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
    expect(runtime.state.fieldStates._widget_t!.errors).toEqual([]);
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
    expect(runtime.state.fieldStates._widget_m!.errors).toEqual(['选项已失效']);
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

// ---- v5 字段显隐规则：运行时可见性引擎 ----

function rulesDocumentOf(
  items: FormItem[],
  fieldShowRules: FormSchemaDocument['content']['fieldShowRules'],
): FormSchemaDocument {
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items,
      layout_fields: [],
      field_layout: items.map((entry) => entry.widget.widgetName),
      fieldShowRules,
    },
  };
}

describe('createFormRuntime 显隐规则引擎', () => {
  const outingForm = () =>
    rulesDocumentOf(
      [
        item(
          {
            type: 'radiogroup',
            widgetName: '_widget_out',
            options: [
              { label: '是', value: 'yes' },
              { label: '否', value: 'no' },
            ],
          },
          { label: '是否外出' },
        ),
        item({ type: 'text', widgetName: '_widget_city' }, { label: '外出城市' }),
        item({ type: 'text', widgetName: '_widget_note' }, { label: '住宿说明' }),
      ],
      [
        {
          id: '_rule_city',
          filter: {
            rel: 'and',
            cond: [{ field: '_widget_out', type: 'radiogroup', method: 'eq', value: ['yes'] }],
          },
          fields: ['_widget_city'],
        },
        {
          id: '_rule_note',
          filter: {
            rel: 'and',
            cond: [{ field: '_widget_city', type: 'text', method: 'notEmpty' }],
          },
          fields: ['_widget_note'],
        },
      ],
    );

  it('初始化即按规则求值：未满足条件的目标字段隐藏', () => {
    const runtime = createFormRuntime({ schema: outingForm() });
    expect(runtime.state.fieldStates['_widget_out']!.visible).toBe(true);
    expect(runtime.state.fieldStates['_widget_city']!.visible).toBe(false);
    expect(runtime.state.fieldStates['_widget_note']!.visible).toBe(false);
  });

  it('值变化仅重算下游闭包并多级传播：外出→城市→说明', () => {
    const runtime = createFormRuntime({ schema: outingForm() });
    runtime.setValue('_widget_out', 'yes');
    expect(runtime.state.fieldStates['_widget_city']!.visible).toBe(true);
    expect(runtime.state.fieldStates['_widget_note']!.visible).toBe(false);

    runtime.setValue('_widget_city', '上海');
    expect(runtime.state.fieldStates['_widget_note']!.visible).toBe(true);

    runtime.setValue('_widget_out', 'no');
    // 上游隐藏时下游自然隐藏，且已填值保留（再次显示恢复）。
    expect(runtime.state.fieldStates['_widget_city']!.visible).toBe(false);
    expect(runtime.state.fieldStates['_widget_note']!.visible).toBe(false);
    expect(runtime.state.values['_widget_city']).toBe('上海');

    runtime.setValue('_widget_out', 'yes');
    expect(runtime.state.fieldStates['_widget_city']!.visible).toBe(true);
    expect(runtime.state.values['_widget_city']).toBe('上海');
    expect(runtime.state.fieldStates['_widget_note']!.visible).toBe(true);
  });

  it('规则隐藏字段跳过必填校验、不进入提交载荷，草稿保留全部值', async () => {
    const requiredForm = () =>
      rulesDocumentOf(
        [
          item(
            {
              type: 'radiogroup',
              widgetName: '_widget_out',
              options: [
                { label: '是', value: 'yes' },
                { label: '否', value: 'no' },
              ],
            },
            { label: '是否外出' },
          ),
          item(
            { type: 'text', widgetName: '_widget_city', allowBlank: false },
            { label: '外出城市' },
          ),
        ],
        [
          {
            id: '_rule_city',
            filter: {
              rel: 'and',
              cond: [{ field: '_widget_out', type: 'radiogroup', method: 'eq', value: ['yes'] }],
            },
            fields: ['_widget_city'],
          },
        ],
      );
    const runtime = createFormRuntime({ schema: requiredForm() });
    runtime.setValue('_widget_out', 'no');
    runtime.markTouched('_widget_city');
    // 隐藏的必填字段不阻塞提交。
    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(true);
    if (!outcome.ok) return;
    expect(outcome.payload.values['_widget_city']).toEqual({ visible: false });

    // 显式填写后进入载荷。
    runtime.setValue('_widget_out', 'yes');
    runtime.setValue('_widget_city', '杭州');
    const second = await runtime.submit();
    expect(second.ok).toBe(true);
    if (!second.ok) return;
    expect(second.payload.values['_widget_city']).toEqual({ visible: true, data: '杭州' });

    // 草稿保留隐藏字段值，恢复填写后可继续使用。
    const draft = runtime.buildDraftPayload();
    expect(draft.values['_widget_city']).toBe('杭州');
  });

  it('includeCurrentMember 经 currentMemberId 注入比较集合', () => {
    const memberForm = () =>
      rulesDocumentOf(
        [
          item({ type: 'user', widgetName: '_widget_owner' }, { label: '负责人' }),
          item({ type: 'text', widgetName: '_widget_secret' }, { label: '专属项' }),
        ],
        [
          {
            id: '_rule_secret',
            filter: {
              rel: 'and',
              cond: [
                {
                  field: '_widget_owner',
                  type: 'user',
                  method: 'eq',
                  value: ['member_a'],
                  includeCurrentMember: true,
                },
              ],
            },
            fields: ['_widget_secret'],
          },
        ],
      );
    const anonymous = createFormRuntime({ schema: memberForm() });
    anonymous.setValue('_widget_owner', 'member_current');
    expect(anonymous.state.fieldStates['_widget_secret']!.visible).toBe(false);

    const named = createFormRuntime({ schema: memberForm(), currentMemberId: 'member_current' });
    expect(named.state.fieldStates['_widget_secret']!.visible).toBe(false);
    named.setValue('_widget_owner', 'member_current');
    expect(named.state.fieldStates['_widget_secret']!.visible).toBe(true);
  });

  it('静态隐藏字段与规则合成为交集', () => {
    const runtime = createFormRuntime({
      schema: rulesDocumentOf(
        [
          item({ type: 'text', widgetName: '_widget_src' }),
          item({ type: 'text', widgetName: '_widget_target', visible: false }),
        ],
        [
          {
            id: '_rule_t',
            filter: {
              rel: 'and',
              cond: [{ field: '_widget_src', type: 'text', method: 'eq', value: ['甲'] }],
            },
            fields: ['_widget_target'],
          },
        ],
      ),
    });
    runtime.setValue('_widget_src', '甲');
    expect(runtime.state.fieldStates['_widget_target']!.visible).toBe(false);
  });
});

describe('createFormRuntime 字段权限合成（v5）', () => {
  const permissionForm = () =>
    rulesDocumentOf(
      [
        item({ type: 'text', widgetName: '_widget_src' }),
        item({ type: 'text', widgetName: '_widget_target' }, { label: '目标' }),
        item({ type: 'text', widgetName: '_widget_limited' }, { label: '受限字段' }),
      ],
      [
        {
          id: '_rule_target',
          filter: {
            rel: 'and',
            cond: [{ field: '_widget_src', type: 'text', method: 'eq', value: ['甲'] }],
          },
          fields: ['_widget_target'],
        },
      ],
    );
  const matrix = {
    _widget_src: { visible: false, editable: false },
    _widget_target: { visible: true, editable: true },
    _widget_limited: { visible: true, editable: false },
  };

  it('权限隐藏的条件源条件视为不成立，下游不显示（防推断）', () => {
    const runtime = createFormRuntime({
      schema: permissionForm(),
      fieldPermissions: matrix,
      initialValues: { _widget_src: '甲' },
    });
    // 条件源值存在但权限不可见：不得读取，目标保持隐藏。
    expect(runtime.state.fieldStates['_widget_target']!.visible).toBe(false);
  });

  it('权限合成为交集：权限隐藏字段渲染不可见', () => {
    const runtime = createFormRuntime({
      schema: permissionForm(),
      fieldPermissions: matrix,
    });
    expect(runtime.state.fieldStates['_widget_src']!.visible).toBe(false);
  });

  it('权限不可编辑字段禁用且不进入提交 data；信封 visible 与权限解耦', async () => {
    const runtime = createFormRuntime({
      schema: permissionForm(),
      fieldPermissions: { ...matrix, _widget_src: { visible: true, editable: true } },
      initialValues: { _widget_src: '甲', _widget_limited: '旧值' },
    });
    expect(runtime.state.fieldStates['_widget_limited']!.disabled).toBe(true);
    expect(runtime.state.fieldStates['_widget_target']!.visible).toBe(true);

    const outcome = await runtime.submit();
    expect(outcome.ok).toBe(true);
    // 受限字段携带信封口径 visible=true + 无 data（服务端权限管线复核回填）。
    expect(outcome.ok && outcome.payload.values['_widget_limited']).toEqual({ visible: true });
    // 正常字段照常携带 data。
    expect(outcome.ok && outcome.payload.values['_widget_src']).toEqual({
      visible: true,
      data: '甲',
    });
  });

  it('未提供权限矩阵时全量放行（预览/草稿回放口径）', () => {
    const runtime = createFormRuntime({
      schema: permissionForm(),
      initialValues: { _widget_src: '甲' },
    });
    expect(runtime.state.fieldStates['_widget_target']!.visible).toBe(true);
    expect(runtime.state.fieldStates['_widget_target']!.envelopeVisible).toBe(true);
  });
});
