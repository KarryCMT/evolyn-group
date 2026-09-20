import { reactive } from 'vue';
import { describe, expect, it } from 'vitest';
import { createFormEvent, formEventFieldTypeLabel, normalizeFormEvent, referencedWidgetNames } from '../frontend-events';

describe('frontend event helpers', () => {
  it('extracts unique widget tokens in template order', () => {
    expect(referencedWidgetNames('x=${_widget_name}&id=${_widget_id}&again=${_widget_name}')).toEqual([
      '_widget_name',
      '_widget_id',
    ]);
  });

  it('rebuilds request and action dependency indexes from the configured templates', () => {
    const event = createFormEvent();
    event.request.url = 'https://api.example.test?name=${_widget_name}';
    event.request.header = [{ key: 'x-id', value: '${_widget_id}' }];
    event.action = [{ field: '_widget_phone', value: '${_widget_name}-$response.data.phone' }];
    event.request_rely = ['stale'];
    event.action_rely = ['stale'];

    expect(normalizeFormEvent(event)).toMatchObject({
      request_rely: ['_widget_name', '_widget_id'],
      action_rely: ['_widget_name'],
    });
  });

  it('converts a reactive editor draft into a serializable event payload', () => {
    const event = reactive(createFormEvent());
    event.name = '查询员工信息';
    event.request.url = 'https://api.lingyanyun.test/employees/${_widget_phone}';
    event.request.header = [{ key: 'X-Source', value: 'designer' }];
    event.action = [{ field: '_widget_name', value: '$response.data.name' }];

    expect(normalizeFormEvent(event)).toMatchObject({
      name: '查询员工信息',
      request: {
        method: 'get',
        url: 'https://api.lingyanyun.test/employees/${_widget_phone}',
        header: [{ key: 'X-Source', value: 'designer' }],
        body: [],
        format: 'json',
      },
      request_rely: ['_widget_phone'],
      action: [{ field: '_widget_name', value: '$response.data.name' }],
    });
  });

  it('maps internal widget types to labels intended for form designers', () => {
    expect(formEventFieldTypeLabel('text')).toBe('文本');
    expect(formEventFieldTypeLabel('textarea')).toBe('文本');
    expect(formEventFieldTypeLabel('unknown')).toBe('字段');
  });
});
