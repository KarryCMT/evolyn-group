<script setup lang="ts">
import { computed } from 'vue';
import type { CheckboxGroupWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId, fieldLabelId } from '../../field-dom';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/** 复选组（checkboxgroup）：值协议 string[]（空数组=未选）。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as CheckboxGroupWidget);
const key = computed(() => props.item.widget.widgetName);
const inputId = computed(() => fieldInputId(key.value));
const labelId = computed(() => fieldLabelId(key.value));
const options = computed(() => readWidgetOptions(widget.value));
const horizontal = computed(() => widget.value.layout === 'horizontal');
const modelValue = computed(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as string[]) : [],
);
const describedBy = computed(() =>
  fieldAriaDescribedBy(key.value, props.item.description !== '', props.errors.length > 0),
);

function isChecked(value: string): boolean {
  return modelValue.value.includes(value);
}

function onChange(value: string, checked: boolean): void {
  const next = checked
    ? [...modelValue.value, value]
    : modelValue.value.filter((entry) => entry !== value);
  emit('update:modelValue', next as FormValue);
}
</script>

<template>
  <div
    class="evf-choice-group"
    :class="{ 'evf-choice-group--horizontal': horizontal }"
    role="group"
    :aria-labelledby="labelId"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
  >
    <label v-for="option in options" :key="option.value" class="evf-choice">
      <input
        type="checkbox"
        :name="inputId"
        :value="option.value"
        :checked="isChecked(option.value)"
        :disabled="disabled"
        @change="onChange(option.value, ($event.target as HTMLInputElement).checked)"
        @blur="emit('blur')"
      />
      <span class="evf-choice__label">{{ option.label }}</span>
    </label>
  </div>
</template>
