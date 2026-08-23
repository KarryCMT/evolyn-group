import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue';
import {
  cloneFormDocument,
  normalizeFormDocument,
  type FormDocument,
  type FormSchemaNormalizationOptions,
  type FormSchemaValidationIssue,
} from '../schema';

/** 共享包通过 adapter 与 Web 的 API、鉴权和租户上下文解耦。 */
export interface FormPersistenceAdapter {
  load: () => Promise<unknown>;
  save: (document: FormDocument) => Promise<unknown>;
}

export interface UseFormPersistenceOptions extends FormSchemaNormalizationOptions {
  initialDocument: FormDocument;
  adapter: FormPersistenceAdapter;
}

export interface FormPersistence {
  document: ShallowRef<FormDocument>;
  isLoading: Readonly<ShallowRef<boolean>>;
  isSaving: Readonly<ShallowRef<boolean>>;
  isDirty: ComputedRef<boolean>;
  issues: Readonly<ShallowRef<FormSchemaValidationIssue[]>>;
  load: () => Promise<FormDocument>;
  save: () => Promise<FormDocument>;
  reset: () => void;
}

export function useFormPersistence(options: UseFormPersistenceOptions): FormPersistence {
  const initialDocument = resolveDocument(options.initialDocument, options);
  const document = shallowRef(cloneFormDocument(initialDocument));
  const savedDocument = shallowRef(cloneFormDocument(initialDocument));
  const isLoading = shallowRef(false);
  const isSaving = shallowRef(false);
  const issues = shallowRef<FormSchemaValidationIssue[]>([]);
  const isDirty = computed(() => !isSameDocument(document.value, savedDocument.value));

  async function load() {
    isLoading.value = true;
    issues.value = [];
    try {
      const response = await options.adapter.load();
      if (response === null || response === undefined) {
        commit(initialDocument);
        return document.value;
      }

      const resolved = normalizeFormDocument(response, options);
      if (!resolved) {
        issues.value = [
          {
            code: 'invalid-persisted-document',
            path: '$',
            message: '已保存的表单配置无效，已恢复空白表单。',
          },
        ];
        commit(initialDocument);
        return document.value;
      }

      commit(resolved);
      return document.value;
    } finally {
      isLoading.value = false;
    }
  }

  async function save() {
    const normalized = normalizeFormDocument(document.value, options);
    if (!normalized) {
      issues.value = [
        {
          code: 'invalid-current-document',
          path: '$',
          message: '当前表单配置无效，无法保存。',
        },
      ];
      throw new Error('Form document validation failed.');
    }

    isSaving.value = true;
    issues.value = [];
    try {
      const response = await options.adapter.save(normalized);
      const resolved =
        response === null || response === undefined
          ? normalized
          : normalizeFormDocument(response, options);
      if (!resolved) {
        issues.value = [
          {
            code: 'invalid-saved-document',
            path: '$',
            message: '保存结果中的表单配置无效。',
          },
        ];
        throw new Error('Saved form document validation failed.');
      }

      commit(resolved);
      return document.value;
    } finally {
      isSaving.value = false;
    }
  }

  function reset() {
    document.value = cloneFormDocument(savedDocument.value);
    issues.value = [];
  }

  function commit(value: FormDocument) {
    const snapshot = cloneFormDocument(value);
    document.value = snapshot;
    savedDocument.value = cloneFormDocument(snapshot);
  }

  return { document, isLoading, isSaving, isDirty, issues, load, save, reset };
}

function resolveDocument(input: unknown, options: FormSchemaNormalizationOptions): FormDocument {
  const document = normalizeFormDocument(input, options);
  if (document) return document;

  throw new Error('Initial form document validation failed.');
}

function isSameDocument(left: FormDocument, right: FormDocument) {
  return JSON.stringify(left) === JSON.stringify(right);
}
