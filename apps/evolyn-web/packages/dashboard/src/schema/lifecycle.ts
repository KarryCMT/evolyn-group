import { DASHBOARD_SCHEMA_VERSION, type DashboardSchema, type DashboardWidget } from './types';

export interface DashboardSchemaValidationIssue {
  code: string;
  path: string;
  message: string;
}

export type DashboardSchemaValidationResult<TType extends string> =
  | { valid: true; schema: DashboardSchema<TType> }
  | { valid: false; issues: DashboardSchemaValidationIssue[] };

export interface DashboardSchemaNormalizationOptions<TType extends string> {
  /** 接入应用可在此收窄业务卡片类型，避免未知卡片进入运行时渲染。 */
  isWidgetType?: (type: string) => type is TType;
}

/**
 * 兼容尚未保存 version 字段的早期 JSON；其布局字段与 v1 一致，因此只需补齐版本号。
 * 后续版本迁移按 version 逐级追加在此处，调用方始终传入任意历史 JSON 即可。
 */
export function migrateDashboardSchema(input: unknown): unknown {
  if (!isRecord(input) || 'version' in input) return input;

  return { ...input, version: DASHBOARD_SCHEMA_VERSION };
}

/** 校验迁移后的 schema 是否可安全进入渲染器或设计器。 */
export function validateDashboardSchema<TType extends string>(
  input: unknown,
  options: DashboardSchemaNormalizationOptions<TType> = {},
): DashboardSchemaValidationResult<TType> {
  const migrated = migrateDashboardSchema(input);
  const issues: DashboardSchemaValidationIssue[] = [];
  if (!isRecord(migrated)) {
    return invalid(issues, 'invalid-root', '$', '工作台配置必须是对象。');
  }

  if (migrated.version !== DASHBOARD_SCHEMA_VERSION) {
    return invalid(
      issues,
      'unsupported-version',
      '$.version',
      `暂不支持工作台配置版本 ${String(migrated.version)}。`,
    );
  }
  if (!Array.isArray(migrated.widgets)) {
    return invalid(issues, 'invalid-widgets', '$.widgets', 'widgets 必须是数组。');
  }

  const ids = new Set<string>();
  const widgets = migrated.widgets.flatMap((value, index) => {
    const widget = validateWidget(value, index, ids, issues, options);
    return widget ? [widget] : [];
  });
  if (issues.length) return { valid: false, issues };

  return {
    valid: true,
    schema: { version: DASHBOARD_SCHEMA_VERSION, widgets },
  };
}

/**
 * 返回仅包含可持久化字段的新 schema。无效数据返回 null，由应用决定回退默认布局或展示错误。
 */
export function normalizeDashboardSchema<TType extends string>(
  input: unknown,
  options: DashboardSchemaNormalizationOptions<TType> = {},
): DashboardSchema<TType> | null {
  const result = validateDashboardSchema(input, options);
  if (!result.valid) return null;

  return {
    ...result.schema,
    widgets: result.schema.widgets.map((widget) => ({
      ...widget,
      config: widget.config ? { ...widget.config } : undefined,
    })),
  };
}

function validateWidget<TType extends string>(
  value: unknown,
  index: number,
  ids: Set<string>,
  issues: DashboardSchemaValidationIssue[],
  options: DashboardSchemaNormalizationOptions<TType>,
): DashboardWidget<TType> | null {
  const path = `$.widgets[${index}]`;
  if (!isRecord(value)) {
    addIssue(issues, 'invalid-widget', path, '卡片必须是对象。');
    return null;
  }

  const issueCount = issues.length;
  const id = readText(value.id, `${path}.id`, issues);
  const type = readText(value.type, `${path}.type`, issues);
  const title = readText(value.title, `${path}.title`, issues);
  const x = readInteger(value.x, `${path}.x`, issues, false);
  const y = readInteger(value.y, `${path}.y`, issues, false);
  const w = readInteger(value.w, `${path}.w`, issues, true);
  const h = readInteger(value.h, `${path}.h`, issues, true);
  const minW = readOptionalInteger(value.minW, `${path}.minW`, issues);
  const minH = readOptionalInteger(value.minH, `${path}.minH`, issues);
  const maxW = readOptionalInteger(value.maxW, `${path}.maxW`, issues);
  const maxH = readOptionalInteger(value.maxH, `${path}.maxH`, issues);

  if (id && ids.has(id))
    addIssue(issues, 'duplicate-widget-id', `${path}.id`, '卡片 id 不能重复。');
  if (id) ids.add(id);
  if (type && options.isWidgetType && !options.isWidgetType(type)) {
    addIssue(issues, 'unknown-widget-type', `${path}.type`, `不支持卡片类型 ${type}。`);
  }
  if (minW !== undefined && maxW !== undefined && minW > maxW) {
    addIssue(issues, 'invalid-width-range', path, 'minW 不能大于 maxW。');
  }
  if (minH !== undefined && maxH !== undefined && minH > maxH) {
    addIssue(issues, 'invalid-height-range', path, 'minH 不能大于 maxH。');
  }
  if (w !== null && minW !== undefined && w < minW) {
    addIssue(issues, 'width-below-minimum', `${path}.w`, 'w 不能小于 minW。');
  }
  if (w !== null && maxW !== undefined && w > maxW) {
    addIssue(issues, 'width-above-maximum', `${path}.w`, 'w 不能大于 maxW。');
  }
  if (h !== null && minH !== undefined && h < minH) {
    addIssue(issues, 'height-below-minimum', `${path}.h`, 'h 不能小于 minH。');
  }
  if (h !== null && maxH !== undefined && h > maxH) {
    addIssue(issues, 'height-above-maximum', `${path}.h`, 'h 不能大于 maxH。');
  }
  if (!isOptionalBoolean(value.noMove)) {
    addIssue(issues, 'invalid-no-move', `${path}.noMove`, 'noMove 必须是布尔值。');
  }
  if (!isOptionalBoolean(value.noResize)) {
    addIssue(issues, 'invalid-no-resize', `${path}.noResize`, 'noResize 必须是布尔值。');
  }
  if (value.presetKey !== undefined && typeof value.presetKey !== 'string') {
    addIssue(issues, 'invalid-preset-key', `${path}.presetKey`, 'presetKey 必须是字符串。');
  }
  if (value.config !== undefined && (!isRecord(value.config) || !isJsonValue(value.config))) {
    addIssue(issues, 'invalid-config', `${path}.config`, 'config 必须是可序列化的对象。');
  }
  if (
    issues.length > issueCount ||
    !id ||
    !type ||
    !title ||
    x === null ||
    y === null ||
    w === null ||
    h === null
  ) {
    return null;
  }

  return {
    id,
    type: type as TType,
    title,
    x,
    y,
    w,
    h,
    ...(minW === undefined ? {} : { minW }),
    ...(minH === undefined ? {} : { minH }),
    ...(maxW === undefined ? {} : { maxW }),
    ...(maxH === undefined ? {} : { maxH }),
    ...(value.noMove === undefined ? {} : { noMove: value.noMove }),
    ...(value.noResize === undefined ? {} : { noResize: value.noResize }),
    ...(value.config === undefined ? {} : { config: { ...value.config } }),
    ...(typeof value.presetKey === 'string' ? { presetKey: value.presetKey } : {}),
  };
}

function readText(
  value: unknown,
  path: string,
  issues: DashboardSchemaValidationIssue[],
): string | null {
  if (typeof value === 'string' && value.trim()) return value;

  addIssue(issues, 'invalid-text', path, '必须是非空字符串。');
  return null;
}

function readInteger(
  value: unknown,
  path: string,
  issues: DashboardSchemaValidationIssue[],
  positive: boolean,
): number | null {
  if (typeof value === 'number' && Number.isInteger(value) && (positive ? value > 0 : value >= 0)) {
    return value;
  }

  addIssue(issues, 'invalid-layout-value', path, positive ? '必须是正整数。' : '必须是非负整数。');
  return null;
}

function readOptionalInteger(
  value: unknown,
  path: string,
  issues: DashboardSchemaValidationIssue[],
): number | undefined {
  if (value === undefined) return undefined;

  const result = readInteger(value, path, issues, true);
  return result === null ? undefined : result;
}

function isOptionalBoolean(value: unknown) {
  return value === undefined || typeof value === 'boolean';
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** schema 会直接写入 JSON，递归排除函数、NaN、循环引用等不可持久化的配置。 */
function isJsonValue(value: unknown, seen = new WeakSet<object>()): boolean {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return true;
  if (typeof value === 'number') return Number.isFinite(value);
  if (typeof value !== 'object') return false;
  if (seen.has(value)) return false;

  seen.add(value);
  const isValid = Array.isArray(value)
    ? value.every((item) => isJsonValue(item, seen))
    : (Object.getPrototypeOf(value) === Object.prototype ||
        Object.getPrototypeOf(value) === null) &&
      Object.values(value).every((item) => isJsonValue(item, seen));
  seen.delete(value);
  return isValid;
}

function addIssue(
  issues: DashboardSchemaValidationIssue[],
  code: string,
  path: string,
  message: string,
) {
  issues.push({ code, path, message });
}

function invalid(
  issues: DashboardSchemaValidationIssue[],
  code: string,
  path: string,
  message: string,
): DashboardSchemaValidationResult<never> {
  addIssue(issues, code, path, message);
  return { valid: false, issues };
}
