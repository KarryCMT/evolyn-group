import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import type { FormItem } from '../../schema/types';
import FormSchemaSubmitRuleDialog from '../FormSchemaSubmitRuleDialog.vue';
import FormSchemaSubmitRuleFieldPicker from '../FormSchemaSubmitRuleFieldPicker.vue';

function field(name: string, label: string, type: FormItem['widget']['type'] = 'text'): FormItem {
  return {
    widget: {
      type,
      widgetName: name,
      enable: true,
      visible: true,
      allowBlank: true,
    } as FormItem['widget'],
    label,
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

const items: FormItem[] = [
  field('_widget_name', '姓名'),
  field('_widget_amount', '金额', 'number'),
  field('_widget_memo', '备注'),
];

function mountDialog(
  props: Partial<InstanceType<typeof FormSchemaSubmitRuleDialog>['$props']> = {},
) {
  return mount(FormSchemaSubmitRuleDialog, {
    props: {
      modelValue: true,
      submitRule: 2,
      widgetSubmitRules: {},
      items,
      ...props,
    },
    global: {
      stubs: {
        ElDialog: {
          template:
            '<div class="el-dialog-stub"><header><slot name="header" /></header><slot /><footer><slot name="footer" /></footer></div>',
          props: ['modelValue'],
        },
        ElPopover: {
          template: '<div class="el-popover-stub"><slot name="reference" /><slot /></div>',
          props: ['visible'],
        },
        ElTooltip: { template: '<span><slot /></span>' },
      },
    },
  });
}

function confirm(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.find('.form-submit-rule-dialog__confirm').trigger('click');
}

describe('FormSchemaSubmitRuleDialog', () => {
  it('只展示与默认策略不同的两段，并保留截图中的弹窗、页脚与字段入口结构', () => {
    const wrapper = mountDialog({ submitRule: 1 });

    expect(wrapper.find('.form-submit-rule-dialog__header').exists()).toBe(true);
    expect(wrapper.find('.form-submit-rule-dialog__footer').exists()).toBe(true);
    expect(wrapper.find('.form-submit-rule-dialog__help-link').text()).toBe('查看帮助文档');
    expect(wrapper.findAll('.form-submit-rule-dialog__section')).toHaveLength(2);
    expect(
      wrapper.findAll('.form-submit-rule-dialog__section-title').map((node) => node.text()),
    ).toEqual(['空值', '始终重新计算']);
    expect(wrapper.findAll('.form-submit-rule-dialog__add')).toHaveLength(2);
  });

  it('已有特殊规则以可移除标签展示；移回默认策略时保存结果归一化剔除', async () => {
    const wrapper = mountDialog({
      submitRule: 1,
      widgetSubmitRules: { _widget_name: 2, _widget_amount: 2 },
    });

    expect(wrapper.findAll('.form-submit-rule-dialog__tag').map((node) => node.text())).toEqual(
      expect.arrayContaining(['姓名', '金额']),
    );

    await wrapper.find('[aria-label="移除 姓名"]').trigger('click');
    await confirm(wrapper);
    await nextTick();

    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({ _widget_amount: 2 });
    expect(wrapper.emitted('update:model-value')?.[0]).toEqual([false]);
  });

  it('字段选择浮层选择字段后写入草稿，确认后仅上抛非默认规则', async () => {
    const wrapper = mountDialog({ submitRule: 1 });

    await wrapper.find('.form-submit-rule-dialog__add').trigger('click');
    await nextTick();
    const picker = wrapper.findComponent(FormSchemaSubmitRuleFieldPicker);
    expect(picker.exists()).toBe(true);

    picker.vm.$emit('select', '_widget_name');
    await nextTick();
    expect(wrapper.find('[aria-label="移除 姓名"]').exists()).toBe(true);

    await confirm(wrapper);
    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({ _widget_name: 2 });
  });

  it('与默认策略相同的历史项在保存时被归一化剔除', async () => {
    const wrapper = mountDialog({ submitRule: 2, widgetSubmitRules: { _widget_name: 2 } });

    await confirm(wrapper);
    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({});
  });

  it('取消不上抛 save，草稿选择会被丢弃', async () => {
    const wrapper = mountDialog({ submitRule: 1 });
    await wrapper.find('.form-submit-rule-dialog__add').trigger('click');
    await nextTick();
    wrapper.findComponent(FormSchemaSubmitRuleFieldPicker).vm.$emit('select', '_widget_name');
    await nextTick();

    await wrapper.find('.form-submit-rule-dialog__cancel').trigger('click');
    expect(wrapper.emitted('save')).toBeUndefined();
    expect(wrapper.emitted('update:model-value')?.[0]).toEqual([false]);
  });

  it('始终重新计算与其他规则组一致，可添加字段并保存为规则 3', async () => {
    const wrapper = mountDialog({ submitRule: 1 });
    const recompute = wrapper.findAll('.form-submit-rule-dialog__section')[1]!;

    await recompute.find('.form-submit-rule-dialog__add').trigger('click');
    await nextTick();
    const picker = wrapper.findComponent(FormSchemaSubmitRuleFieldPicker);
    expect(picker.props('strategy')).toBe(3);

    picker.vm.$emit('select', '_widget_amount');
    await confirm(wrapper);

    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({ _widget_amount: 3 });
  });
});
