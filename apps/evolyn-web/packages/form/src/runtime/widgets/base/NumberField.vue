<script setup lang="ts">
import { computed } from 'vue';
import type { NumberWidget } from '../../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 数字（number）：值协议 number|null；空输入与非有限值回写 null，
 * 范围与小数位约束交给校验层（schema/codec，前后端一致）。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as NumberWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const placeholder = computed(() => widget.value.placeholder ?? '');
const min = computed(() => widget.value.min ?? undefined);
const max = computed(() => widget.value.max ?? undefined);
// step 按小数位推导（precision=2 → 0.01），未启用时 any。
const step = computed(() => {
  const precision = widget.value.precision;
  return precision === null || precision === undefined ? 'any' : String(1 / 10 ** precision);
});
const modelValue = computed(() =>
  typeof props.modelValue === 'number' && Number.isFinite(props.modelValue)
    ? String(props.modelValue)
    : '',
);
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onInput(event: Event): void {
  const raw = (event.target as HTMLInputElement).value;
  if (raw.trim() === '') {
    emit('update:modelValue', null);
    return;
  }
  const parsed = Number(raw);
  emit('update:modelValue', Number.isFinite(parsed) ? parsed : null);
}
</script>

<template>
  <input
    :id="inputId"
    class="evf-input evf-input--number"
    type="number"
    inputmode="decimal"
    :step="step"
    :value="modelValue"
    :placeholder="placeholder"
    :min="min"
    :max="max"
    :disabled="disabled"
    :readonly="readonly"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @input="onInput"
    @blur="emit('blur')"
  />
</template>
