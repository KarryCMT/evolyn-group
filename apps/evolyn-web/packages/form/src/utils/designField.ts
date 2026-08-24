import type {
  PluginDesignAuthentication,
  FormDesignField,
  PluginDesignResponseField,
  FormDesignTemplateField,
  PluginDesignTemplateSection,
} from '../types';

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};

const getStringValue = (value: unknown) => (typeof value === 'string' ? value : '');

const getBooleanValue = (value: unknown, legacyValue: unknown) => {
  if (typeof value === 'boolean') return value;
  return typeof legacyValue === 'boolean' ? legacyValue : false;
};

/**
 * 将子表单字段统一为 FormDesignTemplateField。
 * @param value 待归一化的模板字段数据。
 * @param fallbackKey 字段缺少标识时使用的稳定兜底值。
 */
export const normalizeFormDesignTemplateField = (
  value: unknown,
  fallbackKey: string,
): FormDesignTemplateField => {
  const source = isRecord(value) ? value : {};
  const fieldKey = getStringValue(source.fieldKey) || getStringValue(source.id) || fallbackKey;
  const rest = { ...source };
  delete rest.type;
  delete rest.fieldType;
  delete rest.required;
  delete rest.label;

  return {
    ...rest,
    id: getStringValue(source.id) || null,
    fieldKey,
    fieldLabel: getStringValue(source.fieldLabel) || getStringValue(source.label) || fieldKey,
    widgetName: getStringValue(source.widgetName),
    dataType: getStringValue(source.dataType),
    isRequired: getBooleanValue(source.isRequired, source.required),
    fieldConf: isRecord(source.fieldConf) ? { ...source.fieldConf } : {},
  };
};

/**
 * 将设计器字段归一为使用 widgetName 的当前字段结构。
 * @param value 待归一化的字段数据。
 * @param fallbackKey 字段缺少标识时使用的稳定兜底值。
 */
export const normalizeFormDesignField = (value: unknown, fallbackKey: string): FormDesignField => {
  const source = isRecord(value) ? value : {};
  const fieldKey = getStringValue(source.fieldKey) || getStringValue(source.id) || fallbackKey;
  const fieldConf = isRecord(source.fieldConf) ? { ...source.fieldConf } : undefined;
  const childFields = fieldConf?.fields;
  // 旧字段名只用于读取兼容，归一化结果不再携带它们，避免保存时重新提交旧结构。
  const rest = { ...source };
  delete rest.type;
  delete rest.fieldType;
  delete rest.required;
  delete rest.label;

  if (fieldConf && Array.isArray(childFields)) {
    fieldConf.fields = childFields.map((field, index) =>
      normalizeFormDesignTemplateField(field, `${fieldKey}_child_${index}`),
    );
  }

  return {
    ...rest,
    id: getStringValue(source.id) || null,
    fieldKey,
    fieldLabel: getStringValue(source.fieldLabel) || getStringValue(source.label) || fieldKey,
    widgetName: getStringValue(source.widgetName),
    dataType: getStringValue(source.dataType),
    isRequired: getBooleanValue(source.isRequired, source.required),
    fieldConf,
  } as FormDesignField;
};

/**
 * 归一化字段集合，并通过作用域生成可重复的兜底标识。
 * @param fields 待归一化的字段集合。
 * @param scope 字段所属函数或模块标识。
 */
export const normalizeFormDesignFields = (fields: unknown, scope: string): FormDesignField[] => {
  if (!Array.isArray(fields)) return [];
  return fields.map((field, index) => normalizeFormDesignField(field, `${scope}_field_${index}`));
};

/** @deprecated 旧插件模块迁移期间的兼容别名，新表单代码使用 Form 命名。 */
export const normalizePluginDesignTemplateField = normalizeFormDesignTemplateField;
/** @deprecated */
export const normalizePluginDesignField = normalizeFormDesignField;
/** @deprecated */
export const normalizePluginDesignFields = normalizeFormDesignFields;

/**
 * 将代码视图中的返回参数递归归一为设计器结构，并保留 fieldConf 内的扩展配置。
 * @param value 待归一化的返回参数字段。
 * @param fallbackKey 字段缺少标识时使用的稳定兜底值。
 */
export const normalizePluginDesignResponseField = (
  value: unknown,
  fallbackKey: string,
): PluginDesignResponseField => {
  const source = isRecord(value) ? value : {};
  const fieldKey = getStringValue(source.fieldKey) || fallbackKey;
  const fieldConf = isRecord(source.fieldConf) ? { ...source.fieldConf } : {};
  if (Array.isArray(fieldConf.fields)) {
    fieldConf.fields = fieldConf.fields.map((field, index) =>
      normalizePluginDesignResponseField(field, `${fieldKey}_child_${index}`),
    );
  }
  return {
    id: source.id == null ? null : String(source.id),
    fieldKey,
    fieldLabel: getStringValue(source.fieldLabel) || fieldKey,
    widgetName: getStringValue(source.widgetName),
    dataType: getStringValue(source.dataType),
    fieldConf,
  };
};

/**
 * 归一化代码视图提交的返回参数集合。
 * @param fields 待归一化的返回参数字段集合。
 * @param scope 字段所属函数标识。
 */
export const normalizePluginDesignResponseFields = (
  fields: unknown,
  scope: string,
): PluginDesignResponseField[] => {
  if (!Array.isArray(fields)) return [];
  return fields.map((field, index) =>
    normalizePluginDesignResponseField(field, `${scope}_response_${index}`),
  );
};

const normalizeTemplateSection = (
  value: unknown,
  scope: string,
): PluginDesignTemplateSection | undefined => {
  if (!isRecord(value)) return undefined;
  const fields = Array.isArray(value.fields) ? value.fields : [];
  return {
    ...value,
    fields: fields.map((field, index) =>
      normalizeFormDesignTemplateField(field, `${scope}_field_${index}`),
    ),
  };
};

/**
 * 归一化身份验证内的全部模板分区，兼容历史字段别名。
 * @param value 身份验证配置。
 */
export const normalizePluginDesignAuthentication = (value: unknown): PluginDesignAuthentication => {
  const source = isRecord(value) ? value : {};
  const result = {
    ...source,
    type: getStringValue(source.type) || 'none',
    conf_template: normalizeTemplateSection(source.conf_template, 'conf_template') || {
      fields: [],
    },
  } as PluginDesignAuthentication;
  const optionalSectionKeys = [
    'credential_template',
    'oauth2_request_template',
    'refresh_token_request_template',
    'app_auth_template',
  ] as const;

  optionalSectionKeys.forEach((key) => {
    const section = normalizeTemplateSection(source[key], key);
    if (section) result[key] = section;
  });
  return result;
};
