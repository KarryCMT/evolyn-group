import {
  BUSINESS_DASHBOARD_COLUMNS,
  BUSINESS_DASHBOARD_VERSION,
  type BusinessDashboardDocument,
  type BusinessDashboardIssue,
  type BusinessDashboardNormalizationResult,
  type BusinessDashboardWidget,
  type BusinessDashboardWidgetType,
} from './types.js';

const widgetTypes = new Set<BusinessDashboardWidgetType>(['chart', 'table']);

/** 新建资产的唯一空画布定义；不注入任何演示组件或假数据。 */
export function createEmptyBusinessDashboardDocument(): BusinessDashboardDocument {
  return {
    version: BUSINESS_DASHBOARD_VERSION,
    settings: { desktop: { columns: BUSINESS_DASHBOARD_COLUMNS, rowHeight: 80 } },
    datasets: [],
    widgets: [],
    filters: [],
    interactions: [],
    publishScope: { type: 'all' },
  };
}

/** 深克隆业务文档，确保编辑态和服务端快照互不共享引用。 */
export function cloneBusinessDashboardDocument(
  document: BusinessDashboardDocument,
): BusinessDashboardDocument {
  return JSON.parse(JSON.stringify(document)) as BusinessDashboardDocument;
}

/**
 * 前端镜像阶段一后端协议并输出白名单字段。服务端仍是最终裁决者；这里负责
 * 在进入设计器前拦住损坏草稿，并让本地编辑拥有路径级反馈。
 */
export function normalizeBusinessDashboardDocument(
  input: unknown,
): BusinessDashboardNormalizationResult {
  const issues: BusinessDashboardIssue[] = [];
  if (!isRecord(input))
    return invalid(issues, '$', 'DASHBOARD_DOCUMENT_INVALID', '仪表盘文档必须是对象');
  if (input.version !== BUSINESS_DASHBOARD_VERSION) {
    addIssue(issues, 'version', 'DASHBOARD_VERSION_UNSUPPORTED', '不支持的仪表盘协议版本');
  }

  const desktop =
    isRecord(input.settings) && isRecord(input.settings.desktop) ? input.settings.desktop : null;
  if (
    !desktop ||
    desktop.columns !== BUSINESS_DASHBOARD_COLUMNS ||
    !isPositiveInteger(desktop.rowHeight)
  ) {
    addIssue(
      issues,
      'settings.desktop',
      'DASHBOARD_LAYOUT_INVALID',
      '桌面画布必须使用 12 列且行高为正整数',
    );
  }

  const datasets = readIdentifiedList(input.datasets, 'datasets', issues);
  const filters = readIdentifiedList(input.filters, 'filters', issues);
  const interactions = readIdentifiedList(input.interactions, 'interactions', issues);
  const widgets = readWidgets(input.widgets, issues);
  if (!isRecord(input.publishScope) || input.publishScope.type !== 'all') {
    addIssue(
      issues,
      'publishScope.type',
      'DASHBOARD_PUBLISH_SCOPE_INVALID',
      '当前仅支持全体成员范围',
    );
  }
  if (issues.length || !desktop) return { document: null, issues };

  return {
    document: {
      version: BUSINESS_DASHBOARD_VERSION,
      settings: {
        desktop: {
          columns: BUSINESS_DASHBOARD_COLUMNS,
          rowHeight: desktop.rowHeight as number,
        },
      },
      datasets,
      widgets,
      filters,
      interactions,
      publishScope: { type: 'all' },
    },
    issues: [],
  };
}

function readIdentifiedList(
  value: unknown,
  path: 'datasets' | 'filters' | 'interactions',
  issues: BusinessDashboardIssue[],
): Array<{ id: string }> {
  if (!Array.isArray(value)) {
    addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', `${path} 必须是数组`);
    return [];
  }
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    const id = isRecord(item) && typeof item.id === 'string' ? item.id.trim() : '';
    const itemPath = `${path}[${index}].id`;
    if (!id) {
      addIssue(issues, itemPath, 'DASHBOARD_ID_REQUIRED', '标识不能为空');
      return [];
    }
    if (seen.has(id)) {
      addIssue(issues, itemPath, 'DASHBOARD_ID_DUPLICATED', `标识 ${id} 重复`);
      return [];
    }
    seen.add(id);
    return [{ id }];
  });
}

function readWidgets(value: unknown, issues: BusinessDashboardIssue[]): BusinessDashboardWidget[] {
  if (!Array.isArray(value)) {
    addIssue(issues, 'widgets', 'DASHBOARD_DOCUMENT_INVALID', 'widgets 必须是数组');
    return [];
  }
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    const path = `widgets[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', '组件必须是对象');
      return [];
    }
    const id = typeof item.id === 'string' ? item.id.trim() : '';
    const type = typeof item.type === 'string' ? item.type : '';
    const layout = isRecord(item.layout) ? item.layout : null;
    if (!id) addIssue(issues, `${path}.id`, 'DASHBOARD_ID_REQUIRED', '组件标识不能为空');
    else if (seen.has(id))
      addIssue(issues, `${path}.id`, 'DASHBOARD_ID_DUPLICATED', `组件标识 ${id} 重复`);
    else seen.add(id);
    if (!widgetTypes.has(type as BusinessDashboardWidgetType)) {
      addIssue(issues, `${path}.type`, 'DASHBOARD_WIDGET_TYPE_UNSUPPORTED', '不支持的组件类型');
    }
    if (!isValidLayout(layout)) {
      addIssue(
        issues,
        `${path}.layout`,
        'DASHBOARD_LAYOUT_INVALID',
        '组件布局必须位于 12 列画布内',
      );
    }
    if (item.settings !== undefined && (!isRecord(item.settings) || !isJsonValue(item.settings))) {
      addIssue(
        issues,
        `${path}.settings`,
        'DASHBOARD_WIDGET_SETTINGS_INVALID',
        '组件设置必须是可序列化对象',
      );
    }
    if (!id || !widgetTypes.has(type as BusinessDashboardWidgetType) || !isValidLayout(layout))
      return [];

    return [
      {
        id,
        type: type as BusinessDashboardWidgetType,
        ...(typeof item.title === 'string' && item.title.trim()
          ? { title: item.title.trim() }
          : {}),
        layout: {
          x: layout.x as number,
          y: layout.y as number,
          w: layout.w as number,
          h: layout.h as number,
        },
        ...(typeof item.datasetId === 'string' && item.datasetId.trim()
          ? { datasetId: item.datasetId.trim() }
          : {}),
        ...(isRecord(item.settings) ? { settings: { ...item.settings } } : {}),
      },
    ];
  });
}

function isValidLayout(
  value: Record<string, unknown> | null,
): value is Record<'x' | 'y' | 'w' | 'h', number> {
  return Boolean(
    value &&
    isNonNegativeInteger(value.x) &&
    isNonNegativeInteger(value.y) &&
    isPositiveInteger(value.w) &&
    isPositiveInteger(value.h) &&
    (value.x as number) + (value.w as number) <= BUSINESS_DASHBOARD_COLUMNS,
  );
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0;
}

function isPositiveInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value > 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isJsonValue(value: unknown, seen = new WeakSet<object>()): boolean {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return true;
  if (typeof value === 'number') return Number.isFinite(value);
  if (typeof value !== 'object' || seen.has(value)) return false;
  seen.add(value);
  const valid = Array.isArray(value)
    ? value.every((item) => isJsonValue(item, seen))
    : (Object.getPrototypeOf(value) === Object.prototype ||
        Object.getPrototypeOf(value) === null) &&
      Object.values(value).every((item) => isJsonValue(item, seen));
  seen.delete(value);
  return valid;
}

function addIssue(issues: BusinessDashboardIssue[], path: string, code: string, message: string) {
  issues.push({ path, code, message });
}

function invalid(
  issues: BusinessDashboardIssue[],
  path: string,
  code: string,
  message: string,
): BusinessDashboardNormalizationResult {
  addIssue(issues, path, code, message);
  return { document: null, issues };
}
