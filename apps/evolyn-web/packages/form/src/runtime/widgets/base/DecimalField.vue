<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import type { DecimalFamilyWidget } from '../../../schema/types';
import { formatMoneyInputValue, moneyCurrencySymbol, parseMoneyInput } from '../../../schema/money';
import { formatPercentRatio, parsePercentInput, usesPercentRatio } from '../../../schema/percent';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';

/**
 * 数值字段族（decimal/money/percent）：值协议 decimal string|null（设计 §15/§16
 * ——输入组件产出字符串原样保存，运算时才经 NumericRuntime；禁止 float 中转）。
 * 失焦时仅做宽松形状整理（去空白/正号、补裸小数点前导 0），位数与范围约束
 * 交给校验层（schema/codec，前后端一致）；百分比在此显示为 `15%`、存为
 * `0.15`；金额失焦后以字段币种显示千分位与定长小数，底层仍为原币种 decimal。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const widget = computed(() => props.item.widget as DecimalFamilyWidget);
const isPercent = computed(() => usesPercentRatio(widget.value));
const isMoney = computed(() => widget.value.type === 'money');
const isMoneyEditing = shallowRef(false);
const moneySymbol = computed(() => moneyCurrencySymbol(widget.value));
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const placeholder = computed(() => widget.value.placeholder ?? '');
const modelValue = computed(() => {
  const value = typeof props.modelValue === 'string' ? props.modelValue : '';
  if (isPercent.value) return formatPercentRatio(value);
  return isMoney.value && !isMoneyEditing.value
    ? formatMoneyInputValue(value, widget.value)
    : value;
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
  emit(
    'update:modelValue',
    isPercent.value
      ? parsePercentInput(raw)
      : isMoney.value
        ? parseMoneyInput(raw, widget.value)
        : raw,
  );
}

function onFocus(): void {
  isMoneyEditing.value = true;
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
  if (text !== raw) {
    emit(
      'update:modelValue',
      isPercent.value
        ? parsePercentInput(text)
        : isMoney.value
          ? parseMoneyInput(text, widget.value)
          : text,
    );
  }
  isMoneyEditing.value = false;
  emit('blur');
}
</script>

<template>
  <div
    class="evf-decimal-input"
    :class="{ 'evf-decimal-input--percent': isPercent, 'evf-decimal-input--money': isMoney }"
  >
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
      @focus="onFocus"
      @input="onInput"
      @blur="onBlur"
    />
    <span v-if="isPercent" class="evf-decimal-input__suffix" aria-hidden="true">%</span>
    <span v-if="isMoney" class="evf-decimal-input__prefix" aria-hidden="true">{{
      moneySymbol
    }}</span>
  </div>
</template>

<style scoped>
.evf-decimal-input {
  position: relative;
}

.evf-decimal-input--percent .evf-input {
  padding-right: 2rem;
}

.evf-decimal-input--money .evf-input {
  padding-left: 2.4rem;
}

.evf-decimal-input__suffix {
  position: absolute;
  top: 50%;
  right: 0.75rem;
  color: var(--evf-text-secondary, #6b7280);
  pointer-events: none;
  transform: translateY(-50%);
}

.evf-decimal-input__prefix {
  position: absolute;
  top: 50%;
  left: 0.75rem;
  color: var(--evf-text-secondary, #6b7280);
  pointer-events: none;
  transform: translateY(-50%);
}
</style>
