import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import { EvolynRichTextEditor } from '@evolyn.do/ui';
import type {
  FormItem,
  FormMultitabLayout,
  FormSchemaDocument,
  SubmitRule,
} from '../../schema/types';
import { createWidgetItem } from '../../schema/dictionary';
import FormSchemaCommonPropertyPanel from '../FormSchemaCommonPropertyPanel.vue';
import FormSchemaPropertyPanel from '../FormSchemaPropertyPanel.vue';

const layout: FormMultitabLayout = {
  name: '_layout_tabs',
  type: 'multitab',
  tabStyle: 'style2',
  container: [
    { name: '_tab_first', type: 'tab', title: '标签页1', field_layout: [] },
    { name: '_tab_second', type: 'tab', title: '标签页2', field_layout: [] },
  ],
};

const field: FormItem = {
  widget: {
    type: 'text',
    widgetName: '_widget_name',
    enable: true,
    visible: true,
    allowBlank: true,
  },
  label: '姓名',
  description: '',
  labelHidden: false,
  lineWidth: 12,
};

describe('FormSchemaPropertyPanel', () => {
  it('选中标签页布局后在右侧展示并上抛标签配置动作', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { layout } });
    expect(wrapper.text()).toContain('多标签显示');
    expect(wrapper.text()).toContain('添加标签页');

    await wrapper.find('input[aria-label="标签页标题"]').setValue('基础信息');
    expect(wrapper.emitted('rename-tab')?.[0]).toEqual(['_tab_first', '基础信息']);
  });

  it('字段属性通过副本事件提交，不直接修改父级字段对象', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });
    await wrapper.find('input[placeholder="请输入标题"]').setValue('联系人');
    await nextTick();

    expect(field.label).toBe('姓名');
    const updates = wrapper.emitted('update-item') ?? [];
    expect(updates[updates.length - 1]?.[0]).toMatchObject({ label: '联系人' });
  });

  it('字段说明使用富文本编辑器', () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });

    expect(wrapper.findComponent(EvolynRichTextEditor).exists()).toBe(true);
  });

  it('提示文字读取并写入字段 placeholder，缺省时展示画布的默认占位文案', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });
    const promptInput = wrapper.find('input[aria-label="提示文字"]');

    expect((promptInput.element as HTMLInputElement).value).toBe('请输入');
    await promptInput.setValue('请输入姓名');
    await nextTick();

    const updates = wrapper.emitted('update-item') ?? [];
    expect(updates[updates.length - 1]?.[0]).toMatchObject({
      widget: { placeholder: '请输入姓名' },
    });
  });

  it('单行文本按属性分区展示标题、默认值、校验、权限、宽度和安全栏位', () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });

    expect(wrapper.findComponent(FormSchemaCommonPropertyPanel).exists()).toBe(true);
    expect(wrapper.find('[aria-label="字段类型"]').exists()).toBe(true);
    expect(wrapper.find('input[placeholder="字段值与规则引用的稳定键"]').exists()).toBe(false);
    expect(wrapper.findAll('h3').map((node) => node.text())).toEqual([
      '描述信息',
      '提示文字',
      '格式',
      '默认值',
      '校验',
      '字段权限',
      '字段宽度',
      '字段安全',
    ]);
    expect(wrapper.text()).toContain('不允许重复值');
    expect(wrapper.text()).toContain('脱敏显示');
    expect(wrapper.text()).not.toContain('最小长度');
    expect(wrapper.text()).not.toContain('最大长度');
  });

  it('单行文本默认值来源可在自定义、数据联动和公式编辑之间选择', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });
    const defaultValueType = wrapper.find('[aria-label="默认值类型"]');

    expect(defaultValueType.attributes('disabled')).toBeUndefined();
    await defaultValueType.trigger('click');
    await nextTick();

    expect(document.body.textContent).toContain('自定义');
    expect(document.body.textContent).toContain('数据联动');
    expect(document.body.textContent).toContain('公式编辑');
  });

  it('默认值来源下拉复用于所有支持默认值的控件，分割线和标签页不展示', () => {
    for (const type of [
      'textarea',
      'number',
      'datetime',
      'radiogroup',
      'checkboxgroup',
      'combo',
      'combocheck',
    ] as const) {
      const wrapper = mount(FormSchemaPropertyPanel, { props: { item: createWidgetItem(type) } });
      expect(wrapper.find('[aria-label="默认值类型"]').exists()).toBe(true);
    }

    expect(
      mount(FormSchemaPropertyPanel, { props: { item: createWidgetItem('separator') } })
        .find('[aria-label="默认值类型"]')
        .exists(),
    ).toBe(false);
    expect(
      mount(FormSchemaPropertyPanel, { props: { layout } })
        .find('[aria-label="默认值类型"]')
        .exists(),
    ).toBe(false);
  });

  it('其他字段复用单行文本的公共分区，并在提示文字与校验之间放置专属配置', () => {
    const number = createWidgetItem('number');
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: number } });

    expect(wrapper.find('[aria-label="字段类型"]').exists()).toBe(true);
    expect(wrapper.findAll('h3').map((node) => node.text())).toEqual([
      '描述信息',
      '提示文字',
      '默认值',
      '数值范围',
      '校验',
      '字段权限',
      '字段宽度',
    ]);
    expect(wrapper.text()).not.toContain('字段安全');
    expect(wrapper.text()).not.toContain('脱敏显示');
  });

  it('多行文本同样不展示字符长度限制栏位', () => {
    const textarea = createWidgetItem('textarea');
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: textarea } });

    expect(wrapper.text()).not.toContain('字符长度');
    expect(wrapper.text()).not.toContain('最小长度');
    expect(wrapper.text()).not.toContain('最大长度');
  });

  it('表单属性上抛布局切换，字段宽度按产品映射展示', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { formLayout: 'grid-2' } });
    wrapper.findComponent({ name: 'ElSegmented' }).vm.$emit('update:modelValue', 'form');
    await nextTick();

    const layoutSelect = wrapper.findComponent({ name: 'ElSelect' });
    layoutSelect.vm.$emit('update:modelValue', 'grid-4');
    expect(wrapper.emitted('update-form-layout')?.[0]).toEqual(['grid-4']);

    await wrapper.setProps({ item: field });
    wrapper.findComponent({ name: 'ElSegmented' }).vm.$emit('update:modelValue', 'field');
    await nextTick();
    expect(wrapper.text()).toContain('1/4');
    expect(wrapper.text()).toContain('2/3');
    expect(wrapper.text()).toContain('整行');
  });

  it('字段显隐规则使用全宽入口，并在已有规则时展示配置数量', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, {
      props: {
        fieldShowRules: [
          {
            id: 'rule_show_detail',
            filter: { rel: 'and', cond: [] },
            fields: ['_widget_name'],
          },
        ],
      },
    });
    wrapper.findComponent({ name: 'ElSegmented' }).vm.$emit('update:modelValue', 'form');
    await nextTick();

    const entry = wrapper.find('.form-schema-property__show-rules-entry');
    expect(entry.exists()).toBe(true);
    expect(entry.classes()).toContain('el-button--primary');
    expect(entry.text()).toBe('管理显隐规则');
    expect(wrapper.text()).toContain('已配置 1 条规则');
    expect(wrapper.find('[aria-label="字段显隐规则说明"]').exists()).toBe(true);
  });

  it('子表单属性区展示子字段、权限与双端样式配置', () => {
    const subform = createWidgetItem('subform');
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: subform } });

    expect(wrapper.text()).toContain('子字段');
    expect(wrapper.text()).toContain('快速填报');
    expect(wrapper.text()).toContain('可新增记录');
    expect(wrapper.text()).toContain('子表单展示样式');
    expect(wrapper.text()).toContain('电脑端');
    expect(wrapper.text()).toContain('移动端');
  });
});

describe('FormSchemaPropertyPanel 不可见字段赋值（v6）', () => {
  const schemaDocument: FormSchemaDocument = {
    content: {
      type: 'form',
      layout: 'normal',
      items: [
        {
          widget: {
            type: 'text',
            widgetName: '_widget_name',
            enable: true,
            visible: true,
            allowBlank: true,
          },
          label: '姓名',
          description: '',
          labelHidden: false,
          lineWidth: 12,
        },
        {
          widget: {
            type: 'number',
            widgetName: '_widget_amount',
            enable: true,
            visible: true,
            allowBlank: true,
          },
          label: '金额',
          description: '',
          labelHidden: false,
          lineWidth: 12,
        },
      ] as FormItem[],
      layout_fields: [],
      field_layout: ['_widget_name', '_widget_amount'],
      fieldShowRules: [],
      submitRule: 2 as SubmitRule,
      widget_submit_rules: { _widget_name: 1 } as Record<string, SubmitRule>,
    },
  };

  async function mountFormTab() {
    const wrapper = mount(FormSchemaPropertyPanel, {
      props: {
        schemaDocument,
        submitRule: 2,
        widgetSubmitRules: { _widget_name: 1 },
      },
    });
    wrapper.findComponent({ name: 'ElSegmented' }).vm.$emit('update:modelValue', 'form');
    await nextTick();
    return wrapper;
  }

  it('默认赋值策略的三个选项均可选择并上抛', async () => {
    const wrapper = await mountFormTab();
    // 表单属性页有两个选择框（表单布局 / 不可见字段赋值），取后者。
    const selects = wrapper.findAllComponents({ name: 'ElSelect' });
    expect(selects.length).toBeGreaterThanOrEqual(2);
    selects[selects.length - 1]!.vm.$emit('update:modelValue', 3);
    await nextTick();
    expect(wrapper.emitted('update-submit-rule')?.[0]).toEqual([3]);
    expect(wrapper.text()).toContain('空值');
    expect(wrapper.find('[aria-label="配置特殊字段赋值规则"]').exists()).toBe(true);
  });

  it('存在特殊规则时展示摘要卡：默认收起、展开后按策略分组只显示标签', async () => {
    const wrapper = await mountFormTab();
    const summary = wrapper.find('.form-schema-property__submit-rule-summary');
    expect(summary.exists()).toBe(true);
    // 初始收起：分组内容不渲染。
    expect(wrapper.find('.form-schema-property__submit-rule-summary-body').exists()).toBe(false);
    await wrapper.find('.form-schema-property__submit-rule-summary-toggle').trigger('click');
    expect(wrapper.find('.form-schema-property__submit-rule-summary-body').exists()).toBe(true);
    // 分组展示策略名与字段标签，不暴露 widgetName。
    const text = wrapper.find('.form-schema-property__submit-rule-summary-body').text();
    expect(text).toContain('保持原值');
    expect(text).toContain('姓名');
    expect(text).not.toContain('_widget_name');
  });
});
