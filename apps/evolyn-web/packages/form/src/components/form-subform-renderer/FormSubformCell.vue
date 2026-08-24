<template>
  <FormDesignFieldControl
    v-if="column.mode === 'custom'"
    :field="childField"
    :model-value="customFieldValue"
    :is-smart-assistant="isSmartAssistant"
    :selector-options="currentSelectorOptions"
    @update:model-value="updateCustomValue"
  />
  <slot
    v-else
    name="depend-field"
    :cell="dependCell"
    :child-field="childField"
    :column="column"
    :row="row"
    :row-index="rowIndex"
    :update-cell="updateDependValue"
  >
    <el-input :model-value="dependCell.beforeDependField" :placeholder="t('请选择字段')" readonly />
  </slot>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type {
  FormDesignFieldDefaultValue,
  FormDesignSelectorOptionsResolver,
  FormDesignTemplateField,
} from '../../types';
import FormDesignFieldControl from '../FormDesignFieldControl.vue';
import type {
  FormSubformCellBinding,
  FormSubformColumnBinding,
  FormSubformDependCellBinding,
  FormSubformRowBinding,
} from './types';

/**
 * 子表单单元格，根据列级赋值模式渲染自定义控件或业务侧字段选择器。
 * @property childField 子字段定义。
 * @property column 当前列绑定配置。
 * @property row 当前行绑定配置。
 * @property rowIndex 当前行下标。
 * @property modelValue 当前单元格绑定值。
 * @property isSmartAssistant 是否使用智能助手场景的人员/部门下拉选择器。
 * @property selectorOptions 根据子字段获取智能助手人员/部门下拉选项的方法。
 */
const props = defineProps<{
  childField: FormDesignTemplateField;
  column: FormSubformColumnBinding;
  row: FormSubformRowBinding;
  rowIndex: number;
  modelValue: FormSubformCellBinding;
  isSmartAssistant?: boolean;
  selectorOptions?: FormDesignSelectorOptionsResolver;
}>();

const emits = defineEmits<{
  (event: 'update:modelValue', value: FormSubformCellBinding): void;
}>();

const t = (text: string) => text;

// 子表单每一列可能是人员或部门字段，需要按当前子字段动态取得对应选项。
const currentSelectorOptions = computed(() => props.selectorOptions?.(props.childField) || []);

const customFieldValue = computed<FormDesignFieldDefaultValue | undefined>(() => {
  if (!('sourceData' in props.modelValue)) return undefined;
  return props.modelValue.sourceData ?? undefined;
});

const dependCell = computed<FormSubformDependCellBinding>(() => {
  if ('dependField' in props.modelValue) return props.modelValue;
  return {
    linkNodeId: '',
    dependField: '',
    beforeDependField: '',
    dependParentKey: null,
  };
});

/** 更新自定义值，保持单元格配置只包含当前模式的数据。 */
const updateCustomValue = (sourceData: FormDesignFieldDefaultValue) => {
  emits('update:modelValue', { sourceData });
};

/** 更新字段引用，具体字段树解析由使用方通过插槽完成。 */
const updateDependValue = (value: FormSubformDependCellBinding) => {
  emits('update:modelValue', value);
};
</script>
