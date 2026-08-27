<script setup lang="ts">
import { computed } from 'vue';
import type { ComboWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/** 下拉单选（combo）：值协议 string|null；原生 select 最小实现。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as ComboWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const options = computed(() => readWidgetOptions(widget.value));
const placeholder = computed(() => widget.value.placeholder ?? '');
const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onChange(event: Event): void {
  const raw = (event.target as HTMLSelectElement).value;
  emit('update:modelValue', raw === '' ? null : raw);
}
</script>

<template>
  <select
    :id="inputId"
    class="evf-select"
    :value="modelValue"
    :disabled="disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="onChange"
    @blur="emit('blur')"
  >
    <option value="" :disabled="!item.widget.allowBlank">{{ placeholder || '请选择' }}</option>
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>
