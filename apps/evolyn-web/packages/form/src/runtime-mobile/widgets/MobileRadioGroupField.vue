<script setup lang="ts">
import { Radio as VanRadio, RadioGroup as VanRadioGroup } from 'vant';
import { computed } from 'vue';
import type { RadioGroupWidget } from '../../schema/types';
import { readWidgetOptions } from '../../schema/codec';
import { fieldAriaDescribedBy, fieldLabelId } from '../../runtime/field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as RadioGroupWidget);
const options = computed(() => readWidgetOptions(widget.value));
const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    widget.value.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function updateValue(value: string | number): void {
  emit('update:modelValue', String(value));
}
</script>

<template>
  <VanRadioGroup
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
    <VanRadio v-for="option in options" :key="option.value" :name="option.value">
      {{ option.label }}
    </VanRadio>
  </VanRadioGroup>
</template>
