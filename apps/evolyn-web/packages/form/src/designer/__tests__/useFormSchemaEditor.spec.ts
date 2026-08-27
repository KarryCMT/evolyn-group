import { describe, expect, it } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import { createEmptyFormSchemaDocument, useFormSchemaEditor } from '../useFormSchemaEditor';

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
  it('空文档初始化与整体替换', () => {
    const editor = useFormSchemaEditor();
    expect(editor.items.value).toEqual([]);
    const next: FormSchemaDocument = {
      content: { type: 'form', items: [textItem('_widget_a')] },
    };
    editor.replaceDocument(next);
    expect(editor.items.value).toHaveLength(1);
    expect(editor.selectedKey.value).toBe('');
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

  it('拖入的临时对象（无 widget）不会让 selectedItem 等响应式计算抛错', () => {
    const editor = useFormSchemaEditor();
    editor.addItem('text');
    // 模拟 vuedraggable 拖入瞬间的素材标记对象：只有 paletteType、没有 widget。
    (editor.items.value as unknown[]).push({ paletteType: 'number' });
    editor.addItem('number', 1); // add 事件内以真实字段替换标记
    expect(() => editor.selectedItem.value).not.toThrow();
    expect(editor.items.value).toHaveLength(2);
    expect(editor.items.value.every((item) => item.widget)).toBe(true);

    // 删除回落路径同样容忍标记对象
    (editor.items.value as unknown[]).unshift({ paletteType: 'text' });
    editor.removeItem(editor.items.value[1].widget.widgetName);
    expect(editor.items.value).toHaveLength(2);
  });
});
