<script setup lang="ts">
import { computed } from 'vue';
import type { TextWidget } from '../../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 单行文本（text）：原生 input 的最小包装，不依赖 UI 组件库。
 * 值协议：string；空值在 Store 侧以 null 表示，输入态空字符串仅存在于组件内部。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as TextWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const placeholder = computed(() => widget.value.placeholder ?? '');
const maxLength = computed(() => widget.value.maxLength ?? undefined);
const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value);
}
</script>

<template>
  <input
    :id="inputId"
    class="evf-input"
    type="text"
    :value="modelValue"
    :placeholder="placeholder"
    :maxlength="maxLength"
    :disabled="disabled"
    :readonly="readonly"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @input="onInput"
    @blur="emit('blur')"
  />
</template>
