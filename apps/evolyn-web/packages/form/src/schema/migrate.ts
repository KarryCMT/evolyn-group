/**
 * 目标协议版本迁移器（P1，字段字典 1.3）。
 *
 * 文档内不携带版本号：协议版本由持久层外部承载（forms.protocol_version 列 +
 * FORM_PROTOCOL_VERSION 常量）。迁移器职责是「读入 → 迁移为当前版本 → 校验」三步；
 * v1 会补齐平铺引用，v1/v2 再补默认单列；v1–v3 为子表单补齐 v4 展示与权限配置，
 * 所有受支持版本都会把子表单归一化为整行宽度；v4 及更早版本补齐 v5 的
 * fieldShowRules 空数组。禁止在旧版本校验器内隐式兼容新结构。
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
  if (sourceVersion < 1 || sourceVersion > FORM_PROTOCOL_VERSION) {
    return {
      document: null,
      issues: [{ path: 'content', message: `不支持的表单协议版本：${sourceVersion}` }],
      protocolVersion: sourceVersion,
    };
  }
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
  }
  if (isV1Document(candidate)) {
    candidate = normalizeSubformV4(candidate, sourceVersion <= 3);
  }
  if (sourceVersion <= 4 && isV1Document(candidate)) {
    candidate = normalizeFieldShowRulesV5(candidate);
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

/** v4 补齐子表单配置，并将其容器宽度固定为整行 12 栅格。 */
function normalizeSubformV4(input: unknown, fillMissingConfig: boolean): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  for (const item of document.content.items) {
    if (item.widget.type !== 'subform') continue;
    item.lineWidth = 12;
    if (!fillMissingConfig) continue;
    const widget = item.widget;
    widget.subformCreate ??= true;
    widget.subformInsert ??= true;
    widget.subformEdit ??= true;
    widget.subformDelete ??= true;
    widget.quickFill ??= true;
    widget.pcStickyColumn ??= { enable: true, limit: 1 };
    widget.mobileStickyColumn ??= { enable: false, limit: 1 };
    widget.mobileViewStyle ??= 'vertical';
    widget.mobileSummaryFieldCount ??= 3;
  }
  return document;
}

/** v4 → v5：补齐 fieldShowRules 空数组（v5 起 content 必填键）。 */
function normalizeFieldShowRulesV5(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  const content = document.content as unknown as Record<string, unknown>;
  if (!Array.isArray(content.fieldShowRules)) {
    content.fieldShowRules = [];
  }
  return document;
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
