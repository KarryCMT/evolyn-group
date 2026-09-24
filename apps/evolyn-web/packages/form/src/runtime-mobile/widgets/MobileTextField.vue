<script setup lang="ts">
import { Field as VanField } from 'vant';
import { computed } from 'vue';
import type { TextAreaWidget, TextWidget } from '../../schema/types';
import { fieldAriaDescribedBy, fieldInputId } from '../../runtime/field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

/** Vant 文本输入适配：保持 Runtime Core 的 string 值协议与统一失焦校验语义。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as TextWidget | TextAreaWidget);
const isTextarea = computed(() => widget.value.type === 'textarea');
const inputId = computed(() => fieldInputId(widget.value.widgetName));
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
  <VanField
    :id="inputId"
    class="evf-mobile-field"
    :model-value="modelValue"
    :type="isTextarea ? 'textarea' : 'text'"
    :rows="isTextarea ? 3 : undefined"
    :autosize="isTextarea && Boolean((widget as TextAreaWidget).autoHeight)"
    :placeholder="widget.placeholder || '请输入'"
    :maxlength="widget.maxLength ?? undefined"
    :show-word-limit="Boolean(widget.maxLength)"
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
