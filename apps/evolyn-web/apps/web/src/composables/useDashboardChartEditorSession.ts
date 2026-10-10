import type {
  BusinessDashboardChartWidget,
  BusinessDashboardDataset,
  BusinessDashboardLayout,
  BusinessDashboardWidgetDescriptor,
} from '@evolyn.do/dashboard';
import type { DashboardFormDataSource } from '~/types';
import { computed, shallowReadonly, shallowRef, toRaw } from 'vue';

export interface DashboardChartEditorDraft {
  mode: 'create' | 'edit';
  widget: BusinessDashboardChartWidget;
  dataset: BusinessDashboardDataset;
  source: DashboardFormDataSource;
}

interface DashboardChartEditorSessionOptions {
  commit: (widget: BusinessDashboardChartWidget, dataset: BusinessDashboardDataset) => void;
  createID?: () => string;
}

/**
 * 统计图编辑会话与仪表盘文档隔离：返回或取消只丢弃会话，保存时才原子提交
 * Dataset 与组件，避免产生没有数据源的半成品组件。
 */
export function useDashboardChartEditorSession(options: DashboardChartEditorSessionOptions) {
  const draft = shallowRef<DashboardChartEditorDraft | null>(null);
  const baseline = shallowRef('');
  const isDirty = computed(
    () =>
      Boolean(draft.value) &&
      (draft.value?.mode === 'create' || JSON.stringify(draft.value) !== baseline.value),
  );

  function beginCreate(input: {
    descriptor: BusinessDashboardWidgetDescriptor;
    layout: BusinessDashboardLayout;
    source: DashboardFormDataSource;
  }): DashboardChartEditorDraft {
    if (input.descriptor.type !== 'chart') {
      throw new Error('当前编辑页仅支持统计图组件');
    }
    const widgetID = createScopedID('widget', options.createID);
    const datasetID = createScopedID('dataset', options.createID);
    const next: DashboardChartEditorDraft = {
      mode: 'create',
      source: structuredClone(toRaw(input.source)),
      widget: {
        id: widgetID,
        type: 'chart',
        title: input.descriptor.defaultTitle,
        layout: { ...input.layout },
        datasetId: datasetID,
        settings: structuredClone(
          input.descriptor.defaultSettings,
        ) as BusinessDashboardChartWidget['settings'],
      },
      dataset: {
        id: datasetID,
        name: input.source.name,
        source: { type: 'form', formCode: input.source.code },
        query: {
          version: 1,
          sorts: [],
          paging: { page: 1, pageSize: 20 },
          projection: [],
          groupBy: [],
          aggregates: [],
        },
      },
    };
    adopt(next);
    return next;
  }

  function beginEdit(input: {
    widget: BusinessDashboardChartWidget;
    dataset: BusinessDashboardDataset;
    source: DashboardFormDataSource;
  }): DashboardChartEditorDraft {
    const next: DashboardChartEditorDraft = {
      mode: 'edit',
      widget: structuredClone(toRaw(input.widget)),
      dataset: structuredClone(toRaw(input.dataset)),
      source: structuredClone(toRaw(input.source)),
    };
    adopt(next);
    return next;
  }

  function updateWidget(
    update: (widget: BusinessDashboardChartWidget) => BusinessDashboardChartWidget,
  ) {
    if (!draft.value) return;
    draft.value = { ...draft.value, widget: update(structuredClone(draft.value.widget)) };
  }

  function updateDataset(update: (dataset: BusinessDashboardDataset) => BusinessDashboardDataset) {
    if (!draft.value) return;
    draft.value = { ...draft.value, dataset: update(structuredClone(draft.value.dataset)) };
  }

  /** 更换来源会清空旧字段绑定，防止不同发布 Schema 的 fieldId 被交叉复用。 */
  function replaceSource(source: DashboardFormDataSource) {
    if (!draft.value) return;
    draft.value = {
      ...draft.value,
      source: structuredClone(toRaw(source)),
      dataset: {
        ...draft.value.dataset,
        name: source.name,
        source: { type: 'form', formCode: source.code },
        query: {
          version: 1,
          sorts: [],
          paging: { page: 1, pageSize: 20 },
          projection: [],
          groupBy: [],
          aggregates: [],
        },
      },
      widget: {
        ...draft.value.widget,
        settings: {
          ...draft.value.widget.settings,
          encoding: { dimensions: [], metrics: [] },
        },
      },
    };
  }

  function commit() {
    if (!draft.value) return null;
    options.commit(structuredClone(draft.value.widget), structuredClone(draft.value.dataset));
    const committed = draft.value;
    clear();
    return committed;
  }

  function discard() {
    clear();
  }

  function adopt(next: DashboardChartEditorDraft) {
    draft.value = next;
    baseline.value = JSON.stringify(next);
  }

  function clear() {
    draft.value = null;
    baseline.value = '';
  }

  return {
    draft: shallowReadonly(draft),
    isDirty,
    beginCreate,
    beginEdit,
    updateWidget,
    updateDataset,
    replaceSource,
    commit,
    discard,
  };
}

function createScopedID(prefix: 'widget' | 'dataset', createID?: () => string): string {
  const value =
    createID?.() ??
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `${prefix}_${value.replaceAll('-', '')}`;
}
