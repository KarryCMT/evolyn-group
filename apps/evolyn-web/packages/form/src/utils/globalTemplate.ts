import type { PluginAuthTemplateField, PluginAuthTemplateSection } from '../api';
import type { PluginDesignField, PluginDesignTemplateField } from '../types';
import {
  createPluginSelectorDefaultValue,
  getPluginSelectorValueKey,
  isPluginSelectWidget,
  isFormSubformWidget,
  isPluginTextWidget,
} from './widgetName';

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};

const getTemplateFieldKey = (field: Partial<PluginAuthTemplateField>) => {
  // 通用参数接口与设计器内部统一使用 fieldKey。
  return typeof field.fieldKey === 'string' ? field.fieldKey : '';
};

const getTemplateFieldOptions = (field: PluginAuthTemplateField) => {
  const items = field.fieldConf?.items;
  if (!Array.isArray(items)) return undefined;
  return items.map((item) => {
    if (!isRecord(item)) return String(item ?? '');
    return String(item.text ?? item.value ?? '');
  });
};

const getTemplateFieldConf = (field: PluginDesignField): Record<string, unknown> => {
  if (isFormSubformWidget(field.widgetName)) {
    return {
      ...field.fieldConf,
      fields: normalizeSubformFields(field.fieldConf?.fields),
    };
  }
  if (isPluginSelectWidget(field.widgetName)) {
    return {
      items: (field.options || []).map((option) => ({
        text: option,
        value: option,
      })),
    };
  }
  if (isPluginTextWidget(field.widgetName)) {
    return {
      isMultiLine: false,
    };
  }
  return {};
};

const getTemplateFieldDefaultValue = (field: PluginDesignField) => {
  if (isFormSubformWidget(field.widgetName)) return undefined;
  const selectorValueKey = getPluginSelectorValueKey(field.widgetName);
  if (selectorValueKey) {
    // 成员/部门字段默认值按插件字段元数据保存为 id 数组对象，不能降级成字符串。
    return isRecord(field.defaultValue)
      ? field.defaultValue
      : createPluginSelectorDefaultValue(field.widgetName);
  }
  // 文本字段保留空字符串；其他类型没有有效默认值时不向接口提交 defaultValue。
  if (isPluginTextWidget(field.widgetName)) return field.defaultValue ?? '';
  if (field.defaultValue === '' || field.defaultValue === undefined || field.defaultValue === null)
    return undefined;
  return field.defaultValue;
};

const createSubformFieldsFromTemplate = (fields: unknown): PluginDesignTemplateField[] => {
  if (!Array.isArray(fields)) return [];
  return fields.map((field) => {
    const templateField = field as Partial<PluginAuthTemplateField>;
    const fieldKey = getTemplateFieldKey(templateField);
    const widgetName = templateField.widgetName || '';
    const fieldConf = isRecord(templateField.fieldConf) ? templateField.fieldConf : {};
    const nextField: PluginDesignTemplateField = {
      id: typeof templateField.id === 'string' ? templateField.id : null,
      fieldKey,
      fieldLabel: String(templateField.fieldLabel || fieldKey),
      description: templateField.description || '',
      widgetName,
      dataType: templateField.dataType || '',
      isHidden: Boolean(templateField.isHidden),
      isEnabled: templateField.isEnabled !== false,
      isRequired: Boolean(templateField.isRequired),
      fieldConf: isFormSubformWidget(widgetName)
        ? {
            ...fieldConf,
            fields: createSubformFieldsFromTemplate(fieldConf.fields),
          }
        : fieldConf,
    };
    if (templateField.defaultValue !== undefined && templateField.defaultValue !== null) {
      nextField.defaultValue =
        templateField.defaultValue as PluginDesignTemplateField['defaultValue'];
    }
    return nextField;
  });
};

const normalizeSubformFields = (fields: unknown): PluginAuthTemplateField[] => {
  if (!Array.isArray(fields)) return [];
  return fields.map((field) => {
    const childField = field as Partial<PluginDesignTemplateField>;
    const widgetName = childField.widgetName || '';
    const fieldConf = isRecord(childField.fieldConf) ? childField.fieldConf : {};
    const nextField: PluginAuthTemplateField = {
      id: childField.id ?? null,
      fieldKey: String(childField.fieldKey || childField.id || ''),
      fieldLabel: String(childField.fieldLabel || ''),
      description: childField.description || '',
      widgetName,
      dataType: childField.dataType || '',
      isHidden: Boolean(childField.isHidden),
      isEnabled: childField.isEnabled !== false,
      isRequired: Boolean(childField.isRequired),
      fieldConf: isFormSubformWidget(widgetName)
        ? {
            ...fieldConf,
            fields: normalizeSubformFields(fieldConf.fields),
          }
        : fieldConf,
    };
    if (childField.defaultValue !== undefined && childField.defaultValue !== null) {
      nextField.defaultValue = childField.defaultValue;
    }
    return nextField;
  });
};

/**
 * 将接口 globalTemplate 转换为通用参数画布使用的字段结构。
 * @param section 插件级通用参数模板。
 */
export const createGlobalFieldsFromTemplate = (
  section?: PluginAuthTemplateSection,
): PluginDesignField[] => {
  if (!section?.fields?.length) return [];
  return section.fields
    .filter((field) => field.isHidden !== true && field.isEnabled !== false)
    .map((field) => {
      const fieldKey = getTemplateFieldKey(field);
      const widgetName = field.widgetName;
      const fieldConf = field.fieldConf || {};
      const nextField: PluginDesignField = {
        id: typeof field.id === 'string' ? field.id : null,
        fieldKey,
        fieldLabel: field.fieldLabel || fieldKey,
        widgetName,
        dataType: field.dataType,
        placeholder: field.description || '',
        isRequired: Boolean(field.isRequired),
        options: getTemplateFieldOptions(field),
        fieldConf: isFormSubformWidget(widgetName)
          ? {
              ...fieldConf,
              fields: createSubformFieldsFromTemplate(fieldConf.fields),
            }
          : fieldConf,
      };
      if (field.defaultValue !== undefined) {
        nextField.defaultValue = field.defaultValue as PluginDesignField['defaultValue'];
      } else if (isPluginTextWidget(widgetName)) {
        nextField.defaultValue = '';
      } else if (getPluginSelectorValueKey(widgetName)) {
        nextField.defaultValue = createPluginSelectorDefaultValue(widgetName);
      }
      return nextField;
    });
};

/**
 * 将通用参数画布字段转换为接口要求的 globalTemplate。
 * @param fields 通用参数画布字段。
 */
export const createGlobalTemplateByFields = (
  fields: PluginDesignField[],
): PluginAuthTemplateSection => ({
  fields: fields.map((field) => {
    const nextField: PluginAuthTemplateField = {
      id: field.id,
      fieldKey: field.fieldKey,
      fieldLabel: field.fieldLabel,
      description: field.placeholder || '',
      widgetName: field.widgetName,
      dataType: field.dataType,
      isHidden: false,
      isEnabled: true,
      isRequired: Boolean(field.isRequired),
      fieldConf: getTemplateFieldConf(field),
    };
    const defaultValue = getTemplateFieldDefaultValue(field);
    if (defaultValue !== undefined) nextField.defaultValue = defaultValue;
    return nextField;
  }),
});
