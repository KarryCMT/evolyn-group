import type { FormSchemaDocument } from '@evolyn.do/form/schema';

/**
 * 表单预览的 sessionStorage 传递约定（宿主侧机制，不进入 form 包）：
 * 设计器把当前草稿全文（目标保存协议）写入会话存储，预览页在表单未发布时
 * 读取草稿本地回放；已发布时优先走后端 bootstrap。键按表单 ID 隔离，
 * 仅存活于当前浏览器会话。
 */
const STORAGE_PREFIX = 'evolyn.form.preview.';

export function formPreviewStorageKey(formId: string): string {
  return `${STORAGE_PREFIX}${formId}`;
}

export function saveFormPreviewDocument(formId: string, document: FormSchemaDocument): void {
  sessionStorage.setItem(formPreviewStorageKey(formId), JSON.stringify(document));
}

export function loadFormPreviewDocument(formId: string): FormSchemaDocument | null {
  const raw = sessionStorage.getItem(formPreviewStorageKey(formId));
  if (!raw) return null;
  try {
    return JSON.parse(raw) as FormSchemaDocument;
  } catch {
    return null;
  }
}
