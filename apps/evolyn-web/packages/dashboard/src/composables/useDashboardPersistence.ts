import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue';
import {
  normalizeDashboardSchema,
  type DashboardDocument,
  type DashboardSchemaNormalizationOptions,
  type DashboardSchemaValidationIssue,
} from '../schema';

/** 接入应用实现数据来源；共享包不感知接口、权限、成员或租户。 */
export interface DashboardPersistenceAdapter<TType extends string> {
  load: () => Promise<unknown>;
  save: (document: DashboardDocument<TType>) => Promise<unknown>;
}

export interface UseDashboardPersistenceOptions<
  TType extends string,
> extends DashboardSchemaNormalizationOptions<TType> {
  initialDocument: DashboardDocument<TType>;
  adapter: DashboardPersistenceAdapter<TType>;
}

export interface DashboardPersistence<TType extends string> {
  document: ShallowRef<DashboardDocument<TType>>;
  isLoading: Readonly<ShallowRef<boolean>>;
  isSaving: Readonly<ShallowRef<boolean>>;
  isDirty: ComputedRef<boolean>;
  issues: Readonly<ShallowRef<DashboardSchemaValidationIssue[]>>;
  load: () => Promise<DashboardDocument<TType>>;
  save: () => Promise<DashboardDocument<TType>>;
  reset: () => void;
}

/**
 * 管理工作台文档的快照、脏状态和持久化流程。所有输入输出都会先经过 schema 归一化，
 * 因此历史 JSON、损坏数据与运行时字段不会越过应用和共享包之间的边界。
 */
export function useDashboardPersistence<TType extends string>(
  options: UseDashboardPersistenceOptions<TType>,
): DashboardPersistence<TType> {
  const initialDocument = resolveDocument(options.initialDocument, options);
  const document = shallowRef(cloneDocument(initialDocument));
  const savedDocument = shallowRef(cloneDocument(initialDocument));
  const isLoading = shallowRef(false);
  const isSaving = shallowRef(false);
  const issues = shallowRef<DashboardSchemaValidationIssue[]>([]);
  const isDirty = computed(() => !isSameDocument(document.value, savedDocument.value));

  async function load() {
    isLoading.value = true;
    issues.value = [];

    try {
      const input = await options.adapter.load();
      if (input === null || input === undefined) {
        commit(initialDocument);
        return document.value;
      }

      const resolved = normalizeDashboardSchema(input, options);
      if (!resolved) {
        issues.value = [
          {
            code: 'invalid-persisted-document',
            path: '$',
            message: '已保存的工作台配置无效，已恢复默认布局。',
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
    const normalized = normalizeDashboardSchema(document.value, options);
    if (!normalized) {
      issues.value = [
        {
          code: 'invalid-current-document',
          path: '$',
          message: '当前工作台配置无效，无法保存。',
        },
      ];
      throw new Error('Dashboard document validation failed.');
    }

    isSaving.value = true;
    issues.value = [];

    try {
      const response = await options.adapter.save(normalized);
      const resolved =
        response === null || response === undefined
          ? normalized
          : normalizeDashboardSchema(response, options);

      if (!resolved) {
        issues.value = [
          {
            code: 'invalid-saved-document',
            path: '$',
            message: '保存结果中的工作台配置无效。',
          },
        ];
        throw new Error('Saved dashboard document validation failed.');
      }

      commit(resolved);
      return document.value;
    } finally {
      isSaving.value = false;
    }
  }

  function reset() {
    document.value = cloneDocument(savedDocument.value);
    issues.value = [];
  }

  function commit(value: DashboardDocument<TType>) {
    const snapshot = cloneDocument(value);
    document.value = snapshot;
    savedDocument.value = cloneDocument(snapshot);
  }

  return { document, isLoading, isSaving, isDirty, issues, load, save, reset };
}

function resolveDocument<TType extends string>(
  input: unknown,
  options: DashboardSchemaNormalizationOptions<TType>,
): DashboardDocument<TType> {
  const document = normalizeDashboardSchema(input, options);
  if (document) return document;

  throw new Error('Initial dashboard document validation failed.');
}

/** schema 已限制为 JSON 数据，序列化克隆可确保编辑草稿和已保存快照完全隔离。 */
function cloneDocument<TType extends string>(
  document: DashboardDocument<TType>,
): DashboardDocument<TType> {
  return JSON.parse(JSON.stringify(document)) as DashboardDocument<TType>;
}

function isSameDocument<TType extends string>(
  left: DashboardDocument<TType>,
  right: DashboardDocument<TType>,
) {
  return JSON.stringify(left) === JSON.stringify(right);
}
