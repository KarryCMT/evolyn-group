import type { FormItem } from '../schema/types';

/**
 * 前端事件的前端兼容模型。字段引用沿用既有 widgetName，而非展示标题；标题可以
 * 随时改名，widgetName 才是表单内稳定的配置键。后端落地后此结构将原样进入
 * content.formEvents，当前设计器先以它作为受控 UI 模型。
 */
export interface FormEventRequestEntry {
  key: string;
  value: string;
}

export interface FormEventRequest {
  method: 'get' | 'post';
  url: string;
  header: FormEventRequestEntry[];
  body: FormEventRequestEntry[];
  format: 'json' | 'xml';
}

export interface FormEventAction {
  field: string;
  value: string;
}

export interface FormEvent {
  id: string;
  enabled: boolean;
  name: string;
  description: string;
  trigger: string;
  trigger_type: 'widget';
  request_type: 0;
  request: FormEventRequest;
  request_rely: string[];
  action: FormEventAction[];
  action_rely: string[];
  subform_fill_rule: 'merge' | 'replace';
}

export interface FormEventFieldOption {
  key: string;
  label: string;
  type: string;
  group?: string;
}

/** 设计器内部控件类型不直接面向用户展示，统一投影为字段选择器可读标签。 */
export function formEventFieldTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    text: '文本',
    textarea: '文本',
    number: '数值',
    decimal: '数值',
    money: '数值',
    percent: '数值',
    date: '日期',
    datetime: '日期时间',
    select: '选项',
    checkbox: '选项',
    radio: '选项',
    member: '成员',
    department: '部门',
    subform: '子表单',
    image: '图片',
    file: '附件',
  };
  return labels[type] ?? '字段';
}

export const FORM_EVENT_LIMITS = {
  maxEvents: 50,
  nameMaxLength: 64,
  descriptionMaxLength: 500,
  templateMaxLength: 4_000,
  maxRequestEntries: 20,
  maxActions: 50,
} as const;

/** 不含随机依赖的可读 ID；碰撞由调用方在同一事件数组内兜底重试。 */
export function createFormEventId(): string {
  return `evt_${Math.random().toString(36).slice(2, 14)}`;
}

export function createFormEvent(): FormEvent {
  return {
    id: createFormEventId(),
    enabled: true,
    name: '',
    description: '',
    trigger: '',
    trigger_type: 'widget',
    request_type: 0,
    request: { method: 'get', url: '', header: [], body: [], format: 'json' },
    request_rely: [],
    action: [],
    action_rely: [],
    subform_fill_rule: 'merge',
  };
}

/** 受控变量仅允许完整 `${widgetName}` 令牌，避免把任意表达式带入运行时。 */
export function referencedWidgetNames(template: string): string[] {
  const names = new Set<string>();
  for (const match of template.matchAll(/\$\{([A-Za-z0-9_]+)\}/g)) names.add(match[1]!);
  return [...names];
}

/** 保存前重建依赖索引；索引不接受手工修改，防止与文本模板漂移。 */
export function normalizeFormEvent(event: FormEvent): FormEvent {
  const next = structuredClone(event);
  const requestTemplates = [
    next.request.url,
    ...next.request.header.flatMap((entry) => [entry.key, entry.value]),
    ...next.request.body.flatMap((entry) => [entry.key, entry.value]),
  ];
  next.request_rely = uniqueReferences(requestTemplates);
  next.action_rely = uniqueReferences(next.action.flatMap((action) => [action.value]));
  return next;
}

export function formEventFieldOptions(items: readonly FormItem[]): FormEventFieldOption[] {
  return items.flatMap((item) => {
    const parent = { key: item.widget.widgetName, label: item.label, type: item.widget.type };
    if (item.widget.type !== 'subform') return [parent];
    return [
      parent,
      ...item.widget.items.map((child) => ({
        key: child.widget.widgetName,
        label: child.label,
        type: child.widget.type,
        group: item.label || '子表单',
      })),
    ];
  });
}

function uniqueReferences(templates: readonly string[]): string[] {
  return [...new Set(templates.flatMap(referencedWidgetNames))];
}
