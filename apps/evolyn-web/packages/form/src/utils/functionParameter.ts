import type {
  PluginFunctionParameter,
  PluginFunctionRequestField,
} from '../api'
import type { PluginDesignField, PluginDesignTemplateField } from '../types'

type PluginFunctionDesignField = PluginDesignField | PluginDesignTemplateField

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

const getStringProperty = (field: PluginFunctionDesignField, key: string) => {
  const value = (field as unknown as Record<string, unknown>)[key]
  return typeof value === 'string' ? value : ''
}

/**
 * 生成函数参数字段配置，并递归转换子表单字段和下拉选项。
 * @param field 设计器请求参数字段。
 */
const createFunctionFieldConf = (field: PluginFunctionDesignField): Record<string, unknown> => {
  const fieldConf = isRecord(field.fieldConf) ? { ...field.fieldConf } : {}
  const childFields = fieldConf.fields
  if (Array.isArray(childFields)) {
    fieldConf.fields = childFields
      .filter(isRecord)
      .map((childField, index) =>
        createFunctionParameterField(childField as unknown as PluginDesignTemplateField, index),
      )
  }
  const options = (field as PluginDesignField).options
  if (Array.isArray(options)) {
    fieldConf.items = options.map((option) => ({
      text: option,
      value: option,
    }))
  }
  return fieldConf
}

/**
 * 将设计器字段转换为函数更新接口使用的新字段结构。
 * @param field 设计器请求参数字段。
 * @param index 字段在当前层级中的排序位置。
 */
function createFunctionParameterField(
  field: PluginFunctionDesignField,
  index: number,
): PluginFunctionRequestField {
  const nextField: PluginFunctionRequestField = {
    id: field.id,
    fieldKey: field.fieldKey,
    fieldLabel: field.fieldLabel,
    description: getStringProperty(field, 'placeholder') || getStringProperty(field, 'description'),
    // 函数更新接口使用 widgetName 识别控件，并携带对应数据类型。
    widgetName: getStringProperty(field, 'widgetName'),
    dataType: getStringProperty(field, 'dataType'),
    isHidden: Boolean((field as PluginDesignTemplateField).isHidden),
    isEnabled: (field as PluginDesignTemplateField).isEnabled !== false,
    isRequired: Boolean(field.isRequired),
    fieldConf: createFunctionFieldConf(field),
    sort: index,
  }
  if (field.defaultValue !== undefined && field.defaultValue !== null) {
    nextField.defaultValue = field.defaultValue
  }
  return nextField
}

/**
 * 将设计器请求字段集合转换为函数更新接口的 requestParameter。
 * @param fields 当前选中函数的请求参数字段。
 */
export const createPluginFunctionRequestParameter = (
  fields: PluginDesignField[],
): PluginFunctionParameter<PluginFunctionRequestField> => ({
  fields: fields.map(createFunctionParameterField),
})
