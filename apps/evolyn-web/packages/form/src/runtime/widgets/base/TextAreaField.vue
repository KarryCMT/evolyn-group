<script setup lang="ts">
import { computed } from 'vue';
import type { TextAreaWidget } from '../../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/** 多行文本（textarea）：值协议 string|null；autoHeight 时随内容增高。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as TextAreaWidget);
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
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value);
}
</script>

<template>
  <textarea
    :id="inputId"
    class="evf-textarea"
    :class="{ 'evf-textarea--auto': widget.autoHeight }"
    rows="3"
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
