import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import { createWidgetItem } from '../../schema/dictionary';
import FormSchemaLayoutCanvas from '../FormSchemaLayoutCanvas.vue';

function field(widgetName: string, lineWidth: number): FormItem {
  return {
    widget: {
      type: 'text',
      widgetName,
      enable: true,
      visible: true,
      allowBlank: true,
    },
    label: widgetName,
    description: '',
    labelHidden: false,
    lineWidth,
  };
}

describe('FormSchemaLayoutCanvas', () => {
  it('顶层与标签页字段共用 12 栅格 lineWidth，标签页容器跨整行', () => {
    const document: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'grid-4',
        items: [field('_widget_top', 3), field('_widget_tab', 9)],
        layout_fields: [
          {
            name: '_layout_tabs',
            type: 'multitab',
            tabStyle: 'style2',
            container: [
              {
                name: '_tab_main',
                type: 'tab',
                title: '标签页1',
                field_layout: ['_widget_tab'],
              },
            ],
          },
        ],
        field_layout: ['_widget_top', '_layout_tabs'],
      },
    };

    const wrapper = mount(FormSchemaLayoutCanvas, {
      props: { document, selectedKey: '' },
    });
    const topNodes = wrapper.findAll(
      '.form-schema-layout-canvas__list > .form-schema-layout-canvas__node',
    );
    expect(topNodes[0]?.attributes('style')).toContain('--form-schema-field-span: 3');
    expect(topNodes[1]?.attributes('style')).toContain('--form-schema-field-span: 12');
    expect(
      wrapper
        .find('.form-schema-layout-canvas__tab-list .form-schema-layout-canvas__node')
        .attributes('style'),
    ).toContain('--form-schema-field-span: 9');
    expect(wrapper.text()).not.toContain('在右侧配置标签页属性');
    expect(wrapper.find('.form-schema-layout-canvas__tabs-delete').attributes('aria-label')).toBe(
      '删除标签页',
    );
  });

  it('画布字段卡片不展示 widgetName', () => {
    const item = field('_widget_internal_key', 12);
    item.label = '客户名称';
    const document: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [item],
        layout_fields: [],
        field_layout: [item.widget.widgetName],
      },
    };

    const wrapper = mount(FormSchemaLayoutCanvas, {
      props: { document, selectedKey: '' },
    });

    expect(wrapper.text()).toContain('客户名称');
    expect(wrapper.text()).not.toContain('_widget_internal_key');
    expect(wrapper.find('.form-schema-field-card__key').exists()).toBe(false);
  });

  it('空标签页提示位于虚线拖拽区域内部', () => {
    const document: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [],
        layout_fields: [
          {
            name: '_layout_empty_tabs',
            type: 'multitab',
            tabStyle: 'style1',
            container: [
              {
                name: '_tab_empty',
                type: 'tab',
                title: '空标签页',
                field_layout: [],
              },
            ],
          },
        ],
        field_layout: ['_layout_empty_tabs'],
      },
    };

    const wrapper = mount(FormSchemaLayoutCanvas, {
      props: { document, selectedKey: '_layout_empty_tabs' },
    });

    const dropArea = wrapper.find('.form-schema-layout-canvas__tab-list');
    expect(dropArea.find('.form-schema-layout-canvas__tab-empty').text()).toBe(
      '将字段拖入当前标签页',
    );
  });

  it('子表单以横向字段表格渲染并提供嵌套拖入提示', () => {
    const subform = createWidgetItem('subform');
    const document: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [subform],
        layout_fields: [],
        field_layout: [subform.widget.widgetName],
      },
    };
    const wrapper = mount(FormSchemaLayoutCanvas, {
      props: { document, selectedKey: subform.widget.widgetName },
    });

    expect(wrapper.find('.form-schema-subform-card').classes()).toContain('is-active');
    expect(wrapper.text()).toContain('从左侧拖入字段，或在右侧添加子字段');
  });
});
