/**
 * 目标协议版本迁移器（P1，字段字典 1.3）。
 *
 * 文档内不携带版本号：协议版本由持久层外部承载（forms.protocol_version 列 +
 * FORM_PROTOCOL_VERSION 常量）。迁移器职责是「读入 → 迁移为当前版本 → 校验」三步；
 * v1 会补齐平铺引用，v1/v2 再补默认单列后升级为 v3。未来递增协议版本时追加显式升级步骤，
 * 禁止在旧版本校验器内隐式兼容新结构。
 */

import { cloneFormSchema } from './clone';
import { FORM_PROTOCOL_VERSION, type FormSchemaDocument } from './types';
import { type FormSchemaIssue, validateFormSchema } from './validate';

export interface FormSchemaMigrationResult {
  /** 迁移 + 校验后的规范文档；失败为 null。 */
  document: FormSchemaDocument | null;
  issues: FormSchemaIssue[];
  protocolVersion: number;
}

/**
 * 读入外部文档并迁移为当前协议版本（读取侧统一入口）：
 * - 结构非法：返回 issues（含 JSON Path），document=null；
 * - 结构合法：返回当前版本的深拷贝文档（v1 字段顺序迁移为 field_layout）。
 */
export function migrateFormSchema(
  input: unknown,
  sourceVersion: number = FORM_PROTOCOL_VERSION,
): FormSchemaMigrationResult {
  let candidate = input;
  if (sourceVersion === 1 && isV1Document(candidate)) {
    const document = candidate as {
      content: { type: 'form'; items: FormSchemaDocument['content']['items'] };
    };
    candidate = {
      content: {
        ...document.content,
        layout: 'normal',
        items: cloneFormSchema(document.content.items),
        layout_fields: [],
        field_layout: document.content.items.map((item) => item.widget.widgetName),
      },
    };
  } else if (sourceVersion === 2 && isV2Document(candidate)) {
    candidate = {
      content: {
        ...cloneFormSchema(candidate.content),
        layout: 'normal',
      },
    };
  } else if (sourceVersion !== FORM_PROTOCOL_VERSION) {
    return {
      document: null,
      issues: [{ path: 'content', message: `不支持的表单协议版本：${sourceVersion}` }],
      protocolVersion: sourceVersion,
    };
  }
  const result = validateFormSchema(candidate);
  if (!result.valid || !result.document) {
    return { document: null, issues: result.issues, protocolVersion: FORM_PROTOCOL_VERSION };
  }
  return {
    document: cloneFormSchema(result.document),
    issues: [],
    protocolVersion: FORM_PROTOCOL_VERSION,
  };
}

function isV2Document(
  input: unknown,
): input is { content: Omit<FormSchemaDocument['content'], 'layout'> } {
  if (!input || typeof input !== 'object' || Array.isArray(input)) return false;
  const content = (input as { content?: unknown }).content;
  return Boolean(
    content &&
    typeof content === 'object' &&
    !Array.isArray(content) &&
    (content as { type?: unknown }).type === 'form' &&
    Array.isArray((content as { items?: unknown }).items) &&
    Array.isArray((content as { layout_fields?: unknown }).layout_fields) &&
    Array.isArray((content as { field_layout?: unknown }).field_layout),
  );
}

function isV1Document(input: unknown): boolean {
  if (!input || typeof input !== 'object' || Array.isArray(input)) return false;
  const content = (input as { content?: unknown }).content;
  return Boolean(
    content &&
    typeof content === 'object' &&
    !Array.isArray(content) &&
    (content as { type?: unknown }).type === 'form' &&
    Array.isArray((content as { items?: unknown }).items),
  );
}
