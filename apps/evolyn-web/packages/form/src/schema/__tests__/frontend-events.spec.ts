import { describe, expect, it } from 'vitest';
import { migrateFormSchema } from '../migrate';
import type { FormSchemaDocument } from '../types';
import { validateFormSchema } from '../validate';

function eventDocument(): FormSchemaDocument {
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items: [
        {
          widget: {
            type: 'text',
            widgetName: '_widget_source',
            fieldId: 'aaaaaaaa01',
            enable: true,
            visible: true,
            allowBlank: true,
          },
          label: '来源',
          description: '',
          labelHidden: false,
          lineWidth: 12,
        },
        {
          widget: {
            type: 'text',
            widgetName: '_widget_target',
            fieldId: 'aaaaaaaa02',
            enable: true,
            visible: true,
            allowBlank: true,
          },
          label: '目标',
          description: '',
          labelHidden: false,
          lineWidth: 12,
        },
      ],
      layout_fields: [],
      field_layout: ['_widget_source', '_widget_target'],
      fieldShowRules: [],
      submitRule: 2,
      widget_submit_rules: {},
      validators: [],
      preSubmitConfirm: {
        enable: false,
        title: '确认继续提交吗？',
        content: '请确认填写内容无误后继续提交。',
      },
      formEvents: [
        {
          id: 'evt_contact01',
          enabled: true,
          name: '补全联系信息',
          description: '',
          trigger: '_widget_source',
          trigger_type: 'widget',
          request_type: 0,
          request: {
            method: 'get',
            url: 'https://api.lingyanyun.com/member?id=${_widget_source}',
            header: [],
            body: [],
            format: 'json',
          },
          request_rely: ['_widget_source'],
          action: [{ field: '_widget_target', value: '$response.data.name' }],
          action_rely: [],
          subform_fill_rule: 'merge',
        },
      ],
    linkages: [],
    fieldFormulas: [],
    },
  };
}

describe('formEvents v10 schema', () => {
  it('accepts Authorization and rejects session-carrying headers', () => {
    const document = eventDocument();
    expect(validateFormSchema(document).issues).toEqual([]);
    document.content.formEvents[0]!.request.header = [{ key: 'Authorization', value: 'secret' }];
    expect(validateFormSchema(document).issues).toEqual([]);
    document.content.formEvents[0]!.request.header = [{ key: 'Cookie', value: 'session=secret' }];
    expect(validateFormSchema(document).issues).toContainEqual({
      path: 'content.formEvents[0].request.header[0].key',
      message: 'Header 名称无效或属于禁止的敏感 Header',
    });
  });

  it('migrates v9 documents to an empty event list', () => {
    const document = eventDocument() as unknown as { content: Record<string, unknown> };
    delete document.content.formEvents;
    const migrated = migrateFormSchema(document, 9);
    expect(migrated.issues).toEqual([]);
    expect(migrated.document?.content.formEvents).toEqual([]);
  });
});
