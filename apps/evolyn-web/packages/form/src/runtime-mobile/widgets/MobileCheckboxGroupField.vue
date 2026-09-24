<script setup lang="ts">
import { Checkbox as VanCheckbox, CheckboxGroup as VanCheckboxGroup } from 'vant';
import { computed } from 'vue';
import type { CheckboxGroupWidget } from '../../schema/types';
import { readWidgetOptions } from '../../schema/codec';
import { fieldAriaDescribedBy, fieldLabelId } from '../../runtime/field-dom';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as CheckboxGroupWidget);
const options = computed(() => readWidgetOptions(widget.value));
const modelValue = computed(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as string[]) : [],
);
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    widget.value.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function updateValue(value: Array<string | number>): void {
  emit('update:modelValue', value.map(String) as FormValue);
}
</script>

<template>
  <VanCheckboxGroup
    class="evf-mobile-choice-group"
    :class="{ 'evf-mobile-choice-group--horizontal': widget.layout !== 'vertical' }"
    :model-value="modelValue"
    :disabled="disabled || readonly"
    :aria-labelledby="fieldLabelId(widget.widgetName)"
    :aria-required="!widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @update:model-value="updateValue"
  >
    <VanCheckbox v-for="option in options" :key="option.value" :name="option.value">
      {{ option.label }}
    </VanCheckbox>
  </VanCheckboxGroup>
</template>
