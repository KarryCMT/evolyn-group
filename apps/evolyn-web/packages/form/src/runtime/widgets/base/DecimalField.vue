<script setup lang="ts">
import { computed } from 'vue';
import type { DecimalFamilyWidget } from '../../../schema/types';
import { formatPercentRatio, parsePercentInput, usesPercentRatio } from '../../../schema/percent';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 数值字段族（decimal/money/percent）：值协议 decimal string|null（设计 §15/§16
 * ——输入组件产出字符串原样保存，运算时才经 NumericRuntime；禁止 float 中转）。
 * 失焦时仅做宽松形状整理（去空白/正号、补裸小数点前导 0），位数与范围约束
 * 交给校验层（schema/codec，前后端一致）；百分比在此显示为 `15%`、存为
 * `0.15`，金额的币种符号与分组格式仍由后续 MoneyRuntime 承担。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as DecimalFamilyWidget);
const isPercent = computed(() => usesPercentRatio(widget.value));
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const placeholder = computed(() => widget.value.placeholder ?? '');
const modelValue = computed(() => {
  const value = typeof props.modelValue === 'string' ? props.modelValue : '';
  return isPercent.value ? formatPercentRatio(value) : value;
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
  if (raw.trim() === '') {
    emit('update:modelValue', null);
    return;
  }
  // 输入期不做形状限制（允许用户敲击过程中的临时非法态），提交前终审收口。
  emit('update:modelValue', isPercent.value ? parsePercentInput(raw) : raw);
}

/** 失焦整理：宽松可修复形状收敛为协议形状，其余原样交由校验层拒绝。 */
function onBlur(): void {
  const raw = modelValue.value;
  if (raw === '') {
    emit('blur');
    return;
  }
  let text = raw.trim();
  if (text.startsWith('+')) text = text.slice(1);
  if (text.startsWith('.')) text = '0' + text;
  if (text.endsWith('.') && /^-?\d+\.$/.test(text)) text = text.slice(0, -1);
  if (text !== raw) emit('update:modelValue', isPercent.value ? parsePercentInput(text) : text);
  emit('blur');
}
</script>

<template>
  <div class="evf-decimal-input" :class="{ 'evf-decimal-input--percent': isPercent }">
    <input
      :id="inputId"
      class="evf-input evf-input--decimal"
      type="text"
      inputmode="decimal"
      autocomplete="off"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :aria-required="!item.widget.allowBlank || undefined"
      :aria-invalid="errors.length > 0 || undefined"
      :aria-describedby="describedBy"
      @input="onInput"
      @blur="onBlur"
    />
    <span v-if="isPercent" class="evf-decimal-input__suffix" aria-hidden="true">%</span>
  </div>
</template>

<style scoped>
.evf-decimal-input {
  position: relative;
}

.evf-decimal-input--percent .evf-input {
  padding-right: 2rem;
}

.evf-decimal-input__suffix {
  position: absolute;
  top: 50%;
  right: 0.75rem;
  color: var(--evf-text-secondary, #6b7280);
  pointer-events: none;
  transform: translateY(-50%);
}
</style>
