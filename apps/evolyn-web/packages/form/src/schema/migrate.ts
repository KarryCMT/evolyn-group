/**
 * 目标协议版本迁移器（P1，字段字典 1.3）。
 *
 * 文档内不携带版本号：协议版本由持久层外部承载（forms.protocol_version 列 +
 * FORM_PROTOCOL_VERSION 常量）。迁移器职责是「读入 → 迁移为当前版本 → 校验」三步；
 * v1 会补齐平铺引用，v1/v2 再补默认单列；v1–v3 为子表单补齐 v4 展示与权限配置，
 * 所有受支持版本都会把子表单归一化为整行宽度；v4 及更早版本补齐 v5 的
 * fieldShowRules 空数组；v5 及更早版本补齐 v6 的 submitRule 默认空值策略与
 * widget_submit_rules 空对象；v6 及更早版本补齐 v7 的 validators 与
 * preSubmitConfirm；v9 及更早版本补齐 v10 的 formEvents 空数组。禁止在旧版本
 * 校验器内隐式兼容新结构。
 */

import { cloneFormSchema } from './clone';
import { DEFAULT_PRE_SUBMIT_CONFIRM } from './dictionary';
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
  if (sourceVersion <= 5 && isV1Document(candidate)) {
    candidate = normalizeSubmitRulesV6(candidate);
  }
  if (sourceVersion <= 6 && isV1Document(candidate)) {
    candidate = normalizeSubmitValidationV7(candidate);
  }
  if (sourceVersion <= 7 && isV1Document(candidate)) {
    candidate = normalizeFieldIdentityV8(candidate);
  }
  if (sourceVersion <= 8 && isV1Document(candidate)) {
    candidate = normalizeSerialNumberV9(candidate);
  }
  if (sourceVersion <= 9 && isV1Document(candidate)) {
    candidate = normalizeFrontendEventsV10(candidate);
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

/** v9 → v10：前端事件成为草稿与发布快照的一部分，旧文档保持未配置语义。 */
function normalizeFrontendEventsV10(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  const content = document.content as unknown as Record<string, unknown>;
  if (!Array.isArray(content.formEvents)) content.formEvents = [];
  return document;
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

/**
 * v5 → v6：补齐 submitRule 默认空值策略与 widget_submit_rules 空对象
 * （v6 起 content 必填键）。v4 数据同时经 v5 归一化，一次补齐三键；
 * 升级后保持原行为——所有不可见字段仍按空值写入。
 */
function normalizeSubmitRulesV6(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  const content = document.content as unknown as Record<string, unknown>;
  if (!isSubmitRuleValue(content.submitRule)) {
    content.submitRule = 2;
  }
  if (!isPlainRecord(content.widget_submit_rules)) {
    content.widget_submit_rules = {};
  }
  return document;
}

/** v7 → v8：为缺失 fieldId 的值字段生成不可变标识（物理表存储 §4.1 契约
 * 冻结）。fieldId 由设计器侧一次性生成：10 位小写 base36 随机串，表单内
 * （含全部子表单）全局唯一；生成后随草稿持久化，此后永不改变。 */
function normalizeFieldIdentityV8(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  const content = document.content as unknown as { items: unknown[] };
  const used = new Set<string>();
  const nextFieldId = (): string => {
    for (;;) {
      let id = '';
      const bytes = new Uint32Array(FIELD_ID_LENGTH);
      crypto.getRandomValues(bytes);
      for (const byte of bytes) id += FIELD_ID_ALPHABET[byte % FIELD_ID_ALPHABET.length];
      if (!used.has(id)) {
        used.add(id);
        return id;
      }
    }
  };
  const walk = (items: unknown[]): void => {
    for (const raw of items) {
      const item = raw as { widget?: Record<string, unknown> };
      const widget = item?.widget;
      if (!widget || typeof widget !== 'object') continue;
      const type = widget.type;
      if (type === 'separator' || type === 'button') continue;
      if (typeof widget.fieldId !== 'string' || widget.fieldId === '') {
        widget.fieldId = nextFieldId();
      } else {
        used.add(widget.fieldId);
      }
      if (type === 'subform' && Array.isArray(widget.items)) walk(widget.items);
    }
  };
  walk(content.items);
  return document;
}

/** v8 → v9：将从未开放运行时的旧 sn.rule 升级为片段数组，保留其显示顺序。 */
function normalizeSerialNumberV9(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  for (const item of document.content.items) {
    if (item.widget.type !== 'sn') continue;
    const legacy = item.widget as unknown as Record<string, unknown>;
    if (Array.isArray(legacy.rules)) continue;
    const rule = isPlainRecord(legacy.rule) ? legacy.rule : {};
    const prefix = typeof rule.prefix === 'string' ? rule.prefix : '';
    const dateFmt = rule.dateFmt === 'yyyyMM' || rule.dateFmt === 'yyyyMMdd' ? rule.dateFmt : null;
    const digits = typeof rule.seqLength === 'number' ? rule.seqLength : 5;
    legacy.rules = [
      ...(prefix ? [{ type: 'literal', value: prefix }] : []),
      ...(dateFmt
        ? [
            {
              type: 'submittedAt',
              format: dateFmt,
              formatType: rule.formatType === 'custom' ? 'custom' : 'preset',
            },
          ]
        : []),
      {
        type: 'counter',
        digits,
        fixedWidth: true,
        resetCycle: typeof rule.resetCycle === 'string' ? rule.resetCycle : 'none',
        initialValue: 1,
      },
    ];
    delete legacy.rule;
  }
  return document;
}

/** fieldId 生成字母表与长度（小写 base36 × 10 位，与后端 storage 包口径一致）。 */
const FIELD_ID_ALPHABET = '0123456789abcdefghijklmnopqrstuvwxyz';
const FIELD_ID_LENGTH = 10;

/** v6 → v7：补齐表单级校验空数组和关闭的二次确认配置。 */
function normalizeSubmitValidationV7(input: unknown): unknown {
  const document = cloneFormSchema(input as FormSchemaDocument);
  const content = document.content as unknown as Record<string, unknown>;
  if (!Array.isArray(content.validators)) content.validators = [];
  if (!isPlainRecord(content.preSubmitConfirm)) {
    content.preSubmitConfirm = cloneFormSchema(DEFAULT_PRE_SUBMIT_CONFIRM);
  }
  return document;
}

function isSubmitRuleValue(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 3;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
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
