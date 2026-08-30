import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
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
  });
});
