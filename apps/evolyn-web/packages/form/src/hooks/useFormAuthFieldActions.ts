import { type ComputedRef, type Ref } from 'vue';
import type { PluginDesignTemplateField } from '../types';
import { cloneFormDesign } from './useFormDesignFactory';

interface UseFormAuthFieldActionsOptions {
  authFields: ComputedRef<PluginDesignTemplateField[]>;
  selectedAuthField: ComputedRef<PluginDesignTemplateField | undefined>;
  selectedAuthFieldKey: Ref<string>;
}

export const useFormAuthFieldActions = ({
  authFields,
  selectedAuthField,
  selectedAuthFieldKey,
}: UseFormAuthFieldActionsOptions) => {
  const createAuthTextField = (): PluginDesignTemplateField => {
    const fieldKey = `_widget_${Date.now()}`;
    return {
      id: null,
      fieldKey,
      fieldLabel: '文本',
      description: '',
      widgetName: 'input',
      dataType: 'String',
      isHidden: false,
      isEnabled: true,
      isRequired: false,
      fieldConf: {
        isMultiLine: false,
      },
      defaultValue: '',
    };
  };

  const addAuthParam = () => {
    const field = createAuthTextField();
    authFields.value.push(field);
    selectedAuthFieldKey.value = field.fieldKey;
  };

  const selectAuthField = (fieldKey: string) => {
    selectedAuthFieldKey.value = fieldKey;
  };

  const clearAuthFieldSelection = () => {
    selectedAuthFieldKey.value = '';
  };

  const copyAuthField = (field: PluginDesignTemplateField) => {
    // 认证字段保持后端模板结构，复制时仅替换标识和显示名称，避免丢失 fieldConf 等提交字段。
    const fieldKey = `${field.fieldKey}_${Date.now()}`;
    const nextField = {
      ...cloneFormDesign(field),
      // 复制字段按新增数据处理，持久化 id 等待后端返回。
      id: null,
      fieldKey,
      fieldLabel: `${field.fieldLabel} copy`,
    };
    const index = authFields.value.findIndex((item) => item.fieldKey === field.fieldKey);
    authFields.value.splice(index + 1, 0, nextField);
    selectedAuthFieldKey.value = nextField.fieldKey;
  };

  const removeAuthField = (fieldKey: string) => {
    const index = authFields.value.findIndex((item) => item.fieldKey === fieldKey);
    if (index === -1) return;
    authFields.value.splice(index, 1);
    if (selectedAuthFieldKey.value === fieldKey) selectedAuthFieldKey.value = '';
  };

  const updateSelectedAuthFieldKey = (fieldKey: string) => {
    if (!selectedAuthField.value) return;
    // fieldKey 是可编辑业务标识；持久化 id 只由后端维护。
    selectedAuthField.value.fieldKey = fieldKey;
    selectedAuthFieldKey.value = fieldKey;
  };

  return {
    addAuthParam,
    clearAuthFieldSelection,
    copyAuthField,
    removeAuthField,
    selectAuthField,
    updateSelectedAuthFieldKey,
  };
};
