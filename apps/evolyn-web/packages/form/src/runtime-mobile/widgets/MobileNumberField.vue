<script setup lang="ts">
import { Field as VanField } from 'vant';
import { computed } from 'vue';
import type { NumberWidget } from '../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../runtime/field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

/** 普通 number 控件仍使用 JS number 协议；decimal/money/percent 继续走十进制字符串组件。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as NumberWidget);
const inputId = computed(() => fieldInputId(widget.value.widgetName));
const modelValue = computed(() =>
  typeof props.modelValue === 'number' && Number.isFinite(props.modelValue)
    ? String(props.modelValue)
    : '',
);
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    widget.value.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function updateValue(value: string | number): void {
  const raw = String(value).trim();
  if (raw === '') {
    emit('update:modelValue', null);
    return;
  }
  const numeric = Number(raw);
  if (Number.isFinite(numeric)) emit('update:modelValue', numeric);
}
</script>

<template>
  <VanField
    :id="inputId"
    class="evf-mobile-field"
    :model-value="modelValue"
    type="number"
    :placeholder="widget.placeholder || '请输入'"
    :disabled="disabled"
    :readonly="readonly"
    :clearable="!readonly && !disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @update:model-value="updateValue"
    @blur="emit('blur')"
  />
</template>
