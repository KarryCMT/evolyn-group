import { describe, expect, it } from 'vitest';
import {
  type FormSchemaDocument,
  compileSubmitValidators,
  evaluateCompiledSubmitValidators,
  evaluateSubmitValidators,
  migrateFormSchema,
  renderSubmitTemplate,
  validateFormSchema,
} from '..';

function document(): FormSchemaDocument {
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items: [
        {
          label: '联系电话',
          description: '',
          labelHidden: false,
          lineWidth: 12,
          widget: {
            type: 'text',
            widgetName: '_widget_phone',
            enable: true,
            visible: true,
            allowBlank: true,
          },
        },
        {
          label: '姓名',
          description: '',
          labelHidden: false,
          lineWidth: 12,
          widget: {
            type: 'text',
            widgetName: '_widget_name',
            enable: true,
            visible: true,
            allowBlank: true,
          },
        },
      ],
      layout_fields: [],
      field_layout: ['_widget_phone', '_widget_name'],
      fieldShowRules: [],
      submitRule: 2,
      widget_submit_rules: {},
      validators: [
        {
          formula: 'LEN($_widget_phone#) == 11',
          remind: '联系电话 ${_widget_phone} 必须为 11 位数字',
          remark: '联系电话长度',
          realtime: true,
          failAction: 0,
        },
      ],
      preSubmitConfirm: {
        enable: true,
        title: '确认 ${_widget_name} 的提交？',
        content: '联系电话：${_widget_phone}',
      },
      formEvents: [],
    linkages: [],
    fieldFormulas: [],
    },
  };
}

describe('v7 提交校验纯逻辑', () => {
  it('将 v6 文档稳定迁移为显式的 v7 默认配置', () => {
    const legacy = document();
    const content = legacy.content as unknown as Record<string, unknown>;
    delete content.validators;
    delete content.preSubmitConfirm;

    const migrated = migrateFormSchema(legacy, 6);
    expect(migrated.document?.content.validators).toEqual([]);
    expect(migrated.document?.content.preSubmitConfirm).toEqual({
      enable: false,
      title: '确认继续提交吗？',
      content: '请确认填写内容无误后继续提交。',
    });
  });

  it('执行受控公式、保留失败顺序，并渲染字段模板', () => {
    const schema = document();
    const context = {
      values: { _widget_phone: '123', _widget_name: '灵衍云' },
      isVisible: () => true,
    };

    expect(evaluateSubmitValidators(schema.content, context)).toEqual([
      {
        index: 0,
        remind: '联系电话 123 必须为 11 位数字',
        fields: ['_widget_phone'],
        failAction: 0,
      },
    ]);
    expect(
      renderSubmitTemplate(schema.content.preSubmitConfirm.title, schema.content.items, context),
    ).toBe('确认 灵衍云 的提交？');
  });

  it('复用发布快照的编译 AST，并支持按依赖规则定向求值', () => {
    const schema = document();
    schema.content.validators.push({
      formula: 'LEN($_widget_name#) > 0',
      remind: '姓名不能为空',
      remark: '姓名长度',
      realtime: true,
      failAction: 1,
    });
    const compiled = compileSubmitValidators(schema.content);
    const context = {
      values: { _widget_phone: '123', _widget_name: '' },
      isVisible: () => true,
    };

    expect(evaluateCompiledSubmitValidators(compiled, context, new Set([1]))).toEqual([
      {
        index: 1,
        remind: '姓名不能为空',
        fields: ['_widget_name'],
        failAction: 1,
      },
    ]);
  });

  it('有效不可见字段在公式和模板中均按空值处理', () => {
    const schema = document();
    const context = {
      values: { _widget_phone: '13800138000', _widget_name: '不应泄露' },
      isVisible: (field: string) => field !== '_widget_phone' && field !== '_widget_name',
    };

    expect(evaluateSubmitValidators(schema.content, context)).toHaveLength(1);
    expect(
      renderSubmitTemplate(schema.content.preSubmitConfirm.content, schema.content.items, context),
    ).toBe('联系电话：');
  });

  it('允许宿主以当前会话已知展示名格式化成员等 ID 型字段', () => {
    const schema = document();
    const context = {
      values: { _widget_phone: 'member_1', _widget_name: '灵衍云' },
      isVisible: () => true,
      formatTemplateValue: (field: string, value: unknown) =>
        field === '_widget_phone' && value === 'member_1' ? '张三' : undefined,
    };

    expect(renderSubmitTemplate('提交人：${_widget_phone}', schema.content.items, context)).toBe(
      '提交人：张三',
    );
  });

  it('严格拒绝未开放函数与错误参数数量', () => {
    const schema = document();
    schema.content.validators[0]!.formula = 'ISEMPTY($_widget_phone#)';
    const unsupported = validateFormSchema(schema);
    expect(unsupported.valid).toBe(false);
    expect(unsupported.issues.some((issue) => issue.message.includes('未开放函数'))).toBe(true);

    schema.content.validators[0]!.formula = 'NOT($_widget_phone#, $_widget_name#)';
    const arity = validateFormSchema(schema);
    expect(arity.valid).toBe(false);
    expect(arity.issues.some((issue) => issue.message.includes('参数数量'))).toBe(true);
  });

  it('在保存期拒绝白名单函数的错误输入类型', () => {
    const schema = document();
    schema.content.items[0]!.widget = {
      type: 'number',
      widgetName: '_widget_phone',
      enable: true,
      visible: true,
      allowBlank: true,
    };
    const result = validateFormSchema(schema);

    expect(result.valid).toBe(false);
    expect(result.issues.map((issue) => issue.message)).toContain('LEN 参数 必须是 text 类型');
  });
});
