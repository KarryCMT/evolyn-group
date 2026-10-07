import { validateQuery } from '@evolyn.do/query';
import {
  BUSINESS_DASHBOARD_COLUMNS,
  BUSINESS_DASHBOARD_MAX_DATASETS,
  BUSINESS_DASHBOARD_MAX_WIDGETS,
  BUSINESS_DASHBOARD_VERSION,
  type BusinessDashboardChartSettings,
  type BusinessDashboardChartWidget,
  type BusinessDashboardDataset,
  type BusinessDashboardDocument,
  type BusinessDashboardFieldRef,
  type BusinessDashboardIssue,
  type BusinessDashboardNormalizationResult,
  type BusinessDashboardTableColumn,
  type BusinessDashboardTableSettings,
  type BusinessDashboardTableSort,
  type BusinessDashboardTableWidget,
  type BusinessDashboardWidget,
  type BusinessDashboardWidgetType,
} from './types.js';

const widgetTypes = new Set<BusinessDashboardWidgetType>(['chart', 'table']);
const chartVariants = new Set(['bar', 'line', 'pie']);
const legendPositions = new Set(['top', 'right', 'bottom', 'left']);
const tableFormats = new Set(['auto', 'text', 'decimal', 'money', 'percent', 'date', 'datetime']);
const tableDensities = new Set(['compact', 'default', 'comfortable']);
const tableAlignments = new Set(['left', 'center', 'right']);

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
  return structuredClone(document);
}

/**
 * 输出严格白名单化的业务仪表盘文档。协议只保存平台语义：函数、原始 SQL、
 * VChart/VTable spec 或运行时实例都会以路径级问题被拒绝。
 */
export function normalizeBusinessDashboardDocument(
  input: unknown,
): BusinessDashboardNormalizationResult {
  const issues: BusinessDashboardIssue[] = [];
  if (!isRecord(input))
    return invalid(issues, '$', 'DASHBOARD_DOCUMENT_INVALID', '仪表盘文档必须是对象');
  rejectUnknownKeys(
    input,
    ['version', 'settings', 'datasets', 'widgets', 'filters', 'interactions', 'publishScope'],
    '$',
    issues,
  );
  if (!isJsonValue(input)) {
    addIssue(
      issues,
      '$',
      'DASHBOARD_RUNTIME_CONFIG_FORBIDDEN',
      '仪表盘文档不得包含函数、实例或其他不可序列化值',
    );
  }
  if (input.version !== BUSINESS_DASHBOARD_VERSION) {
    addIssue(issues, 'version', 'DASHBOARD_VERSION_UNSUPPORTED', '不支持的仪表盘协议版本');
  }

  const desktop = readDesktopSettings(input.settings, issues);
  const datasets = readDatasets(input.datasets, issues);
  const filters = readIdentifiedList(input.filters, 'filters', issues);
  const interactions = readIdentifiedList(input.interactions, 'interactions', issues);
  const widgets = readWidgets(input.widgets, issues);
  validateBindings(datasets, widgets, issues);
  if (!isRecord(input.publishScope) || input.publishScope.type !== 'all') {
    addIssue(
      issues,
      'publishScope.type',
      'DASHBOARD_PUBLISH_SCOPE_INVALID',
      '当前仅支持全体成员范围',
    );
  } else {
    rejectUnknownKeys(input.publishScope, ['type'], 'publishScope', issues);
  }
  if (issues.length || !desktop) return { document: null, issues };

  return {
    document: {
      version: BUSINESS_DASHBOARD_VERSION,
      settings: { desktop },
      datasets,
      widgets,
      filters,
      interactions,
      publishScope: { type: 'all' },
    },
    issues: [],
  };
}

function readDesktopSettings(
  value: unknown,
  issues: BusinessDashboardIssue[],
): BusinessDashboardDocument['settings']['desktop'] | null {
  if (!isRecord(value) || !isRecord(value.desktop)) {
    addIssue(issues, 'settings.desktop', 'DASHBOARD_LAYOUT_INVALID', '桌面画布配置无效');
    return null;
  }
  rejectUnknownKeys(value, ['desktop'], 'settings', issues);
  rejectUnknownKeys(value.desktop, ['columns', 'rowHeight'], 'settings.desktop', issues);
  if (
    value.desktop.columns !== BUSINESS_DASHBOARD_COLUMNS ||
    !isPositiveInteger(value.desktop.rowHeight)
  ) {
    addIssue(
      issues,
      'settings.desktop',
      'DASHBOARD_LAYOUT_INVALID',
      '桌面画布必须使用 12 列且行高为正整数',
    );
    return null;
  }
  return { columns: BUSINESS_DASHBOARD_COLUMNS, rowHeight: value.desktop.rowHeight };
}

function readDatasets(
  value: unknown,
  issues: BusinessDashboardIssue[],
): BusinessDashboardDataset[] {
  if (!Array.isArray(value)) {
    addIssue(issues, 'datasets', 'DASHBOARD_DOCUMENT_INVALID', 'datasets 必须是数组');
    return [];
  }
  if (value.length > BUSINESS_DASHBOARD_MAX_DATASETS) {
    addIssue(
      issues,
      'datasets',
      'DASHBOARD_DATASET_LIMIT_EXCEEDED',
      `单个仪表盘最多允许 ${BUSINESS_DASHBOARD_MAX_DATASETS} 个 Dataset`,
    );
  }
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    const path = `datasets[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', 'Dataset 必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['id', 'name', 'source', 'query'], path, issues);
    const id = readID(item.id, `${path}.id`, seen, issues);
    const name = readRequiredString(
      item.name,
      `${path}.name`,
      'DASHBOARD_DATASET_NAME_REQUIRED',
      issues,
    );
    const source = readDatasetSource(item.source, `${path}.source`, issues);
    const queryResult = validateQuery(item.query);
    for (const diagnostic of queryResult.diagnostics) {
      addIssue(
        issues,
        diagnostic.path === '$' ? `${path}.query` : `${path}.query.${diagnostic.path}`,
        diagnostic.code,
        diagnostic.message,
      );
    }
    if (!id || !name || !source || !queryResult.document) return [];
    return [{ id, name, source, query: queryResult.document }];
  });
}

function readDatasetSource(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardDataset['source'] | null {
  if (!isRecord(value)) {
    addIssue(issues, path, 'DASHBOARD_DATA_SOURCE_INVALID', 'Dataset 数据源必须是对象');
    return null;
  }
  rejectUnknownKeys(value, ['type', 'formCode'], path, issues);
  const formCode = readRequiredString(
    value.formCode,
    `${path}.formCode`,
    'DASHBOARD_DATA_SOURCE_INVALID',
    issues,
  );
  if (value.type !== 'form' || !formCode) {
    addIssue(issues, `${path}.type`, 'DASHBOARD_DATA_SOURCE_INVALID', '当前仅支持表单数据源');
    return null;
  }
  return { type: 'form', formCode };
}

function readIdentifiedList(
  value: unknown,
  path: 'filters' | 'interactions',
  issues: BusinessDashboardIssue[],
): Array<{ id: string }> {
  if (!Array.isArray(value)) {
    addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', `${path} 必须是数组`);
    return [];
  }
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    if (!isRecord(item)) {
      addIssue(issues, `${path}[${index}]`, 'DASHBOARD_DOCUMENT_INVALID', '配置必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['id'], `${path}[${index}]`, issues);
    const id = readID(item.id, `${path}[${index}].id`, seen, issues);
    return id ? [{ id }] : [];
  });
}

function readWidgets(value: unknown, issues: BusinessDashboardIssue[]): BusinessDashboardWidget[] {
  if (!Array.isArray(value)) {
    addIssue(issues, 'widgets', 'DASHBOARD_DOCUMENT_INVALID', 'widgets 必须是数组');
    return [];
  }
  if (value.length > BUSINESS_DASHBOARD_MAX_WIDGETS) {
    addIssue(
      issues,
      'widgets',
      'DASHBOARD_WIDGET_LIMIT_EXCEEDED',
      `单个仪表盘最多允许 ${BUSINESS_DASHBOARD_MAX_WIDGETS} 个组件`,
    );
  }
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    const path = `widgets[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', '组件必须是对象');
      return [];
    }
    rejectUnknownKeys(
      item,
      ['id', 'type', 'title', 'layout', 'datasetId', 'settings'],
      path,
      issues,
    );
    const id = readID(item.id, `${path}.id`, seen, issues);
    const type = typeof item.type === 'string' ? item.type : '';
    if (!widgetTypes.has(type as BusinessDashboardWidgetType)) {
      addIssue(issues, `${path}.type`, 'DASHBOARD_WIDGET_TYPE_UNSUPPORTED', '不支持的组件类型');
    }
    const layout = readLayout(item.layout, `${path}.layout`, issues);
    const title = readOptionalString(item.title, `${path}.title`, issues);
    const datasetId = readOptionalString(item.datasetId, `${path}.datasetId`, issues);
    const settings =
      type === 'chart'
        ? readChartSettings(item.settings, `${path}.settings`, issues)
        : type === 'table'
          ? readTableSettings(item.settings, `${path}.settings`, issues)
          : null;
    if (!id || !widgetTypes.has(type as BusinessDashboardWidgetType) || !layout || !settings)
      return [];
    const base = {
      id,
      ...(title ? { title } : {}),
      layout,
      ...(datasetId ? { datasetId } : {}),
    };
    return type === 'chart'
      ? [{ ...base, type: 'chart', settings } as BusinessDashboardChartWidget]
      : [{ ...base, type: 'table', settings } as BusinessDashboardTableWidget];
  });
}

function readChartSettings(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardChartSettings | null {
  if (!isRecord(value) || !isRecord(value.encoding) || !isRecord(value.display)) {
    addIssue(issues, path, 'DASHBOARD_CHART_CONFIG_INVALID', '统计图配置必须是受控对象');
    return null;
  }
  rejectUnknownKeys(value, ['encoding', 'display'], path, issues);
  rejectUnknownKeys(value.encoding, ['dimensions', 'metrics'], `${path}.encoding`, issues);
  const dimensions = readFieldBindings(
    value.encoding.dimensions,
    `${path}.encoding.dimensions`,
    issues,
  );
  const metrics = readMetricBindings(value.encoding.metrics, `${path}.encoding.metrics`, issues);
  const display = value.display;
  rejectUnknownKeys(
    display,
    ['variant', 'orientation', 'stack', 'legend', 'labels'],
    `${path}.display`,
    issues,
  );
  if (
    !chartVariants.has(String(display.variant)) ||
    (display.orientation !== 'vertical' && display.orientation !== 'horizontal') ||
    (display.stack !== 'none' && display.stack !== 'normal') ||
    !isRecord(display.legend) ||
    typeof display.legend.visible !== 'boolean' ||
    !legendPositions.has(String(display.legend.position)) ||
    !isRecord(display.labels) ||
    typeof display.labels.visible !== 'boolean'
  ) {
    addIssue(issues, `${path}.display`, 'DASHBOARD_CHART_CONFIG_INVALID', '统计图展示配置无效');
    return null;
  }
  rejectUnknownKeys(display.legend, ['visible', 'position'], `${path}.display.legend`, issues);
  rejectUnknownKeys(display.labels, ['visible'], `${path}.display.labels`, issues);
  return {
    encoding: { dimensions, metrics },
    display: {
      variant: display.variant as BusinessDashboardChartSettings['display']['variant'],
      orientation: display.orientation,
      stack: display.stack,
      legend: {
        visible: display.legend.visible,
        position: display.legend
          .position as BusinessDashboardChartSettings['display']['legend']['position'],
      },
      labels: { visible: display.labels.visible },
    },
  };
}

function readFieldBindings(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardChartSettings['encoding']['dimensions'] {
  if (!Array.isArray(value)) {
    addIssue(issues, path, 'DASHBOARD_CHART_ENCODING_INVALID', '维度必须是数组');
    return [];
  }
  return value.flatMap((item, index) => {
    const itemPath = `${path}[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, itemPath, 'DASHBOARD_CHART_ENCODING_INVALID', '维度必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['field', 'label'], itemPath, issues);
    const field = readFieldRef(item.field, `${itemPath}.field`, issues);
    const label = readOptionalString(item.label, `${itemPath}.label`, issues);
    return field ? [{ field, ...(label ? { label } : {}) }] : [];
  });
}

function readMetricBindings(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardChartSettings['encoding']['metrics'] {
  if (!Array.isArray(value)) {
    addIssue(issues, path, 'DASHBOARD_CHART_ENCODING_INVALID', '指标必须是数组');
    return [];
  }
  return value.flatMap((item, index) => {
    const itemPath = `${path}[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, itemPath, 'DASHBOARD_CHART_ENCODING_INVALID', '指标必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['aggregateAlias', 'label'], itemPath, issues);
    const aggregateAlias = readRequiredString(
      item.aggregateAlias,
      `${itemPath}.aggregateAlias`,
      'DASHBOARD_CHART_ENCODING_INVALID',
      issues,
    );
    const label = readOptionalString(item.label, `${itemPath}.label`, issues);
    return aggregateAlias ? [{ aggregateAlias, ...(label ? { label } : {}) }] : [];
  });
}

function readTableSettings(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardTableSettings | null {
  if (
    !isRecord(value) ||
    !isRecord(value.pagination) ||
    !isRecord(value.display) ||
    !Array.isArray(value.columns) ||
    !Array.isArray(value.sorts)
  ) {
    addIssue(issues, path, 'DASHBOARD_TABLE_CONFIG_INVALID', '明细表配置必须是受控对象');
    return null;
  }
  rejectUnknownKeys(value, ['columns', 'sorts', 'pagination', 'display'], path, issues);
  const columns = readTableColumns(value.columns, `${path}.columns`, issues);
  const sorts = readTableSorts(value.sorts, `${path}.sorts`, issues);
  rejectUnknownKeys(value.pagination, ['pageSize'], `${path}.pagination`, issues);
  if (!isPositiveInteger(value.pagination.pageSize) || value.pagination.pageSize > 100) {
    addIssue(
      issues,
      `${path}.pagination.pageSize`,
      'DASHBOARD_TABLE_PAGE_SIZE_INVALID',
      '明细表每页数量必须为 1 到 100',
    );
  }
  const display = value.display;
  rejectUnknownKeys(
    display,
    ['density', 'striped', 'bordered', 'showHeader', 'emptyText'],
    `${path}.display`,
    issues,
  );
  if (
    !tableDensities.has(String(display.density)) ||
    typeof display.striped !== 'boolean' ||
    typeof display.bordered !== 'boolean' ||
    typeof display.showHeader !== 'boolean' ||
    typeof display.emptyText !== 'string'
  ) {
    addIssue(issues, `${path}.display`, 'DASHBOARD_TABLE_CONFIG_INVALID', '明细表展示配置无效');
    return null;
  }
  if (!isPositiveInteger(value.pagination.pageSize) || value.pagination.pageSize > 100) return null;
  return {
    columns,
    sorts,
    pagination: { pageSize: value.pagination.pageSize },
    display: {
      density: display.density as BusinessDashboardTableSettings['display']['density'],
      striped: display.striped,
      bordered: display.bordered,
      showHeader: display.showHeader,
      emptyText: display.emptyText.trim().slice(0, 100),
    },
  };
}

function readTableColumns(
  value: unknown[],
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardTableColumn[] {
  const seen = new Set<string>();
  return value.flatMap((item, index) => {
    const itemPath = `${path}[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, itemPath, 'DASHBOARD_TABLE_COLUMN_INVALID', '表格列必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['id', 'field', 'label', 'width', 'align', 'format'], itemPath, issues);
    const id = readID(item.id, `${itemPath}.id`, seen, issues);
    const field = readFieldRef(item.field, `${itemPath}.field`, issues);
    const label = readOptionalString(item.label, `${itemPath}.label`, issues);
    const width =
      item.width === undefined
        ? undefined
        : isPositiveInteger(item.width) && item.width <= 1200
          ? item.width
          : null;
    if (width === null) {
      addIssue(issues, `${itemPath}.width`, 'DASHBOARD_TABLE_COLUMN_INVALID', '列宽无效');
    }
    if (!tableAlignments.has(String(item.align)) || !tableFormats.has(String(item.format))) {
      addIssue(issues, itemPath, 'DASHBOARD_TABLE_COLUMN_INVALID', '列对齐或格式无效');
    }
    if (!id || !field || width === null) return [];
    return [
      {
        id,
        field,
        ...(label ? { label } : {}),
        ...(width ? { width } : {}),
        align: item.align as BusinessDashboardTableColumn['align'],
        format: item.format as BusinessDashboardTableColumn['format'],
      },
    ];
  });
}

function readTableSorts(
  value: unknown[],
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardTableSort[] {
  return value.flatMap((item, index) => {
    const itemPath = `${path}[${index}]`;
    if (!isRecord(item)) {
      addIssue(issues, itemPath, 'DASHBOARD_TABLE_SORT_INVALID', '表格排序必须是对象');
      return [];
    }
    rejectUnknownKeys(item, ['field', 'direction'], itemPath, issues);
    const field = readFieldRef(item.field, `${itemPath}.field`, issues);
    if (item.direction !== 'asc' && item.direction !== 'desc') {
      addIssue(issues, `${itemPath}.direction`, 'DASHBOARD_TABLE_SORT_INVALID', '排序方向无效');
      return [];
    }
    return field ? [{ field, direction: item.direction }] : [];
  });
}

function readFieldRef(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardFieldRef | null {
  if (!isRecord(value)) {
    addIssue(issues, path, 'DASHBOARD_FIELD_REF_INVALID', '字段引用必须是对象');
    return null;
  }
  rejectUnknownKeys(value, ['fieldId'], path, issues);
  const fieldId = readRequiredString(
    value.fieldId,
    `${path}.fieldId`,
    'DASHBOARD_FIELD_REF_INVALID',
    issues,
  );
  return fieldId ? { fieldId } : null;
}

function validateBindings(
  datasets: BusinessDashboardDataset[],
  widgets: BusinessDashboardWidget[],
  issues: BusinessDashboardIssue[],
) {
  const byID = new Map(datasets.map((dataset) => [dataset.id, dataset]));
  widgets.forEach((widget, index) => {
    if (!widget.datasetId) return;
    const dataset = byID.get(widget.datasetId);
    if (!dataset) {
      addIssue(
        issues,
        `widgets[${index}].datasetId`,
        'DASHBOARD_DATASET_NOT_FOUND',
        '组件引用的 Dataset 不存在',
      );
      return;
    }
    const projected = new Set(dataset.query.projection ?? []);
    const grouped = new Set(dataset.query.groupBy ?? []);
    const aggregateAliases = new Set(
      (dataset.query.aggregates ?? []).map((aggregate) => aggregate.alias),
    );
    if (widget.type === 'chart') {
      widget.settings.encoding.dimensions.forEach((dimension, dimensionIndex) => {
        if (!grouped.has(dimension.field.fieldId) && !projected.has(dimension.field.fieldId)) {
          addIssue(
            issues,
            `widgets[${index}].settings.encoding.dimensions[${dimensionIndex}].field.fieldId`,
            'DASHBOARD_FIELD_BINDING_INVALID',
            '维度字段不在 Dataset 输出中',
          );
        }
      });
      widget.settings.encoding.metrics.forEach((metric, metricIndex) => {
        if (!aggregateAliases.has(metric.aggregateAlias)) {
          addIssue(
            issues,
            `widgets[${index}].settings.encoding.metrics[${metricIndex}].aggregateAlias`,
            'DASHBOARD_AGGREGATE_BINDING_INVALID',
            '指标未引用 Dataset 聚合结果',
          );
        }
      });
    } else {
      widget.settings.columns.forEach((column, columnIndex) => {
        if (!projected.has(column.field.fieldId)) {
          addIssue(
            issues,
            `widgets[${index}].settings.columns[${columnIndex}].field.fieldId`,
            'DASHBOARD_FIELD_BINDING_INVALID',
            '明细列字段不在 Dataset 投影中',
          );
        }
      });
    }
  });
}

function readLayout(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): BusinessDashboardWidget['layout'] | null {
  if (!isRecord(value)) {
    addIssue(issues, path, 'DASHBOARD_LAYOUT_INVALID', '组件布局必须是对象');
    return null;
  }
  rejectUnknownKeys(value, ['x', 'y', 'w', 'h'], path, issues);
  if (
    !isNonNegativeInteger(value.x) ||
    !isNonNegativeInteger(value.y) ||
    !isPositiveInteger(value.w) ||
    !isPositiveInteger(value.h) ||
    value.x + value.w > BUSINESS_DASHBOARD_COLUMNS
  ) {
    addIssue(issues, path, 'DASHBOARD_LAYOUT_INVALID', '组件布局必须位于 12 列画布内');
    return null;
  }
  return { x: value.x, y: value.y, w: value.w, h: value.h };
}

function readID(
  value: unknown,
  path: string,
  seen: Set<string>,
  issues: BusinessDashboardIssue[],
): string {
  const id = typeof value === 'string' ? value.trim() : '';
  if (!id) addIssue(issues, path, 'DASHBOARD_ID_REQUIRED', '标识不能为空');
  else if (seen.has(id)) addIssue(issues, path, 'DASHBOARD_ID_DUPLICATED', `标识 ${id} 重复`);
  else seen.add(id);
  return id;
}

function readRequiredString(
  value: unknown,
  path: string,
  code: string,
  issues: BusinessDashboardIssue[],
): string {
  const result = typeof value === 'string' ? value.trim() : '';
  if (!result) addIssue(issues, path, code, '该字段不能为空');
  return result;
}

function readOptionalString(
  value: unknown,
  path: string,
  issues: BusinessDashboardIssue[],
): string | undefined {
  if (value === undefined) return undefined;
  if (typeof value !== 'string') {
    addIssue(issues, path, 'DASHBOARD_DOCUMENT_INVALID', '该字段必须是字符串');
    return undefined;
  }
  return value.trim() || undefined;
}

function rejectUnknownKeys(
  value: Record<string, unknown>,
  allowed: readonly string[],
  path: string,
  issues: BusinessDashboardIssue[],
) {
  const accepted = new Set(allowed);
  for (const key of Object.keys(value)) {
    if (!accepted.has(key)) {
      addIssue(
        issues,
        path === '$' ? key : `${path}.${key}`,
        'DASHBOARD_RUNTIME_CONFIG_FORBIDDEN',
        '包含未受支持或渲染器私有配置',
      );
    }
  }
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
