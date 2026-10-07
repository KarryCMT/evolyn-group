import type {
  BusinessDashboardDocument,
  BusinessDashboardIssue,
} from '@evolyn.do/dashboard';
import type { DashboardDetail, DashboardDraftSaveResult } from '~/types';
import {
  cloneBusinessDashboardDocument,
  createEmptyBusinessDashboardDocument,
  normalizeBusinessDashboardDocument,
  useBusinessDashboardEditor,
} from '@evolyn.do/dashboard';
import { ApiError } from '@evolyn.do/utils';
import { computed, shallowRef, watch } from 'vue';
import { getDashboard, saveDashboardDraft } from '~/api/dashboard';

export type DashboardDesignerLoadStatus = 'idle' | 'loading' | 'ready' | 'error';
export type DashboardDesignerSaveStatus = 'idle' | 'saving' | 'saved' | 'error' | 'conflict';

export interface DashboardDesignerGateway {
  get: (code: string) => Promise<DashboardDetail>;
  save: (
    code: string,
    expectedRevision: number,
    document: BusinessDashboardDocument,
  ) => Promise<DashboardDraftSaveResult>;
}

export interface PrepareDashboardPreviewOptions {
  dirty: boolean;
  revision: number;
  save: () => Promise<DashboardDraftSaveResult | null>;
}

/** dirty 草稿必须先保存；失败返回 null，调用方据此保持在设计页。 */
export async function prepareDashboardPreview(
  options: PrepareDashboardPreviewOptions,
): Promise<number | null> {
  if (!options.dirty) return options.revision;
  const result = await options.save();
  return result?.draftRevision ?? null;
}

const defaultGateway: DashboardDesignerGateway = {
  get: getDashboard,
  save: saveDashboardDraft,
};

/**
 * 业务仪表盘工作区状态机。serverSnapshot 只在加载或服务端确认保存后推进，
 * 保存失败、校验失败和 409 冲突都保留 editingDocument，避免吞掉本地编辑。
 */
export function useDashboardDesigner(gateway: DashboardDesignerGateway = defaultGateway) {
  const detail = shallowRef<DashboardDetail | null>(null);
  const serverSnapshot = shallowRef<BusinessDashboardDocument | null>(null);
  const editingDocument = shallowRef<BusinessDashboardDocument>(
    createEmptyBusinessDashboardDocument(),
  );
  const draftRevision = shallowRef(0);
  const loadStatus = shallowRef<DashboardDesignerLoadStatus>('idle');
  const saveStatus = shallowRef<DashboardDesignerSaveStatus>('idle');
  const issues = shallowRef<BusinessDashboardIssue[]>([]);
  const errorMessage = shallowRef('');
  const focusedIssuePath = shallowRef('');
  let loadSequence = 0;

  const editor = useBusinessDashboardEditor({ document: editingDocument });
  const isDirty = computed(() => {
    const snapshot = serverSnapshot.value;
    return snapshot !== null && JSON.stringify(editingDocument.value) !== JSON.stringify(snapshot);
  });
  const isSaving = computed(() => saveStatus.value === 'saving');
  const isSaved = computed(
    () => Boolean(serverSnapshot.value) && !isDirty.value && saveStatus.value === 'saved',
  );
  const issueWidgetIds = computed(() => {
    const ids = new Set<string>();
    for (const issue of issues.value) {
      const index = widgetIndexFromPath(issue.path);
      const widget = index === null ? null : editingDocument.value.widgets[index];
      if (widget) ids.add(widget.id);
    }
    return [...ids];
  });

  watch(editingDocument, () => {
    if (isDirty.value && saveStatus.value === 'saved') saveStatus.value = 'idle';
  });

  async function load(code: string): Promise<boolean> {
    const sequence = ++loadSequence;
    loadStatus.value = 'loading';
    errorMessage.value = '';
    issues.value = [];
    focusedIssuePath.value = '';
    try {
      const response = await gateway.get(code);
      if (sequence !== loadSequence) return false;
      const normalized = normalizeBusinessDashboardDocument(response.draft);
      if (!normalized.document) {
        issues.value = normalized.issues;
        errorMessage.value = '服务端草稿无法进入设计器，请联系管理员检查协议版本。';
        loadStatus.value = 'error';
        return false;
      }
      adoptDetail(response, normalized.document);
      saveStatus.value = 'idle';
      loadStatus.value = 'ready';
      return true;
    } catch (error) {
      if (sequence !== loadSequence) return false;
      errorMessage.value =
        error instanceof ApiError ? error.message : '仪表盘加载失败，请稍后重试。';
      loadStatus.value = 'error';
      return false;
    }
  }

  async function save(code: string): Promise<DashboardDraftSaveResult | null> {
    if (saveStatus.value === 'saving') return null;
    const normalized = normalizeBusinessDashboardDocument(editingDocument.value);
    if (!normalized.document) {
      issues.value = normalized.issues;
      saveStatus.value = 'error';
      focusIssue(normalized.issues[0]);
      return null;
    }

    saveStatus.value = 'saving';
    issues.value = [];
    errorMessage.value = '';
    try {
      const response = await gateway.save(code, draftRevision.value, normalized.document);
      const accepted = normalizeBusinessDashboardDocument(response.document);
      if (!accepted.document) {
        issues.value = accepted.issues;
        errorMessage.value = '服务端返回了无法识别的规范化文档。';
        saveStatus.value = 'error';
        return null;
      }
      const snapshot = cloneBusinessDashboardDocument(accepted.document);
      serverSnapshot.value = snapshot;
      editingDocument.value = cloneBusinessDashboardDocument(snapshot);
      draftRevision.value = response.draftRevision;
      issues.value = [];
      focusedIssuePath.value = '';
      saveStatus.value = 'saved';
      return response;
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.errCode === 'DASHBOARD_DRAFT_CONFLICT') {
          saveStatus.value = 'conflict';
          errorMessage.value = '服务端草稿已更新。本地修改仍然保留，请决定是否重新加载。';
          return null;
        }
        if (error.errCode === 'DASHBOARD_SCHEMA_INVALID') {
          issues.value = extractIssues(error);
          focusIssue(issues.value[0]);
        }
        errorMessage.value = error.message;
      } else {
        errorMessage.value = '保存失败，本地修改仍然保留，请稍后重试。';
      }
      saveStatus.value = 'error';
      return null;
    }
  }

  /** 只有用户明确选择重新加载时才覆盖冲突中的本地文档。 */
  async function reloadServerVersion(code: string): Promise<boolean> {
    return load(code);
  }

  function focusIssue(issue?: BusinessDashboardIssue) {
    if (!issue) return;
    focusedIssuePath.value = issue.path;
    const index = widgetIndexFromPath(issue.path);
    const widget = index === null ? null : editingDocument.value.widgets[index];
    editor.selectWidget(widget?.id ?? null);
  }

  function adoptDetail(response: DashboardDetail, document: BusinessDashboardDocument) {
    const snapshot = cloneBusinessDashboardDocument(document);
    detail.value = response;
    serverSnapshot.value = snapshot;
    editingDocument.value = cloneBusinessDashboardDocument(snapshot);
    draftRevision.value = response.draftRevision;
    editor.selectWidget(null);
  }

  return {
    detail,
    serverSnapshot,
    editingDocument,
    draftRevision,
    loadStatus,
    saveStatus,
    issues,
    errorMessage,
    focusedIssuePath,
    isDirty,
    isSaving,
    isSaved,
    issueWidgetIds,
    ...editor,
    load,
    save,
    reloadServerVersion,
    focusIssue,
  };
}

function widgetIndexFromPath(path: string): number | null {
  const match = /^\$?\.?widgets\[(\d+)\]/.exec(path);
  return match ? Number(match[1]) : null;
}

function extractIssues(error: ApiError): BusinessDashboardIssue[] {
  const payload = error.data as { issues?: unknown } | undefined;
  if (!Array.isArray(payload?.issues)) return [];
  return payload.issues.flatMap((item) => {
    if (!isIssue(item)) return [];
    return [{ path: item.path, code: item.code, message: item.message }];
  });
}

function isIssue(value: unknown): value is BusinessDashboardIssue {
  if (typeof value !== 'object' || value === null) return false;
  const item = value as Record<string, unknown>;
  return (
    typeof item.path === 'string' &&
    typeof item.code === 'string' &&
    typeof item.message === 'string'
  );
}
