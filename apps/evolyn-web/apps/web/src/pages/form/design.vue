<script setup lang="ts">
import {
  RiCalendarScheduleFill,
  RiCheckboxMultipleFill,
  RiEyeFill,
  RiHashtag,
  RiLightbulbFlashFill,
  RiOrganizationChart,
  RiSave3Fill,
  RiShareForwardFill,
  RiTableFill,
  RiText,
  RiUser3Fill,
} from '@remixicon/vue';
import {
  cloneFormDesign,
  FormDesignPalette,
  FormDesignPropertyPanel,
  FormFieldCanvas,
  type FormDesignDragField,
  type FormDesignField,
  type FormDesignPaletteItem,
  type FormDesignTemplateField,
  useFormDesignFactory,
  useFormFieldActions,
} from '@evolyn.do/form';
import { ElMessage } from 'element-plus';
import { computed, ref } from 'vue';

defineOptions({ name: 'FormDesignPage' });

/** 保存、预览等服务端能力尚未接入，本期保留明确的交互反馈。 */
function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在表单设计器接入后提供`);
}

// 页面直接维护新字段模型，不再通过 FormDocument 或设计器适配层转换。
const fields = ref<FormDesignField[]>([]);
const selectedFieldKey = ref('');
const selectedSubformChildFieldKey = ref('');
const { createField, createSubformChildField } = useFormDesignFactory();

const paletteFields: FormDesignPaletteItem[] = [
  { label: '单行文本', widgetName: 'input', dataType: 'String', icon: RiText },
  { label: '数字', widgetName: 'number', dataType: 'Number', icon: RiHashtag },
  { label: '日期时间', widgetName: 'datetime', dataType: 'Date', icon: RiCalendarScheduleFill },
  { label: '下拉框', widgetName: 'selectGroup', dataType: 'Object', icon: RiCheckboxMultipleFill },
  { label: '成员选择', widgetName: 'userGroup', dataType: 'Object', icon: RiUser3Fill },
  { label: '部门选择', widgetName: 'deptGroup', dataType: 'Object', icon: RiOrganizationChart },
  { label: '子表单', widgetName: 'subforms', dataType: '', icon: RiTableFill },
];

const widgetNameTextMap = computed<Record<string, string>>(() =>
  Object.fromEntries(paletteFields.map((item) => [item.widgetName, item.label])),
);
const activeFields = computed(() => fields.value);
const selectedField = computed(() =>
  fields.value.find((field) => field.fieldKey === selectedFieldKey.value),
);
const selectedWidgetNameLabel = computed(() => {
  const field = selectedField.value;
  return field ? widgetNameTextMap.value[field.widgetName] || field.widgetName : '';
});

const {
  addDragField: addRootDragField,
  addField: addRootField,
  addOption,
  copyField: copyRootField,
  removeField: removeRootField,
  removeOption,
  selectField: selectRootField,
  updateFieldDefaultValue,
  updateOption,
  updateSelectedFieldDefaultValue,
} = useFormFieldActions({
  activeFields,
  createField,
  selectedField,
  selectedFieldKey,
  widgetNameTextMap,
});

const addField = (source: FormDesignPaletteItem) => {
  selectedSubformChildFieldKey.value = '';
  addRootField(source);
};

const addDragField = (value: FormDesignDragField) => {
  selectedSubformChildFieldKey.value = '';
  addRootDragField(value);
};

const copyField = (field: FormDesignField) => {
  selectedSubformChildFieldKey.value = '';
  copyRootField(field);
};

const removeField = (fieldKey: string) => {
  selectedSubformChildFieldKey.value = '';
  removeRootField(fieldKey);
};

const selectField = (fieldKey: string) => {
  selectedSubformChildFieldKey.value = '';
  selectRootField(fieldKey);
};

const updateSelectedFieldKey = (fieldKey: string) => {
  if (!selectedField.value) return;
  selectedField.value.fieldKey = fieldKey;
  selectedFieldKey.value = fieldKey;
};

const getSubformChildFields = (field?: FormDesignField, ensureFields = false) => {
  if (ensureFields && field) {
    field.fieldConf ??= {};
    if (!Array.isArray(field.fieldConf.fields)) field.fieldConf.fields = [];
  }
  const childFields = field?.fieldConf?.fields;
  return Array.isArray(childFields) ? (childFields as FormDesignTemplateField[]) : [];
};

let subformChildFieldSeed = 0;
const createSubformChildFieldKey = () => `_widget_${Date.now()}${subformChildFieldSeed++}`;

const selectSubformChildField = (value: { parentKey: string; childKey: string }) => {
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value = value.childKey;
};

const copySubformChildField = (value: { parentKey: string; childKey: string }) => {
  const childFields = getSubformChildFields(
    fields.value.find((field) => field.fieldKey === value.parentKey),
  );
  const index = childFields.findIndex((field) => field.fieldKey === value.childKey);
  if (index === -1) return;
  const nextField = {
    ...cloneFormDesign(childFields[index]),
    id: null,
    fieldKey: createSubformChildFieldKey(),
    fieldLabel: `${childFields[index].fieldLabel} copy`,
  };
  childFields.splice(index + 1, 0, nextField);
  selectSubformChildField({ parentKey: value.parentKey, childKey: nextField.fieldKey });
};

const removeSubformChildField = (value: { parentKey: string; childKey: string }) => {
  const childFields = getSubformChildFields(
    fields.value.find((field) => field.fieldKey === value.parentKey),
  );
  const index = childFields.findIndex((field) => field.fieldKey === value.childKey);
  if (index === -1) return;
  childFields.splice(index, 1);
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value =
    childFields[index]?.fieldKey || childFields[index - 1]?.fieldKey || '';
};

const addSubformDragField = (value: FormDesignDragField & { parentKey: string }) => {
  const childFields = getSubformChildFields(
    fields.value.find((field) => field.fieldKey === value.parentKey),
    true,
  );
  const field = createSubformChildField(value.widgetName, value.dataType);
  if (!field) return;
  childFields.splice(value.index >= 0 ? value.index : childFields.length, 0, field);
  selectSubformChildField({ parentKey: value.parentKey, childKey: field.fieldKey });
};
</script>

<template>
  <section class="form-design-page" aria-label="表单设计工作台">
    <div class="form-design-page__toolbar" aria-label="表单设计操作">
      <button
        class="form-design-page__guide-button"
        type="button"
        @click="notifyUnavailable('新手引导')"
      >
        <RiLightbulbFlashFill />
        <span class="form-design-page__guide-label">查看新手引导</span>
      </button>
      <div class="form-design-page__toolbar-actions">
        <button
          class="form-design-page__action-button form-design-page__action-button--secondary"
          type="button"
          @click="notifyUnavailable('预览')"
        >
          <RiEyeFill />
          <span class="form-design-page__action-label">预览</span>
        </button>
        <button
          class="form-design-page__action-button form-design-page__action-button--primary"
          type="button"
          @click="notifyUnavailable('保存')"
        >
          <RiSave3Fill />
          <span class="form-design-page__action-label">保存</span>
        </button>
        <button
          class="form-design-page__icon-button form-design-page__share-button"
          type="button"
          aria-label="分享表单"
          @click="notifyUnavailable('分享')"
        >
          <RiShareForwardFill />
        </button>
      </div>
    </div>

    <div class="form-design-page__workspace">
      <FormDesignPalette :fields="paletteFields" @add-field="addField" />
      <FormFieldCanvas
        :fields="fields"
        :selected-field-key="selectedFieldKey"
        :selected-subform-child-field-key="selectedSubformChildFieldKey"
        @select-field="selectField"
        @select-subform-child="selectSubformChildField"
        @copy-field="copyField"
        @copy-subform-child="copySubformChildField"
        @remove-field="removeField"
        @remove-subform-child="removeSubformChildField"
        @add-drag-field="addDragField"
        @add-subform-drag-field="addSubformDragField"
        @update-field-default-value="updateFieldDefaultValue"
      />
      <FormDesignPropertyPanel
        :field="selectedField"
        :widget-name-label="selectedWidgetNameLabel"
        :selected-subform-child-field-key="selectedSubformChildFieldKey"
        @add-option="addOption"
        @remove-option="removeOption"
        @update-default-value="updateSelectedFieldDefaultValue"
        @update-field-key="updateSelectedFieldKey"
        @update-option="updateOption"
        @update-subform-child-selection="selectedSubformChildFieldKey = $event"
      />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-design-page {
  display: flex;
  min-height: 0;
  margin: 0 var(--el-space-md) var(--el-space-md);
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__toolbar,
  &__toolbar-actions,
  &__guide-button,
  &__action-button {
    display: flex;
    align-items: center;
  }

  &__toolbar {
    height: 50px;
    min-height: 50px;
    padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
    justify-content: space-between;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__guide-button,
  &__action-button,
  &__icon-button {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__guide-button,
  &__action-button {
    justify-content: center;
    gap: var(--el-space-sm);
    font-size: var(--el-font-size-base);
    font-weight: 600;
  }

  &__guide-button {
    height: 32px;
    padding: 0 var(--el-space-md);
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
      color: var(--el-color-primary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__toolbar-actions {
    gap: var(--el-space-md);
  }

  &__action-button {
    min-width: 76px;
    height: 32px;
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-base);

    svg {
      width: 17px;
      height: 17px;
    }

    &--secondary {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      border: 1px solid var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-9);
      }
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-3);
      }
    }
  }

  &__icon-button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 20px;
      height: 20px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__share-button {
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__workspace {
    display: flex;
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }

  &__canvas-placeholder {
    width: 100%;
    min-height: 100%;
  }
}

@media (max-width: 620px) {
  .form-design-page {
    margin: 0 var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-large);

    &__toolbar {
      padding: 0 var(--el-space-md) 0 var(--el-space-lg);
    }

    &__guide-button {
      padding: 0 var(--el-space-xs);
    }

    &__guide-label,
    &__action-label {
      display: none;
    }

    &__toolbar-actions {
      gap: var(--el-space-sm);
    }

    &__action-button {
      min-width: 34px;
      padding: 0 var(--el-space-md);
    }
  }
}
</style>
