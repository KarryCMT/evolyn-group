<script setup lang="ts">
import {
  ElCheckbox,
  ElCheckboxGroup,
  ElDatePicker,
  ElDivider,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
  ElTimePicker,
} from 'element-plus';
import { computed } from 'vue';
import type {
  CheckboxGroupWidget,
  ComboCheckWidget,
  ComboWidget,
  DateTimeWidget,
  DecimalFamilyWidget,
  NumberWidget,
  RadioGroupWidget,
  SeparatorWidget,
  TextAreaWidget,
  TextWidget,
} from '../../schema/types';
import { readWidgetOptions } from '../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../runtime/field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

/**
 * Web 基础字段适配器：将协议字段映射为 Element Plus 控件，同时保持 Runtime 的字符串、
 * 数字与日期时间值协议不变。注册表按 type 注册同一适配器，避免业务 Schema 感知 UI 库。
 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const type = computed(() => props.item.widget.type);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);
const textWidget = computed(() => props.item.widget as TextWidget);
const textareaWidget = computed(() => props.item.widget as TextAreaWidget);
// 多行文本默认维持单行高度；开启自动增高后，仍从一行起随内容扩展。
const textareaAutosize = computed(() =>
  textareaWidget.value.autoHeight ? { minRows: 1 } : undefined,
);
const numberWidget = computed(() => props.item.widget as NumberWidget);
const decimalWidget = computed(() => props.item.widget as DecimalFamilyWidget);
const dateWidget = computed(() => props.item.widget as DateTimeWidget);
const radioWidget = computed(() => props.item.widget as RadioGroupWidget);
const checkboxWidget = computed(() => props.item.widget as CheckboxGroupWidget);
const comboWidget = computed(() => props.item.widget as ComboWidget);
const comboCheckWidget = computed(() => props.item.widget as ComboCheckWidget);
const separatorWidget = computed(() => props.item.widget as SeparatorWidget);
const options = computed(() => readWidgetOptions(props.item.widget));
const inputValue = computed({
  get: () => (typeof props.modelValue === 'string' ? props.modelValue : ''),
  set: (value: string) => emit('update:modelValue', value),
});
const numberValue = computed<number | undefined>({
  get: () => (typeof props.modelValue === 'number' ? props.modelValue : undefined),
  set: (value) => emit('update:modelValue', value ?? null),
});
// 数值字段族：decimal string 原样保存（设计 §16），禁 float 控件承载高精度值。
const decimalValue = computed<string>({
  get: () => (typeof props.modelValue === 'string' ? props.modelValue : ''),
  set: (value) => emit('update:modelValue', value.trim() === '' ? null : value),
});
/** 失焦整理宽松形状（正号/裸小数点），位数与范围约束交由校验层终审。 */
function onDecimalBlur(): void {
  let text = decimalValue.value;
  if (text === '') {
    emit('blur');
    return;
  }
  text = text.trim();
  if (text.startsWith('+')) text = text.slice(1);
  if (text.startsWith('.')) text = '0' + text;
  if (text.endsWith('.') && /^-?\d+\.$/.test(text)) text = text.slice(0, -1);
  if (text !== decimalValue.value) decimalValue.value = text;
  emit('blur');
}
const choicesValue = computed<string>({
  get: () => (typeof props.modelValue === 'string' ? props.modelValue : ''),
  set: (value) => emit('update:modelValue', value === '' ? null : value),
});
const multiChoicesValue = computed<string[]>({
  get: () => (Array.isArray(props.modelValue) ? props.modelValue.filter(isString) : []),
  set: (value) => emit('update:modelValue', value),
});
// time 无日期语义，走独立的 el-time-picker；其余格式映射 el-date-picker 类型。
const isTimeFormat = computed(() => (dateWidget.value.format ?? 'datetime') === 'time');
const dateType = computed<'date' | 'datetime' | 'month'>(() => {
  switch (dateWidget.value.format ?? 'datetime') {
    case 'date':
      return 'date';
    case 'month':
      return 'month';
    default:
      return 'datetime';
  }
});
const dateValueFormat = computed(() => {
  if (isTimeFormat.value) return 'HH:mm';
  switch (dateType.value) {
    case 'date':
      return 'YYYY-MM-DD';
    case 'month':
      return 'YYYY-MM';
    default:
      return 'YYYY-MM-DD HH:mm:ss';
  }
});
const dateValue = computed<string | null>({
  get: () =>
    typeof props.modelValue === 'string' && props.modelValue !== '' ? props.modelValue : null,
  set: (value) => emit('update:modelValue', value || null),
});
const readOnlyDisabled = computed(() => props.disabled || props.readonly);
/** 覆盖 Element Plus 日期/时间选择器默认的 220px CSS 变量。 */
const datePickerStyle = { '--el-date-editor-width': '100%' };

function isString(value: unknown): value is string {
  return typeof value === 'string';
}
</script>

<template>
  <el-input
    v-if="type === 'text'"
    :id="inputId"
    v-model="inputValue"
    :placeholder="textWidget.placeholder ?? '请输入'"
    :maxlength="textWidget.maxLength ?? undefined"
    :disabled="disabled"
    :readonly="readonly"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="emit('blur')"
  />
  <el-input
    v-else-if="type === 'textarea'"
    :id="inputId"
    v-model="inputValue"
    type="textarea"
    :placeholder="textareaWidget.placeholder ?? '请输入'"
    :maxlength="textareaWidget.maxLength ?? undefined"
    :autosize="textareaAutosize"
    :rows="1"
    :disabled="disabled"
    :readonly="readonly"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="emit('blur')"
  />
  <el-input-number
    v-else-if="type === 'number'"
    :id="inputId"
    v-model="numberValue"
    class="evf-web-basic-field__number"
    :min="numberWidget.min ?? undefined"
    :max="numberWidget.max ?? undefined"
    :precision="numberWidget.precision ?? undefined"
    :controls="false"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="emit('blur')"
  />
  <el-input
    v-else-if="type === 'decimal' || type === 'money' || type === 'percent'"
    :id="inputId"
    v-model="decimalValue"
    class="evf-web-basic-field__decimal"
    :placeholder="decimalWidget.placeholder ?? '请输入数值'"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="onDecimalBlur"
  />
  <el-time-picker
    v-else-if="type === 'datetime' && isTimeFormat"
    :id="inputId"
    v-model="dateValue"
    class="evf-web-basic-field__date"
    :style="datePickerStyle"
    value-format="HH:mm"
    :placeholder="dateWidget.placeholder || '请选择时间'"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="emit('blur')"
  />
  <el-date-picker
    v-else-if="type === 'datetime'"
    :id="inputId"
    v-model="dateValue"
    class="evf-web-basic-field__date"
    :style="datePickerStyle"
    :type="dateType"
    :value-format="dateValueFormat"
    :placeholder="dateWidget.placeholder || '请选择日期时间'"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @blur="emit('blur')"
  />
  <el-radio-group
    v-else-if="type === 'radiogroup'"
    v-model="choicesValue"
    :class="{ 'evf-web-basic-field__choices--vertical': radioWidget.layout !== 'horizontal' }"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="emit('blur')"
  >
    <el-radio v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </el-radio>
  </el-radio-group>
  <el-checkbox-group
    v-else-if="type === 'checkboxgroup'"
    v-model="multiChoicesValue"
    :class="{ 'evf-web-basic-field__choices--vertical': checkboxWidget.layout !== 'horizontal' }"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="emit('blur')"
  >
    <el-checkbox v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </el-checkbox>
  </el-checkbox-group>
  <el-select
    v-else-if="type === 'combo'"
    v-model="choicesValue"
    class="evf-web-basic-field__select"
    :placeholder="comboWidget.placeholder || '请选择'"
    :filterable="comboWidget.filterable"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="emit('blur')"
  >
    <el-option
      v-for="option in options"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </el-select>
  <el-select
    v-else-if="type === 'combocheck'"
    v-model="multiChoicesValue"
    class="evf-web-basic-field__select"
    multiple
    collapse-tags
    :placeholder="comboCheckWidget.placeholder || '请选择'"
    :disabled="readOnlyDisabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="emit('blur')"
  >
    <el-option
      v-for="option in options"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </el-select>
  <el-divider
    v-else-if="type === 'separator'"
    class="evf-web-basic-field__divider"
    :direction="separatorWidget.direction ?? 'horizontal'"
    :border-style="separatorWidget.borderStyle ?? separatorWidget.style ?? 'solid'"
    :content-position="separatorWidget.contentPosition ?? 'center'"
  >
    <span v-if="separatorWidget.content">{{ separatorWidget.content }}</span>
  </el-divider>
</template>

<style scoped lang="scss">
.evf-web-basic-field__select {
  width: 100%;
}

// Element Plus 数字输入框默认宽度为 150px，需覆盖后才能与标准输入框同宽。
.evf-web-basic-field__number {
  width: 100% !important;
}

// Element Plus 通过 --el-date-editor-width 设定日期/时间选择器默认 220px；
// 根节点显式覆盖该变量，再以 width 规则兜底，使其随字段 lineWidth 自适应。
.evf-web-basic-field__date {
  width: 100% !important;
}

.evf-web-basic-field__choices--vertical {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-lg);
  align-items: flex-start;
}

.evf-web-basic-field__divider {
  margin: var(--el-space-xs) 0;
}
</style>
