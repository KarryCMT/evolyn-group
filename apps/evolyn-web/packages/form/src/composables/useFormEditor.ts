import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue';
import {
  cloneFormDocument,
  getFormFieldPreset,
  type FormDocument,
  type FormField,
  type FormFieldPreset,
  type FormFieldType,
  type FormJsonValue,
} from '../schema';

export interface UseFormEditorOptions {
  initialDocument: FormDocument;
  /** 应用侧决定 ID 策略，以便与后端 UUID 或离线草稿策略保持一致。 */
  createFieldId: () => string;
  createFieldKey?: (field: Pick<FormField, 'id' | 'type'>) => string;
}

export interface FormEditor {
  document: ShallowRef<FormDocument>;
  selectedFieldIds: Readonly<ShallowRef<string[]>>;
  selectedFields: ComputedRef<FormField[]>;
  addField: (type: FormFieldType) => FormField | null;
  removeFields: (ids: string[]) => void;
  selectFields: (ids: string[]) => void;
  clearSelection: () => void;
  updateField: (
    id: string,
    patch: Partial<Pick<FormField, 'label' | 'required' | 'config'>>,
  ) => void;
  moveField: (id: string, targetIndex: number) => void;
  replaceDocument: (document: FormDocument) => void;
}

/**
 * 编辑器只负责可持久化文档和选择状态。拖拽库、接口调用、路由及权限均由外部接入层负责。
 */
export function useFormEditor(options: UseFormEditorOptions): FormEditor {
  const document = shallowRef(cloneFormDocument(options.initialDocument));
  const selectedFieldIds = shallowRef<string[]>([]);
  const selectedFields = computed(() => {
    const selected = new Set(selectedFieldIds.value);
    return document.value.fields.filter((field) => selected.has(field.id));
  });

  function addField(type: FormFieldType) {
    const preset = getFormFieldPreset(type);
    if (!preset) return null;

    const id = options.createFieldId();
    const field: FormField = {
      id,
      key: options.createFieldKey?.({ id, type }) ?? `field_${id}`,
      type,
      label: preset.defaultLabel,
      required: false,
      config: cloneConfig(preset),
    };
    document.value = { ...document.value, fields: [...document.value.fields, field] };
    selectedFieldIds.value = [id];
    return field;
  }

  function removeFields(ids: string[]) {
    if (!ids.length) return;

    const targetIds = new Set(ids);
    document.value = {
      ...document.value,
      fields: document.value.fields.filter((field) => !targetIds.has(field.id)),
    };
    selectedFieldIds.value = selectedFieldIds.value.filter((id) => !targetIds.has(id));
  }

  function selectFields(ids: string[]) {
    const knownIds = new Set(document.value.fields.map((field) => field.id));
    selectedFieldIds.value = [...new Set(ids)].filter((id) => knownIds.has(id));
  }

  function clearSelection() {
    selectedFieldIds.value = [];
  }

  function updateField(
    id: string,
    patch: Partial<Pick<FormField, 'label' | 'required' | 'config'>>,
  ) {
    document.value = {
      ...document.value,
      fields: document.value.fields.map((field) => {
        if (field.id !== id) return field;

        return {
          ...field,
          ...patch,
          ...(Object.prototype.hasOwnProperty.call(patch, 'config')
            ? { config: { ...patch.config } }
            : {}),
        };
      }),
    };
  }

  function moveField(id: string, targetIndex: number) {
    const sourceIndex = document.value.fields.findIndex((field) => field.id === id);
    if (sourceIndex < 0) return;

    const fields = [...document.value.fields];
    const [field] = fields.splice(sourceIndex, 1);
    if (!field) return;
    fields.splice(Math.min(Math.max(targetIndex, 0), fields.length), 0, field);
    document.value = { ...document.value, fields };
  }

  function replaceDocument(value: FormDocument) {
    document.value = cloneFormDocument(value);
    clearSelection();
  }

  return {
    document,
    selectedFieldIds,
    selectedFields,
    addField,
    removeFields,
    selectFields,
    clearSelection,
    updateField,
    moveField,
    replaceDocument,
  };
}

function cloneConfig(preset: FormFieldPreset): Record<string, FormJsonValue> {
  return preset.defaultConfig ? JSON.parse(JSON.stringify(preset.defaultConfig)) : {};
}
