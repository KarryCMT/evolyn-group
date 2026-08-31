import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import FormRenderer from '../renderer/FormRenderer.vue';
import { buildRenderPlan } from '../renderer/plan';
import type { FormSubmitResult } from '../types';

function item(widget: Record<string, unknown>, extras: Partial<FormItem> = {}): FormItem {
  return {
    widget: {
      enable: true,
      visible: true,
      allowBlank: true,
      ...widget,
    } as FormItem['widget'],
    label: '字段',
    description: '',
    labelHidden: false,
    lineWidth: 12,
    ...extras,
  };
}

const schema = (items: FormItem[]): FormSchemaDocument => ({
  content: {
    type: 'form',
    layout: 'normal',
    items,
    layout_fields: [],
    field_layout: items.map((entry) => entry.widget.widgetName),
  },
});

describe('FormRenderer 渲染', () => {
  it('无效 Schema 渲染受控错误态（含路径），不创建会话', () => {
    const wrapper = mount(FormRenderer, {
      props: {
        schema: {
          content: {
            type: 'form',
            layout: 'normal',
            items: [{ nope: true } as never],
            layout_fields: [],
            field_layout: [],
          },
        },
      },
    });
    expect(wrapper.find('.evf-form__invalid').exists()).toBe(true);
    expect(wrapper.text()).toContain('content.items[0]');
    expect((wrapper.vm as { runtime: unknown }).runtime).toBeNull();
  });

  it('按 widget.type 分派组件并建立 aria 关联（label/describedby/error）', () => {
    const doc = schema([
      item(
        { type: 'text', widgetName: '_widget_t', allowBlank: false },
        { description: '请如实填写' },
      ),
    ]);
    const wrapper = mount(FormRenderer, { props: { schema: doc } });
    const input = wrapper.find('input#evf-field-_widget_t');
    expect(input.exists()).toBe(true);
    expect(wrapper.find('label[for="evf-field-_widget_t"]').text()).toContain('字段');
    expect(input.attributes('aria-required')).toBe('true');
    expect(input.attributes('aria-describedby')).toContain('evf-field-_widget_t-desc');
  });

  it('labelHidden 隐藏标签但保留控件与说明', () => {
    const doc = schema([item({ type: 'text', widgetName: '_widget_t' }, { labelHidden: true })]);
    const wrapper = mount(FormRenderer, { props: { schema: doc } });
    expect(wrapper.find('.evf-field__label').exists()).toBe(false);
    expect(wrapper.find('input').exists()).toBe(true);
  });

  it('visible=false 字段不渲染', () => {
    const doc = schema([item({ type: 'text', widgetName: '_widget_h', visible: false })]);
    const wrapper = mount(FormRenderer, { props: { schema: doc } });
    expect(wrapper.find('.evf-field').exists()).toBe(false);
  });

  it('分割线作为无值布局项仍按 Schema 的可见性渲染，并展示描述', () => {
    const visibleDivider = schema([
      item(
        { type: 'separator', widgetName: '_widget_sep' },
        { description: '<p>用于区分填写区块</p>' },
      ),
    ]);
    const visibleWrapper = mount(FormRenderer, { props: { schema: visibleDivider } });
    expect(visibleWrapper.find('[role="separator"]').exists()).toBe(true);
    expect(visibleWrapper.find('.evf-field__description').text()).toBe('用于区分填写区块');
    expect(
      visibleWrapper.find('.evf-field__description').element.compareDocumentPosition(
        visibleWrapper.find('[role="separator"]').element,
      ),
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);

    const hiddenLabelDivider = schema([
      item(
        { type: 'separator', widgetName: '_widget_hidden_label_sep', content: '独立分割线文案' },
        { label: '隐藏的通用标题', labelHidden: true },
      ),
    ]);
    const hiddenLabelWrapper = mount(FormRenderer, { props: { schema: hiddenLabelDivider } });
    expect(hiddenLabelWrapper.find('[role="separator"]').exists()).toBe(true);
    expect(hiddenLabelWrapper.text()).toContain('独立分割线文案');
    expect(hiddenLabelWrapper.text()).not.toContain('隐藏的通用标题');

    const hiddenDivider = schema([
      item({ type: 'separator', widgetName: '_widget_hidden_sep', visible: false }),
    ]);
    const hiddenWrapper = mount(FormRenderer, { props: { schema: hiddenDivider } });
    expect(hiddenWrapper.find('[role="separator"]').exists()).toBe(false);
  });

  it('未知控件类型渲染受控「暂不支持」并上报 unsupported-field', () => {
    const doc = schema([item({ type: 'user', widgetName: '_widget_u' })]);
    const wrapper = mount(FormRenderer, {
      props: { schema: doc },
    });
    expect(wrapper.text()).toContain('暂不支持填写');
  });

  it('必填空值拦截提交；输入后可通过并触发 adapter', async () => {
    const submit = vi.fn(async (): Promise<FormSubmitResult> => ({ accepted: true }));
    const doc = schema([item({ type: 'text', widgetName: '_widget_t', allowBlank: false })]);
    const wrapper = mount(FormRenderer, {
      props: { schema: doc, adapter: { submit } },
    });
    await wrapper.find('form').trigger('submit');
    expect(submit).not.toHaveBeenCalled();
    expect(wrapper.find('.evf-field__error').text()).toContain('请输入字段');

    await wrapper.find('input').setValue('ok');
    await wrapper.find('form').trigger('submit');
    expect(submit).toHaveBeenCalledTimes(1);
    const payload = submit.mock.calls[0][0];
    expect(payload.values).toEqual({ _widget_t: { data: 'ok', visible: true } });
  });

  it('服务端字段错误回填到对应字段', async () => {
    const submit = vi.fn(
      async (): Promise<FormSubmitResult> => ({
        accepted: false,
        fieldErrors: { _widget_t: ['已被占用'] },
      }),
    );
    const doc = schema([item({ type: 'text', widgetName: '_widget_t' })]);
    const wrapper = mount(FormRenderer, {
      props: { schema: doc, adapter: { submit } },
    });
    await wrapper.find('input').setValue('dup');
    await wrapper.find('form').trigger('submit');
    expect(wrapper.find('.evf-field__error').text()).toContain('已被占用');
  });

  it('不渲染具体操作栏，并为外部操作区提供稳定 form DOM ID', () => {
    const wrapper = mount(FormRenderer, {
      props: { schema: schema([]), formDomId: 'runtime-form-a' },
    });
    expect(wrapper.find('form').attributes('id')).toBe('runtime-form-a');
    expect(wrapper.find('.evf-runtime-action-bar').exists()).toBe(false);
  });

  it('通过公开命令保存填写草稿并发出成功事件', async () => {
    const saveDraft = vi.fn(async () => undefined);
    const wrapper = mount(FormRenderer, {
      props: {
        schema: schema([item({ type: 'text', widgetName: '_widget_t' })]),
        adapter: { saveDraft },
      },
    });
    await wrapper.find('input').setValue('draft');

    const outcome = await (
      wrapper.vm as unknown as { saveDraft(): Promise<{ ok: boolean }> }
    ).saveDraft();

    expect(outcome.ok).toBe(true);
    expect(saveDraft).toHaveBeenCalledTimes(1);
    expect(wrapper.emitted('draft-success')).toHaveLength(1);
  });

  it('公开命令互斥执行，不用新请求覆盖活动请求的取消控制器', async () => {
    let release: (() => void) | undefined;
    const submit = vi.fn(
      () =>
        new Promise<FormSubmitResult>((resolve) => {
          release = () => resolve({ accepted: true });
        }),
    );
    const wrapper = mount(FormRenderer, {
      props: {
        schema: schema([item({ type: 'text', widgetName: '_widget_t' })]),
        adapter: { submit },
      },
    });
    const exposed = wrapper.vm as unknown as {
      submit(): Promise<{ ok: boolean; reason?: string }>;
      saveDraft(): Promise<{ ok: boolean; reason?: string }>;
    };

    const submitting = exposed.submit();
    const competing = await exposed.saveDraft();

    expect(competing).toMatchObject({ ok: false, reason: 'busy' });
    expect(submit).toHaveBeenCalledTimes(1);
    release?.();
    await submitting;
  });

  it('enable=false 字段渲染禁用态控件', () => {
    const doc = schema([item({ type: 'text', widgetName: '_widget_d', enable: false })]);
    const wrapper = mount(FormRenderer, { props: { schema: doc } });
    expect((wrapper.find('input').element as HTMLInputElement).disabled).toBe(true);
  });

  it('按 field_layout 编译标签页并保持字段定义单一事实源', () => {
    const first = item({ type: 'text', widgetName: '_widget_top' });
    const detail = item({ type: 'text', widgetName: '_widget_detail' }, { lineWidth: 3 });
    const doc: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [first, detail],
        layout_fields: [
          {
            name: '_layout_tabs',
            type: 'multitab',
            tabStyle: 'style2',
            container: [
              {
                name: '_tab_detail',
                type: 'tab',
                title: '详情',
                field_layout: ['_widget_detail'],
              },
            ],
          },
        ],
        field_layout: ['_widget_top', '_layout_tabs'],
      },
    };
    const plan = buildRenderPlan(doc);
    expect(plan.sections[0].nodes[0]).toMatchObject({ type: 'field', key: '_widget_top' });
    expect(plan.sections[0].nodes[1]).toMatchObject({
      type: 'multitab',
      tabs: [{ title: '详情', fields: [{ key: '_widget_detail' }] }],
    });

    const wrapper = mount(FormRenderer, { props: { schema: doc } });
    // Core 的默认回退不绑定 UI 库；终端 Surface 注入自己的标签页呈现器。
    expect(wrapper.find('.evf-plain-multitab__pane .evf-field').attributes('style')).toContain(
      '--evf-field-span: 3',
    );
  });
});
