import { createEmptyFormDocument } from './fields';
import {
  FORM_SCHEMA_VERSION,
  type FormDocument,
  type FormField,
  type FormFieldType,
  type FormJsonValue,
} from './types';

export interface FormSchemaValidationIssue {
  code: string;
  path: string;
  message: string;
}

export type FormSchemaValidationResult =
  | { valid: true; document: FormDocument }
  | { valid: false; issues: FormSchemaValidationIssue[] };

export interface FormSchemaNormalizationOptions {
  /** 接入应用可按已启用能力收窄字段类型，防止未知字段进入运行时。 */
  isFieldType?: (type: string) => type is FormFieldType;
}

/** 兼容预版本 JSON；后续 Schema 升级在这里按 version 逐级迁移。 */
export function migrateFormDocument(input: unknown): unknown {
  if (!isRecord(input)) return input;

  const defaults = createEmptyFormDocument();
  return {
    ...defaults,
    ...input,
    version: 'version' in input ? input.version : FORM_SCHEMA_VERSION,
    kind: input.kind === 'workflow' ? 'workflow' : 'standard',
    fields: Array.isArray(input.fields) ? input.fields : [],
    settings: isRecord(input.settings) ? input.settings : {},
  };
}

/** 校验后的文档可直接交给设计器或运行态使用，不携带 Vue 或其他运行时对象。 */
export function validateFormDocument(
  input: unknown,
  options: FormSchemaNormalizationOptions = {},
): FormSchemaValidationResult {
  const migrated = migrateFormDocument(input);
  const issues: FormSchemaValidationIssue[] = [];
  if (!isRecord(migrated)) return invalid(issues, 'invalid-root', '$', '表单配置必须是对象。');
  if (migrated.version !== FORM_SCHEMA_VERSION) {
    return invalid(
      issues,
      'unsupported-version',
      '$.version',
      `暂不支持表单配置版本 ${String(migrated.version)}。`,
    );
  }
  const kind =
    migrated.kind === 'standard' || migrated.kind === 'workflow' ? migrated.kind : undefined;
  if (!kind) {
    addIssue(issues, 'invalid-kind', '$.kind', '表单类型必须是 standard 或 workflow。');
  }
  const title = readText(migrated.title, '$.title', issues);
  if (!Array.isArray(migrated.fields)) {
    addIssue(issues, 'invalid-fields', '$.fields', 'fields 必须是数组。');
  }
  if (!isRecord(migrated.settings) || !isJsonValue(migrated.settings)) {
    addIssue(issues, 'invalid-settings', '$.settings', 'settings 必须是可序列化对象。');
  }

  const ids = new Set<string>();
  const keys = new Set<string>();
  const fields = Array.isArray(migrated.fields)
    ? migrated.fields.flatMap((field, index) => {
        const result = validateField(field, index, ids, keys, issues, options);
        return result ? [result] : [];
      })
    : [];
  if (issues.length || !kind || !title || !isRecord(migrated.settings)) {
    return { valid: false, issues };
  }

  return {
    valid: true,
    document: {
      version: FORM_SCHEMA_VERSION,
      kind,
      title,
      fields,
      settings: { ...migrated.settings },
    },
  };
}

/** 无效文档返回 null，调用方据此决定显示错误还是回退为初始文档。 */
export function normalizeFormDocument(
  input: unknown,
  options: FormSchemaNormalizationOptions = {},
): FormDocument | null {
  const result = validateFormDocument(input, options);
  return result.valid ? cloneFormDocument(result.document) : null;
}

export function cloneFormDocument(document: FormDocument): FormDocument {
  return JSON.parse(JSON.stringify(document)) as FormDocument;
}

function validateField(
  value: unknown,
  index: number,
  ids: Set<string>,
  keys: Set<string>,
  issues: FormSchemaValidationIssue[],
  options: FormSchemaNormalizationOptions,
): FormField | null {
  const path = `$.fields[${index}]`;
  if (!isRecord(value)) {
    addIssue(issues, 'invalid-field', path, '字段必须是对象。');
    return null;
  }

  const issueCount = issues.length;
  const id = readText(value.id, `${path}.id`, issues);
  const key = readText(value.key, `${path}.key`, issues);
  const type = readText(value.type, `${path}.type`, issues);
  const label = readText(value.label, `${path}.label`, issues);
  if (id && ids.has(id)) addIssue(issues, 'duplicate-field-id', `${path}.id`, '字段 id 不能重复。');
  if (key && keys.has(key))
    addIssue(issues, 'duplicate-field-key', `${path}.key`, '字段 key 不能重复。');
  if (id) ids.add(id);
  if (key) keys.add(key);
  if (type && options.isFieldType && !options.isFieldType(type)) {
    addIssue(issues, 'unknown-field-type', `${path}.type`, `不支持字段类型 ${type}。`);
  }
  if (typeof value.required !== 'boolean') {
    addIssue(issues, 'invalid-required', `${path}.required`, 'required 必须是布尔值。');
  }
  if (!isRecord(value.config) || !isJsonValue(value.config)) {
    addIssue(issues, 'invalid-config', `${path}.config`, 'config 必须是可序列化对象。');
  }
  if (issues.length > issueCount || !id || !key || !type || !label || !isRecord(value.config)) {
    return null;
  }

  return {
    id,
    key,
    type: type as FormFieldType,
    label,
    required: value.required as boolean,
    config: { ...value.config },
  };
}

function readText(
  value: unknown,
  path: string,
  issues: FormSchemaValidationIssue[],
): string | null {
  if (typeof value === 'string' && value.trim()) return value;

  addIssue(issues, 'invalid-text', path, '必须是非空字符串。');
  return null;
}

function isRecord(value: unknown): value is Record<string, FormJsonValue> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isJsonValue(value: unknown, seen = new WeakSet<object>()): value is FormJsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return true;
  if (typeof value === 'number') return Number.isFinite(value);
  if (typeof value !== 'object') return false;
  if (seen.has(value)) return false;

  seen.add(value);
  const valid = Array.isArray(value)
    ? value.every((item) => isJsonValue(item, seen))
    : (Object.getPrototypeOf(value) === Object.prototype ||
        Object.getPrototypeOf(value) === null) &&
      Object.values(value).every((item) => isJsonValue(item, seen));
  seen.delete(value);
  return valid;
}

function addIssue(
  issues: FormSchemaValidationIssue[],
  code: string,
  path: string,
  message: string,
) {
  issues.push({ code, path, message });
}

function invalid(
  issues: FormSchemaValidationIssue[],
  code: string,
  path: string,
  message: string,
): FormSchemaValidationResult {
  addIssue(issues, code, path, message);
  return { valid: false, issues };
}
