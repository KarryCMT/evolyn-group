import { type ComputedRef, type Ref } from 'vue';
import type {
  FormDesignDragField,
  FormDesignField,
  FormDesignFieldDefaultValue,
  FormDesignPaletteItem,
} from '../types';
import { cloneFormDesign } from './useFormDesignFactory';

interface UseFormFieldActionsOptions {
  activeFields: ComputedRef<FormDesignField[]>;
  createField: (
    fieldLabel: string,
    widgetName: string,
    dataType: string,
    fieldKey?: string,
  ) => FormDesignField;
  widgetNameTextMap: ComputedRef<Record<string, string>>;
  selectedField: ComputedRef<FormDesignField | undefined>;
  selectedFieldKey: Ref<string>;
}

export const useFormFieldActions = ({
  activeFields,
  createField,
  widgetNameTextMap,
  selectedField,
  selectedFieldKey,
}: UseFormFieldActionsOptions) => {
  const addField = (source: FormDesignPaletteItem) => {
    const label = source.label || widgetNameTextMap.value[source.widgetName] || source.widgetName;
    const field = createField(label, source.widgetName, source.dataType, undefined);
    activeFields.value.push(field);
    selectedFieldKey.value = field.fieldKey;
  };

  const addDragField = ({ index, widgetName, dataType }: FormDesignDragField) => {
    const label = widgetNameTextMap.value[widgetName] || widgetName;
    const field = createField(label, widgetName, dataType);
    if (index < 0) {
      activeFields.value.push(field);
    } else {
      activeFields.value.splice(index, 1, field);
    }
    selectedFieldKey.value = field.fieldKey;
  };

  const selectField = (fieldKey: string) => {
    selectedFieldKey.value = fieldKey;
  };

  const copyField = (field: FormDesignField) => {
    const nextFieldKey = `${field.fieldKey}_${Date.now()}`;
    const nextField = {
      ...cloneFormDesign(field),
      // 复制后作为新增字段提交，后端保存后再返回持久化 id。
      id: null,
      fieldKey: nextFieldKey,
      fieldLabel: `${field.fieldLabel} copy`,
      options: field.options ? [...field.options] : undefined,
    } as FormDesignField;
    const index = activeFields.value.findIndex((item) => item.fieldKey === field.fieldKey);
    activeFields.value.splice(index + 1, 0, nextField);
    selectedFieldKey.value = nextField.fieldKey;
  };

  const removeField = (fieldKey: string) => {
    const index = activeFields.value.findIndex((item) => item.fieldKey === fieldKey);
    if (index === -1) return;
    activeFields.value.splice(index, 1);
    if (selectedFieldKey.value === fieldKey) selectedFieldKey.value = '';
  };

  const addOption = () => {
    if (!selectedField.value) return;
    if (!selectedField.value.options) selectedField.value.options = [];
    selectedField.value.options.push(`${'选项'}${selectedField.value.options.length + 1}`);
  };

  const removeOption = (index: number) => {
    const field = selectedField.value;
    if (!field?.options) return;
    const removedOption = field.options[index];
    field.options.splice(index, 1);
    // 删除当前默认项后同步清空默认值，避免保存不存在的下拉选项。
    if (field.defaultValue === removedOption) field.defaultValue = '';
  };

  const updateOption = (index: number, value: string) => {
    const field = selectedField.value;
    if (!field?.options) return;
    const previousOption = field.options[index];
    field.options[index] = value;
    // 默认项改名时保持选中关系，并将新值提交到 defaultValue。
    if (field.defaultValue === previousOption) field.defaultValue = value;
  };

  const updateFieldDefaultValue = (fieldKey: string, value: FormDesignFieldDefaultValue) => {
    const field = activeFields.value.find((item) => item.fieldKey === fieldKey);
    if (!field) return;
    field.defaultValue = value;
  };

  const updateSelectedFieldDefaultValue = (value: FormDesignFieldDefaultValue) => {
    if (!selectedField.value) return;
    selectedField.value.defaultValue = value;
  };

  return {
    addDragField,
    addField,
    addOption,
    copyField,
    removeField,
    removeOption,
    selectField,
    updateFieldDefaultValue,
    updateOption,
    updateSelectedFieldDefaultValue,
  };
};

/** @deprecated 旧插件设计器尚未拆除时的兼容导出。 */
export const usePluginFieldActions = useFormFieldActions;
