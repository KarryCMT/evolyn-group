import type {
  FormEvent as SchemaFormEvent,
  FormEventAction as SchemaFormEventAction,
  FormEventRequest as SchemaFormEventRequest,
  FormEventRequestEntry as SchemaFormEventRequestEntry,
  FormItem,
} from '../schema/types';

/**
 * 前端事件的设计器别名。协议事实源位于 schema/types，避免设计态与保存协议
 * 各维护一套容易漂移的结构。
 */
export type FormEventRequestEntry = SchemaFormEventRequestEntry;
export type FormEventRequest = SchemaFormEventRequest;
export type FormEventAction = SchemaFormEventAction;
export type FormEvent = SchemaFormEvent;

export interface FormEventDebugRequest {
  values: Record<string, unknown>;
  sequence: number;
}

export interface FormEventDebugResult {
  sequence: number;
  writes: Record<string, unknown>;
  requestSummary: {
    method: string;
    url: string;
    headerNames: string[];
    body?: string;
  };
  responseSummary: {
    statusCode: number;
    durationMs: number;
    format: 'json' | 'xml';
    body: string;
  };
  errorCode?: string;
}

export type FormEventDebugExecutor = (
  eventId: string,
  request: FormEventDebugRequest,
) => Promise<FormEventDebugResult>;

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
  // 编辑器传入的是 Vue reactive proxy；structuredClone 不能可靠克隆 Proxy，
  // 某些组件更新还可能把浏览器 Event 带入值。这里显式收敛为 JSON 协议字段，
  // 既消除 DataCloneError，也避免界面对象进入待保存配置。
  const next: FormEvent = {
    id: stringValue(event.id),
    enabled: event.enabled === true,
    name: stringValue(event.name),
    description: stringValue(event.description),
    trigger: stringValue(event.trigger),
    trigger_type: 'widget',
    request_type: 0,
    request: {
      method: event.request?.method === 'post' ? 'post' : 'get',
      url: stringValue(event.request?.url),
      header: normalizeRequestEntries(event.request?.header),
      body: normalizeRequestEntries(event.request?.body),
      format: event.request?.format === 'xml' ? 'xml' : 'json',
    },
    request_rely: [],
    action: normalizeActions(event.action),
    action_rely: [],
    subform_fill_rule: event.subform_fill_rule === 'replace' ? 'replace' : 'merge',
  };
  const requestTemplates = [
    next.request.url,
    ...next.request.header.flatMap((entry) => [entry.key, entry.value]),
    ...next.request.body.flatMap((entry) => [entry.key, entry.value]),
  ];
  next.request_rely = uniqueReferences(requestTemplates);
  next.action_rely = uniqueReferences(next.action.flatMap((action) => [action.value]));
  return next;
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function normalizeRequestEntries(entries: unknown): FormEventRequestEntry[] {
  if (!Array.isArray(entries)) return [];
  return entries
    .map((entry) => {
      const record = entry as Partial<FormEventRequestEntry> | null;
      return { key: stringValue(record?.key), value: stringValue(record?.value) };
    })
    .filter((entry) => entry.key.trim() !== '' && entry.value !== '');
}

function normalizeActions(actions: unknown): FormEventAction[] {
  if (!Array.isArray(actions)) return [];
  return actions.map((action) => {
    const record = action as Partial<FormEventAction> | null;
    return { field: stringValue(record?.field), value: stringValue(record?.value) };
  });
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
