/**
 * 目标协议版本迁移器（P1，字段字典 1.3）。
 *
 * 文档内不携带版本号：协议版本由持久层外部承载（forms.protocol_version 列 +
 * FORM_PROTOCOL_VERSION 常量）。迁移器职责是「读入 → 迁移为当前版本 → 校验」三步；
 * v1 即原样校验。未来递增协议版本时在此追加形态识别与升级步骤，禁止在 v1 内隐式兼容。
 */

import { cloneFormSchema } from './clone';
import type { FormSchemaDocument } from './types';
import { validateFormSchema, type FormSchemaIssue } from './validate';

export interface FormSchemaMigrationResult {
  /** 迁移 + 校验后的规范文档；失败为 null。 */
  document: FormSchemaDocument | null;
  issues: FormSchemaIssue[];
}

/**
 * 读入外部文档并迁移为当前协议版本（读取侧统一入口）：
 * - 结构非法：返回 issues（含 JSON Path），document=null；
 * - 结构合法：返回深拷贝文档（v1 无需迁移步骤）。
 */
export function migrateFormSchema(input: unknown): FormSchemaMigrationResult {
  const result = validateFormSchema(input);
  if (!result.valid || !result.document) {
    return { document: null, issues: result.issues };
  }
  return { document: cloneFormSchema(result.document), issues: [] };
}
