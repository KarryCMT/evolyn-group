<script setup lang="ts">
import { computed } from 'vue';
import type { DateTimeWidget } from '../../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 日期时间（datetime）：值协议为规范形状字符串（date→YYYY-MM-DD、
 * datetime→YYYY-MM-DD HH:mm:ss、month→YYYY-MM、time→HH:mm，字段字典 §3）。
 * 原生 datetime-local 控件使用「T」分隔且无秒，出入组件时做形状转换；
 * 桌面/移动共用原生控件。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as DateTimeWidget);
const format = computed(() => widget.value.format ?? 'datetime');
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const nativeType = computed(() => (format.value === 'datetime' ? 'datetime-local' : format.value));
const modelValue = computed(() => {
  const value = props.modelValue;
  if (typeof value !== 'string' || value === '') return '';
  if (format.value !== 'datetime') return value;
  // 存储形状「YYYY-MM-DD HH:mm:ss」→ 控件形状「YYYY-MM-DDTHH:mm」。
  return value.replace(' ', 'T').slice(0, 16);
});
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onInput(event: Event): void {
  const raw = (event.target as HTMLInputElement).value;
  if (raw === '') {
    emit('update:modelValue', null);
    return;
  }
  if (format.value !== 'datetime') {
    emit('update:modelValue', raw);
    return;
  }
  // 控件形状 → 规范形状（补秒位）。
  emit('update:modelValue', `${raw.replace('T', ' ')}:00`);
}
</script>

<template>
  <input
    :id="inputId"
    class="evf-input"
    :type="nativeType"
    :value="modelValue"
    :disabled="disabled"
    :readonly="readonly"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @input="onInput"
    @blur="emit('blur')"
  />
</template>
