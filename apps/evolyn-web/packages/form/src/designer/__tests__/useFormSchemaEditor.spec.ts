import { describe, expect, it } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import { useFormSchemaEditor } from '../useFormSchemaEditor';

function textItem(widgetName: string): FormItem {
  return {
    widget: {
      type: 'text',
      widgetName,
      enable: true,
      visible: true,
      allowBlank: true,
    },
    label: `字段 ${widgetName}`,
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

describe('useFormSchemaEditor', () => {
  it('切换表单布局同步更新顶层与标签页字段，新字段继承当前列宽', () => {
    const editor = useFormSchemaEditor();
    const topField = editor.addItem('text');
    const layout = editor.addMultitab();
    const tabField = editor.addItem('textarea', -1, {
      type: 'tab',
      layoutName: layout.name,
      tabName: layout.container[0].name,
    });

    editor.setFormLayout('grid-4');

    expect(editor.document.value.content.layout).toBe('grid-4');
    expect(topField.lineWidth).toBe(3);
    expect(tabField.lineWidth).toBe(3);
    expect(editor.addItem('number').lineWidth).toBe(3);
    expect(editor.addItem('separator').lineWidth).toBe(12);
  });

  it('空文档初始化与整体替换', () => {
    const editor = useFormSchemaEditor();
    expect(editor.items.value).toEqual([]);
    const next: FormSchemaDocument = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [textItem('_widget_a')],
        layout_fields: [],
        field_layout: ['_widget_a'],
      },
    };
    editor.replaceDocument(next);
    expect(editor.items.value).toHaveLength(1);
    expect(editor.document.value.content.field_layout).toEqual(['_widget_a']);
    expect(editor.selectedKey.value).toBe('');
  });

  it('标签页引用字段且删除布局时无损展开', () => {
    const editor = useFormSchemaEditor();
    const field = editor.addItem('text');
    const layout = editor.addMultitab();
    expect(editor.selectedLayout.value?.name).toBe(layout.name);
    editor.replaceReferences({ type: 'top' }, [layout.name]);
    editor.replaceReferences(
      { type: 'tab', layoutName: layout.name, tabName: layout.container[0].name },
      [field.widget.widgetName],
    );
    expect(layout.container[0].field_layout).toEqual([field.widget.widgetName]);
    editor.removeMultitab(layout.name);
    expect(editor.document.value.content.field_layout).toEqual([field.widget.widgetName]);
    expect(editor.items.value.map((item) => item.widget.widgetName)).toContain(
      field.widget.widgetName,
    );
    expect(editor.selectedLayout.value).toBeUndefined();
  });

  it('右侧属性动作可复制并重新排序标签页', () => {
    const editor = useFormSchemaEditor();
    const field = editor.addItem('text');
    const layout = editor.addMultitab();
    const firstTab = layout.container[0];
    editor.replaceReferences({ type: 'top' }, [layout.name]);
    editor.replaceReferences({ type: 'tab', layoutName: layout.name, tabName: firstTab.name }, [
      field.widget.widgetName,
    ]);

    editor.duplicateTab(layout.name, firstTab.name);
    expect(layout.container).toHaveLength(3);
    expect(editor.items.value).toHaveLength(2);
    expect(layout.container[1].field_layout[0]).not.toBe(field.widget.widgetName);

    const order = layout.container.map((tab) => tab.name).reverse();
    editor.reorderTabs(layout.name, order);
    expect(layout.container.map((tab) => tab.name)).toEqual(order);
  });

  it('新增/复制/删除/选中/重命名动作', () => {
    const editor = useFormSchemaEditor();
    const added = editor.addItem('text');
    expect(editor.items.value).toHaveLength(1);
    expect(editor.selectedKey.value).toBe(added.widget.widgetName);

    editor.copyItem(added);
    expect(editor.items.value).toHaveLength(2);
    expect(editor.selectedItem.value?.label).toBe(`${added.label} copy`);

    const second = editor.items.value[1];
    editor.selectItem(second.widget.widgetName);
    editor.renameItemKey('_widget_renamed');
    expect(second.widget.widgetName).toBe('_widget_renamed');
    expect(editor.selectedItem.value).toBe(second);

    editor.removeItem('_widget_renamed');
    expect(editor.items.value).toHaveLength(1);
    // 删除当前选中项后回落到相邻项（items 为响应式代理，按键断言）
    expect(editor.selectedItem.value?.widget.widgetName).toBe(added.widget.widgetName);
  });

  it('拖入的素材标记在引用层转换为字段定义', () => {
    const editor = useFormSchemaEditor();
    editor.addItem('text');
    editor.replaceReferences({ type: 'top' }, [
      editor.document.value.content.field_layout[0],
      { paletteType: 'number' },
    ]);
    expect(() => editor.selectedItem.value).not.toThrow();
    expect(editor.items.value).toHaveLength(2);
    expect(editor.items.value.every((item) => item.widget)).toBe(true);
    expect(editor.document.value.content.field_layout).toHaveLength(2);
  });
});
