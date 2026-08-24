<template>
  <el-select-v2
    v-if="isSmartAssistant && selectorTabs.length"
    :model-value="smartAssistantSelectorModelValue"
    :options="normalizedSelectorOptions"
    :placeholder="fieldPlaceholder || t('请选择')"
    :multiple="false"
    :disabled="disabled"
    class="form-design-field-control__select"
    filterable
    collapse-tags
    teleported
    clearable
    @update:model-value="handleSmartAssistantSelectorValueChange"
  />
  <el-select-v2
    v-else-if="selectorTabs.length"
    :model-value="selectorModelValue"
    :options="normalizedSelectorOptions"
    :placeholder="fieldPlaceholder || t('请选择')"
    :multiple="false"
    :disabled="disabled"
    class="form-design-field-control__select"
    filterable
    collapse-tags
    teleported
    clearable
    @update:model-value="handleSelectorValueChange"
    @change="handleSelectorValueChange"
  />
  <el-select
    v-else-if="widgetName === 'selectGroup'"
    :model-value="stringModelValue"
    class="form-design-field-control__select"
    :placeholder="fieldPlaceholder || t('请选择')"
    :readonly="disabled"
    @update:model-value="handleStringValueChange"
  >
    <el-option
      v-for="option in selectOptions"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </el-select>
  <el-date-picker
    v-else-if="isDateTimeField"
    :model-value="stringModelValue"
    :type="datePickerType"
    :placeholder="fieldPlaceholder || t('请选择')"
    :value-format="dateValueFormat"
    style="width: 100%"
    :readonly="disabled"
    @update:model-value="handleStringValueChange"
  />
  <el-input-number
    v-else-if="widgetName === 'number'"
    :model-value="numberModelValue"
    :placeholder="fieldPlaceholder"
    controls-position="right"
    class="form-design-field-control__number"
    :readonly="disabled"
    @update:model-value="handleNumberValueChange"
  />
  <el-input
    v-else
    :model-value="stringModelValue"
    :type="isMultiLineText ? 'textarea' : 'text'"
    :placeholder="fieldPlaceholder"
    :readonly="disabled"
    @update:model-value="handleStringValueChange"
  />
</template>

<script setup lang="ts">
import { ElDatePicker, ElInput, ElInputNumber, ElOption, ElSelect, ElSelectV2 } from 'element-plus';
import { computed } from 'vue';
import type {
  FormDesignField,
  FormDesignFieldDefaultValue,
  FormDesignSelectorOption,
  FormDesignTemplateField,
} from '../types';
import { getFormSelectorValueKey } from '../utils/widgetName';

/**
 * 插件设计字段通用控件，同时支持外层字段和认证、OAuth、子表单模板字段。
 * @property field 字段配置。
 * @property modelValue 字段默认值。
 * @property disabled 是否作为只读画布预览。
 * @property isSmartAssistant 是否使用智能助手场景的人员/部门下拉选择器。
 * @property selectorOptions 智能助手人员/部门下拉选项。
 */
const props = withDefaults(
  defineProps<{
    field: FormDesignField | FormDesignTemplateField;
    modelValue?: FormDesignField['defaultValue'] | FormDesignTemplateField['defaultValue'];
    disabled?: boolean;
    isSmartAssistant?: boolean;
    selectorOptions?: FormDesignSelectorOption[];
  }>(),
  {
    disabled: false,
    isSmartAssistant: false,
    selectorOptions: () => [],
  },
);

const emits = defineEmits<{
  (event: 'update:modelValue', value: FormDesignFieldDefaultValue): void;
  (event: 'change', value: FormDesignFieldDefaultValue): void;
}>();

const t = (text: string) => text;
const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};
const widgetName = computed(() => props.field.widgetName);
const fieldPlaceholder = computed(() => {
  if ('placeholder' in props.field && typeof props.field.placeholder === 'string')
    return props.field.placeholder;
  if ('description' in props.field && typeof props.field.description === 'string')
    return props.field.description;
  return '';
});
const isMultiLineText = computed(
  () => widgetName.value === 'input' && Boolean(props.field.fieldConf?.isMultiLine),
);
// 外层字段使用字符串 options，模板字段使用 fieldConf.items，在组件内部统一成下拉选项。
const selectOptions = computed(() => {
  if ('options' in props.field && Array.isArray(props.field.options)) {
    return props.field.options.map((option) => ({ label: option, value: option }));
  }
  const items = props.field.fieldConf?.items;
  if (!Array.isArray(items)) return [];
  return items.map((item) => {
    if (!isRecord(item)) {
      const value = String(item ?? '');
      return { label: value, value };
    }
    const value = String(item.value ?? item.text ?? '');
    return {
      label: String(item.text ?? item.value ?? value),
      value,
    };
  });
});
const selectorValueKey = computed(() => getFormSelectorValueKey(widgetName.value));
const selectorTabs = computed<string[]>(() =>
  selectorValueKey.value ? [selectorValueKey.value] : [],
);
// 智能助手下拉框统一使用字符串 id，保证选项值与插件默认值的回显类型一致。
const normalizedSelectorOptions = computed(() =>
  props.selectorOptions.map((option) => ({
    ...option,
    value: String(option.value),
  })),
);
const stringModelValue = computed(() =>
  typeof props.modelValue === 'string' ? props.modelValue : '',
);
const numberModelValue = computed(() =>
  typeof props.modelValue === 'number' ? props.modelValue : undefined,
);
// Element Plus 日期组件自带固定宽度，模板在根节点内联覆盖后与其他控件一致铺满容器。
const isDateTimeField = computed(() => widgetName.value === 'datetime');
const datePickerType = computed(() => 'datetime');
const dateValueFormat = computed(() => 'YYYY-MM-DD HH:mm:ss');

const getSelectorId = (value: unknown) => {
  if (typeof value === 'string') return value;
  if (typeof value === 'number') return String(value);
  if (!isRecord(value)) return '';
  return String(value.id ?? value._id ?? '');
};

const getSelectorIds = (value: unknown, key: 'users' | 'departs') => {
  if (!isRecord(value) || !Array.isArray(value[key])) return [];
  return value[key].map(getSelectorId).filter(Boolean);
};

const selectorModelValue = computed(() => {
  const key = selectorValueKey.value;
  if (!key) return {};
  return {
    quickCheck: [],
    users: [],
    departs: [],
    posts: [],
    deptGroup: [],
    userGroup: [],
    externalDept: [],
    units: [],
    externalUnits: [],
    employments: [],
    [key]: getSelectorIds(props.modelValue, key),
  };
});

// 智能助手使用单选下拉，但对外仍保持插件字段既有的 users/departs 数组结构。
const smartAssistantSelectorModelValue = computed(() => {
  const key = selectorValueKey.value;
  if (!key) return '';
  return getSelectorIds(props.modelValue, key)[0] || '';
});

/** 统一提交人员/部门选择器的对象数组协议，避免不同渲染模式影响上层保存逻辑。 */
const emitSelectorValue = (key: 'users' | 'departs', ids: string[]) => {
  const value = { [key]: ids };
  emits('update:modelValue', value);
  emits('change', value);
};

const handleSelectorValueChange = (value: unknown) => {
  if (props.disabled) return;
  const key = selectorValueKey.value;
  if (!key) return;
  // 组件 change 可能给完整对象，v-model 更新通常给 id 聚合；统一压成插件模板需要的 id 数组。
  emitSelectorValue(key, getSelectorIds(value, key));
};

/** 将智能助手下拉框的单值转换为 FormDesignFieldControl 统一的选择器值结构。 */
const handleSmartAssistantSelectorValueChange = (value: unknown) => {
  if (props.disabled) return;
  const key = selectorValueKey.value;
  if (!key) return;
  const id = getSelectorId(value);
  emitSelectorValue(key, id ? [id] : []);
};

const handleStringValueChange = (value: string | number | null) => {
  if (props.disabled) return;
  emits('update:modelValue', String(value ?? ''));
  emits('change', String(value ?? ''));
};

const handleNumberValueChange = (value: number | undefined) => {
  if (props.disabled) return;
  emits('update:modelValue', value ?? '');
  emits('change', value ?? '');
};
</script>

<style lang="scss">
.form-design-field-control {
  &__selector,
  &__select,
  &__number {
    width: 100%;
  }
}
</style>
