import { describe, expect, it } from 'vitest';
import {
  FORM_FIELD_WIDTH_OPTIONS,
  FORM_LAYOUT_LINE_WIDTH,
  WIDGET_SPECS,
  createWidgetItem,
  generateWidgetName,
} from '../dictionary';
import { cloneFormSchema } from '../clone';
import { migrateFormSchema } from '../migrate';
import type { FormSchemaDocument } from '../types';
import { validateFormSchema, validatePublishableFormSchema } from '../validate';

function subformConfig() {
  return {
    subformCreate: true,
    subformInsert: true,
    subformEdit: true,
    subformDelete: true,
    quickFill: true,
    pcStickyColumn: { enable: true, limit: 1 },
    mobileStickyColumn: { enable: false, limit: 1 },
    mobileViewStyle: 'vertical',
    mobileSummaryFieldCount: 3,
  } as const;
}

/** 构造合法 text 字段项（P1 验收样本的最小形态）。 */
function textItem(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    widget: {
      type: 'text',
      widgetName: '_widget_a1',
      enable: true,
      visible: true,
      allowBlank: true,
      ...overrides,
    },
    label: '单行文本',
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

function documentWith(items: unknown[]): unknown {
  const field_layout = items.flatMap((item) => {
    const name = (item as { widget?: { widgetName?: unknown } })?.widget?.widgetName;
    return typeof name === 'string' ? [name] : [];
  });
  return { content: { type: 'form', layout: 'normal', items, layout_fields: [], field_layout } };
}

describe('validateFormSchema 结构校验', () => {
  it('冻结表单布局与字段宽度产品映射', () => {
    expect(FORM_LAYOUT_LINE_WIDTH).toEqual({
      normal: 12,
      'grid-2': 6,
      'grid-3': 4,
      'grid-4': 3,
    });
    expect(FORM_FIELD_WIDTH_OPTIONS).toEqual([
      { label: '1/4', value: 3 },
      { label: '1/3', value: 4 },
      { label: '1/2', value: 6 },
      { label: '2/3', value: 6 },
      { label: '3/4', value: 9 },
      { label: '整行', value: 12 },
    ]);
  });

  it('接受样本协议文档并返回深拷贝（未编辑属性不丢失）', () => {
    const input = documentWith([textItem()]);
    const result = validateFormSchema(input);
    expect(result.valid).toBe(true);
    expect(result.issues).toEqual([]);
    expect(result.document).toEqual(input);
    // 深拷贝：修改结果不影响入参。
    result.document!.content.items[0].label = '改名';
    expect((input as FormSchemaDocument).content.items[0].label).toBe('单行文本');
  });

  it('接受空表单', () => {
    const result = validateFormSchema(documentWith([]));
    expect(result.valid).toBe(true);
  });

  it('只接受四种表单布局枚举', () => {
    for (const layout of ['normal', 'grid-2', 'grid-3', 'grid-4']) {
      const document = documentWith([]) as FormSchemaDocument;
      document.content.layout = layout as FormSchemaDocument['content']['layout'];
      expect(validateFormSchema(document).valid).toBe(true);
    }
    const invalid = documentWith([]) as { content: Record<string, unknown> };
    invalid.content.layout = 'grid-5';
    expect(validateFormSchema(invalid).issues[0]).toMatchObject({ path: 'content.layout' });
  });

  it('拒绝根/content 未知键与非法 type', () => {
    expect(
      validateFormSchema({
        content: { type: 'form', layout: 'normal', items: [], layout_fields: [], field_layout: [] },
        extra: 1,
      }).issues[0].path,
    ).toBe('content.extra');
    expect(validateFormSchema({ content: { type: 'page', items: [] } }).issues[0].path).toBe(
      'content.type',
    );
  });

  it('拒绝 item 层未知键', () => {
    // placeholder 属于 widget 层：放在 item 层应报未知属性。
    const misplaced = { ...textItem(), placeholder: 'x' };
    const result = validateFormSchema(documentWith([misplaced]));
    expect(result.valid).toBe(false);
    expect(result.issues[0].path).toBe('content.items[0].placeholder');
  });

  it('拒绝 widget 层未知键并给出精确路径', () => {
    const item = textItem();
    (item.widget as Record<string, unknown>).unknownProp = 1;
    const result = validateFormSchema(documentWith([item]));
    expect(result.valid).toBe(false);
    expect(result.issues[0].path).toBe('content.items[0].widget.unknownProp');
  });

  it('拒绝未知控件类型', () => {
    const result = validateFormSchema(documentWith([textItem({ type: 'magic' })]));
    expect(result.valid).toBe(false);
    expect(result.issues[0].path).toBe('content.items[0].widget.type');
  });

  it('公共布尔属性缺省/null 均拒绝', () => {
    const widget = textItem().widget as Record<string, unknown>;
    delete widget.enable;
    expect(validateFormSchema(documentWith([{ ...textItem(), widget }])).valid).toBe(false);

    const widgetNull = textItem({ allowBlank: null }).widget;
    const result = validateFormSchema(documentWith([{ ...textItem(), widget: widgetNull }]));
    expect(result.issues.some((issue) => issue.path.endsWith('allowBlank'))).toBe(true);
  });

  it('widgetName 非法形状与作用域内重复均拒绝', () => {
    expect(validateFormSchema(documentWith([textItem({ widgetName: '1bad' })])).valid).toBe(false);
    expect(
      validateFormSchema(documentWith([textItem(), textItem({ widgetName: '_widget_a1' })])).valid,
    ).toBe(false);
    // 不同作用域（子表单内）允许与顶层同名。
    const child = textItem({ widgetName: '_widget_top' });
    const subform = {
      widget: {
        type: 'subform',
        widgetName: '_widget_sub',
        enable: true,
        visible: true,
        allowBlank: true,
        items: [child],
        ...subformConfig(),
      },
      label: '子表单',
      description: '',
      labelHidden: false,
      lineWidth: 12,
    };
    expect(
      validateFormSchema(documentWith([textItem({ widgetName: '_widget_top' }), subform])).valid,
    ).toBe(true);
    // 子表单作用域内重复仍拒绝。
    const duplicated = {
      ...subform,
      widget: { ...(subform.widget as Record<string, unknown>), items: [child, { ...child }] },
    };
    expect(validateFormSchema(documentWith([duplicated])).valid).toBe(false);
  });

  it('label/description/lineWidth 边界校验', () => {
    expect(validateFormSchema(documentWith([{ ...textItem(), label: '' }])).valid).toBe(false);
    expect(validateFormSchema(documentWith([{ ...textItem(), description: null }])).valid).toBe(
      false,
    );
    expect(validateFormSchema(documentWith([{ ...textItem(), lineWidth: 13 }])).valid).toBe(false);
    const subform = {
      widget: {
        type: 'subform',
        widgetName: '_widget_full_row_subform',
        enable: true,
        visible: true,
        allowBlank: true,
        items: [],
        ...subformConfig(),
      },
      label: '子表单',
      description: '',
      labelHidden: false,
      lineWidth: 6,
    };
    expect(validateFormSchema(documentWith([subform])).issues).toContainEqual({
      path: 'content.items[0].lineWidth',
      message: '子表单必须固定占整行（lineWidth=12）',
    });
    // separator 允许空 label。
    const separator = {
      widget: {
        type: 'separator',
        widgetName: '_widget_sep',
        enable: true,
        visible: true,
        allowBlank: true,
        style: 'dashed',
      },
      label: '',
      description: '',
      labelHidden: false,
      lineWidth: 12,
    };
    expect(validateFormSchema(documentWith([separator])).valid).toBe(true);
  });

  it('选项数组：必填、条目结构、value 唯一性', () => {
    const radio = (options: unknown) =>
      textItem({ type: 'radiogroup', widgetName: '_widget_r1', options });
    expect(validateFormSchema(documentWith([radio(undefined)])).valid).toBe(false);
    expect(validateFormSchema(documentWith([radio([])])).valid).toBe(false);
    expect(
      validateFormSchema(
        documentWith([
          radio([
            { label: 'A', value: 'a' },
            { label: 'B', value: 'a' },
          ]),
        ]),
      ).valid,
    ).toBe(false);
  });

  it('数值交叉约束：min ≤ max / minLength ≤ maxLength / defaultValue 命中选项', () => {
    expect(
      validateFormSchema(
        documentWith([textItem({ type: 'number', widgetName: '_widget_n1', min: 10, max: 1 })]),
      ).valid,
    ).toBe(false);
    expect(
      validateFormSchema(documentWith([textItem({ minLength: 10, maxLength: 2 })])).valid,
    ).toBe(false);
    expect(
      validateFormSchema(
        documentWith([
          textItem({
            type: 'radiogroup',
            widgetName: '_widget_r2',
            options: [{ label: 'A', value: 'a' }],
            defaultValue: 'b',
          }),
        ]),
      ).valid,
    ).toBe(false);
  });

  it('子表单子项类型白名单与禁止嵌套', () => {
    const nestedSubform = {
      widget: {
        type: 'subform',
        widgetName: '_widget_sub2',
        enable: true,
        visible: true,
        allowBlank: true,
        ...subformConfig(),
        items: [
          {
            widget: {
              type: 'subform',
              widgetName: '_widget_inner',
              enable: true,
              visible: true,
              allowBlank: true,
              items: [],
              ...subformConfig(),
            },
            label: '嵌套子表单',
            description: '',
            labelHidden: false,
            lineWidth: 12,
          },
        ],
      },
      label: '子表单',
      description: '',
      labelHidden: false,
      lineWidth: 12,
    };
    const result = validateFormSchema(documentWith([nestedSubform]));
    expect(result.valid).toBe(false);
    expect(result.issues[0].message).toContain('子表单内不允许使用控件');
  });

  it('27 种控件均可由字典工厂生成并通过校验', () => {
    const items = Object.keys(WIDGET_SPECS).map((type) =>
      createWidgetItem(type as keyof typeof WIDGET_SPECS),
    );
    const result = validateFormSchema(documentWith(items));
    expect(result.issues).toEqual([]);
    expect(result.valid).toBe(true);
  });

  it('标签页可引用顶层子表单，但不能引用子表单内部字段', () => {
    const child = textItem({ widgetName: '_widget_child' });
    const subform = {
      widget: {
        type: 'subform',
        widgetName: '_widget_sub',
        enable: true,
        visible: true,
        allowBlank: true,
        items: [child],
        ...subformConfig(),
      },
      label: '子表单',
      description: '',
      labelHidden: false,
      lineWidth: 12,
    };
    const document = {
      content: {
        type: 'form',
        layout: 'normal',
        items: [subform],
        layout_fields: [
          {
            name: '_layout_tabs',
            type: 'multitab',
            tabStyle: 'style2',
            container: [
              {
                name: '_tab_detail',
                title: '明细',
                type: 'tab',
                field_layout: ['_widget_sub'],
              },
            ],
          },
        ],
        field_layout: ['_layout_tabs'],
      },
    };
    expect(validateFormSchema(document).valid).toBe(true);
    document.content.layout_fields[0].container[0].field_layout = ['_widget_child'];
    const result = validateFormSchema(document);
    expect(result.valid).toBe(false);
    expect(result.issues[0].path).toBe('content.layout_fields[0].container[0].field_layout[0]');
  });

  it('拒绝字段在顶层与标签页重复放置', () => {
    const item = textItem();
    const document = documentWith([item]) as FormSchemaDocument;
    document.content.layout_fields.push({
      name: '_layout_tabs',
      type: 'multitab',
      tabStyle: 'style1',
      container: [
        { name: '_tab_main', type: 'tab', title: '标签页', field_layout: ['_widget_a1'] },
      ],
    });
    document.content.field_layout.push('_layout_tabs');
    const result = validateFormSchema(document);
    expect(result.valid).toBe(false);
    expect(result.issues.some((issue) => issue.message.includes('重复'))).toBe(true);
  });
});

describe('validatePublishableFormSchema 发布白名单', () => {
  it('基础 9 类可发布', () => {
    const items = [
      'text',
      'textarea',
      'number',
      'datetime',
      'radiogroup',
      'checkboxgroup',
      'combo',
      'combocheck',
      'separator',
    ].map((type) => createWidgetItem(type as never));
    expect(validatePublishableFormSchema(documentWith(items)).valid).toBe(true);
  });

  it('白名单外控件返回精确路径（如 user）', () => {
    const result = validatePublishableFormSchema(
      documentWith([createWidgetItem('text'), createWidgetItem('user')]),
    );
    expect(result.valid).toBe(false);
    expect(result.issues[0].path).toBe('content.items[1].widget.type');
  });
});

describe('migrateFormSchema / cloneFormSchema', () => {
  it('v3 文档原样校验通过', () => {
    const input = documentWith([textItem()]);
    const result = migrateFormSchema(input, 3);
    expect(result.document).toEqual(input);
    expect(result.issues).toEqual([]);
  });

  it('当前协议读取时把子表单宽度归一化为整行', () => {
    const subform = createWidgetItem('subform');
    subform.lineWidth = 6;
    const result = migrateFormSchema(documentWith([subform]), 4);
    expect(result.issues).toEqual([]);
    expect(result.document?.content.items[0]?.lineWidth).toBe(12);
  });

  it('非法输入返回 issues 且 document 为 null', () => {
    const result = migrateFormSchema({ nope: true }, 3);
    expect(result.document).toBeNull();
    expect(result.issues.length).toBeGreaterThan(0);
  });

  it('v1 平铺文档无损迁移为 v4 引用结构与单列布局', () => {
    const item = textItem();
    const result = migrateFormSchema({ content: { type: 'form', items: [item] } }, 1);
    expect(result.document?.content.layout_fields).toEqual([]);
    expect(result.document?.content.field_layout).toEqual(['_widget_a1']);
    expect(result.document?.content.layout).toBe('normal');
    expect(result.protocolVersion).toBe(4);
  });

  it('v2 引用文档迁移为 v3 时补单列布局', () => {
    const current = documentWith([textItem()]);
    const { layout: _layout, ...content } = current.content;
    const result = migrateFormSchema({ content }, 2);
    expect(result.document?.content.layout).toBe('normal');
    expect(result.document?.content.field_layout).toEqual(['_widget_a1']);
  });

  it('cloneFormSchema 切断引用', () => {
    const doc = documentWith([textItem()]) as FormSchemaDocument;
    const cloned = cloneFormSchema(doc);
    cloned.content.items.push(createWidgetItem('text'));
    expect(doc.content.items.length).toBe(1);
  });

  it('generateWidgetName 生成 _widget_ 前缀且不重复', () => {
    const first = generateWidgetName();
    const second = generateWidgetName();
    expect(first).toMatch(/^_widget_\d+$/);
    expect(first).not.toBe(second);
  });
});
