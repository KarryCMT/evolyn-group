<script setup lang="ts">
import { computed } from 'vue';
import type { ComboCheckWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/** 下拉多选（combocheck）：值协议 string[]；原生 multiple select 最小实现。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as ComboCheckWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const options = computed(() => readWidgetOptions(widget.value));
const modelValue = computed(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as string[]) : [],
);
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onChange(event: Event): void {
  const selected = Array.from((event.target as HTMLSelectElement).selectedOptions).map(
    (option) => option.value,
  );
  emit('update:modelValue', selected as FormValue);
}
</script>

<template>
  <select
    :id="inputId"
    class="evf-select evf-select--multiple"
    multiple
    :value="modelValue"
    :disabled="disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="onChange"
    @blur="emit('blur')"
  >
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>
