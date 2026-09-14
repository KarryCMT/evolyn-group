<script setup lang="ts">
import { computed } from 'vue';
import type { RadioGroupWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId, fieldLabelId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/** 单选组（radiogroup）：值协议 string|null；选项经 readWidgetOptions 归一。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as RadioGroupWidget);
const key = computed(() => props.item.widget.widgetName);
const inputId = computed(() => fieldInputId(key.value));
const labelId = computed(() => fieldLabelId(key.value));
const options = computed(() => readWidgetOptions(widget.value));
// layout 缺失的历史快照按协议默认值横向排列；仅显式 vertical 才切换为纵向。
const vertical = computed(() => widget.value.layout === 'vertical');
const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const describedBy = computed(() =>
  fieldAriaDescribedBy(key.value, props.item.description !== '', props.errors.length > 0),
);

function onChange(event: Event): void {
  const raw = (event.target as HTMLInputElement).value;
  emit('update:modelValue', raw === '' ? null : raw);
}
</script>

<template>
  <div
    class="evf-choice-group"
    :class="{ 'evf-choice-group--vertical': vertical }"
    role="radiogroup"
    :aria-labelledby="labelId"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
  >
    <label v-for="option in options" :key="option.value" class="evf-choice">
      <input
        type="radio"
        :name="inputId"
        :value="option.value"
        :checked="modelValue === option.value"
        :disabled="disabled"
        @change="onChange"
        @blur="emit('blur')"
      />
      <span class="evf-choice__label">{{ option.label }}</span>
    </label>
  </div>
</template>
