<script setup lang="ts">
import {
  ElCheckbox,
  ElCheckboxGroup,
  ElDatePicker,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
  ElTimePicker,
} from 'element-plus';
import { computed, inject, type Component } from 'vue';
import { readWidgetOptions } from '../../schema/codec';
import type { DateTimeWidget, FormItem, FormJsonValue, NumberWidget } from '../../schema/types';
import { FormRendererContextKey } from '../../runtime/store/injection';

/**
 * 子表单单元格编辑器：表格与行编辑弹窗共用同一套值映射，新增可发布子字段时只需
 * 在这里增加一个分支，不会让表格布局和行操作逻辑同步膨胀。
 */
const props = defineProps<{
  field: FormItem;
  modelValue: FormJsonValue;
  disabled: boolean;
  readonly: boolean;
  invalid?: boolean;
  inputId?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: unknown];
  blur: [];
}>();

// 子表单与顶层字段共享宿主注册表。通用 form 包不依赖租户目录实现，宿主可为人员、
// 部门等重型字段按需注册组件；独立挂载该编辑器时允许没有渲染器上下文。
const rendererContext = inject(FormRendererContextKey, null);
const isInteractiveDisabled = computed(
  () => props.disabled || props.readonly || !props.field.widget.enable,
);
const organizationFieldComponent = computed<Component | null>(() => {
  const type = props.field.widget.type;
  if (!['user', 'usergroup', 'dept', 'deptgroup'].includes(type)) return null;
  return rendererContext?.registry.resolve(type) ?? null;
});

const stringValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const numberValue = computed(() =>
  typeof props.modelValue === 'number' ? props.modelValue : undefined,
);
const multiValue = computed(() =>
  Array.isArray(props.modelValue)
    ? props.modelValue.filter((entry): entry is string => typeof entry === 'string')
    : [],
);
const inputId = computed(() => props.inputId ?? `evf-subform-${props.field.widget.widgetName}`);

function dateWidget(): DateTimeWidget {
  return props.field.widget as DateTimeWidget;
}

function numberWidget(): NumberWidget {
  return props.field.widget as NumberWidget;
}

function isTimeField(): boolean {
  return (dateWidget().format ?? 'datetime') === 'time';
}

function dateType(): 'date' | 'datetime' | 'month' {
  switch (dateWidget().format ?? 'datetime') {
    case 'date':
      return 'date';
    case 'month':
      return 'month';
    default:
      return 'datetime';
  }
}

function dateValueFormat(): string {
  if (isTimeField()) return 'HH:mm';
  switch (dateType()) {
    case 'date':
      return 'YYYY-MM-DD';
    case 'month':
      return 'YYYY-MM';
    default:
      return 'YYYY-MM-DD HH:mm:ss';
  }
}

function placeholder(): string {
  const widget = props.field.widget as { placeholder?: unknown };
  if (typeof widget.placeholder === 'string') return widget.placeholder;
  return props.field.widget.type === 'datetime' ? '请选择日期时间' : '请输入';
}

function options() {
  return readWidgetOptions(props.field.widget);
}

function update(value: unknown): void {
  emit('update:modelValue', value);
}

function blur(): void {
  emit('blur');
}
</script>

<template>
  <ElInput
    v-if="field.widget.type === 'text'"
    :id="inputId"
    :model-value="stringValue"
    :placeholder="placeholder()"
    :disabled="isInteractiveDisabled"
    :readonly="readonly"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <ElInput
    v-else-if="field.widget.type === 'textarea'"
    :id="inputId"
    :model-value="stringValue"
    type="textarea"
    :placeholder="placeholder()"
    :rows="3"
    :autosize="field.widget.autoHeight || undefined"
    :disabled="isInteractiveDisabled"
    :readonly="readonly"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <ElInputNumber
    v-else-if="field.widget.type === 'number'"
    :id="inputId"
    :model-value="numberValue"
    class="evf-web-subform-cell__full-width"
    :min="numberWidget().min ?? undefined"
    :max="numberWidget().max ?? undefined"
    :precision="numberWidget().precision ?? undefined"
    :controls="false"
    :disabled="isInteractiveDisabled"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <ElTimePicker
    v-else-if="field.widget.type === 'datetime' && isTimeField()"
    :id="inputId"
    :model-value="stringValue || null"
    class="evf-web-subform-cell__full-width"
    value-format="HH:mm"
    :placeholder="placeholder()"
    :disabled="isInteractiveDisabled"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <ElDatePicker
    v-else-if="field.widget.type === 'datetime'"
    :id="inputId"
    :model-value="stringValue || null"
    class="evf-web-subform-cell__full-width"
    :type="dateType()"
    :value-format="dateValueFormat()"
    :placeholder="placeholder()"
    :disabled="isInteractiveDisabled"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <ElRadioGroup
    v-else-if="field.widget.type === 'radiogroup'"
    :model-value="stringValue"
    class="evf-web-subform-cell__choice-group"
    :disabled="isInteractiveDisabled"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @change="blur"
  >
    <ElRadio v-for="option in options()" :key="option.value" :value="option.value">
      {{ option.label }}
    </ElRadio>
  </ElRadioGroup>
  <ElCheckboxGroup
    v-else-if="field.widget.type === 'checkboxgroup'"
    :model-value="multiValue"
    class="evf-web-subform-cell__choice-group"
    :disabled="isInteractiveDisabled"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @change="blur"
  >
    <ElCheckbox v-for="option in options()" :key="option.value" :value="option.value">
      {{ option.label }}
    </ElCheckbox>
  </ElCheckboxGroup>
  <ElSelect
    v-else-if="field.widget.type === 'combo'"
    :id="inputId"
    :model-value="stringValue"
    class="evf-web-subform-cell__full-width"
    :placeholder="placeholder()"
    :filterable="field.widget.filterable"
    :disabled="isInteractiveDisabled"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @change="blur"
  >
    <ElOption
      v-for="option in options()"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </ElSelect>
  <ElSelect
    v-else-if="field.widget.type === 'combocheck'"
    :id="inputId"
    :model-value="multiValue"
    class="evf-web-subform-cell__full-width"
    multiple
    collapse-tags
    :placeholder="placeholder()"
    :disabled="isInteractiveDisabled"
    :validate-event="false"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @change="blur"
  >
    <ElOption
      v-for="option in options()"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </ElSelect>
  <component
    v-else-if="organizationFieldComponent"
    :is="organizationFieldComponent"
    :item="field"
    :model-value="modelValue"
    :disabled="isInteractiveDisabled"
    :readonly="readonly"
    :errors="invalid ? ['字段校验失败'] : []"
    :class="{ 'is-error': invalid }"
    @update:model-value="update"
    @blur="blur"
  />
  <span v-else class="evf-web-subform-cell__unsupported">不支持的子字段</span>
</template>

<style scoped lang="scss">
.evf-web-subform-cell {
  &__full-width {
    width: 100%;
  }

  &__choice-group {
    display: flex;
    flex-flow: row wrap;
    gap: var(--el-space-sm) var(--el-space-lg);
    min-height: 32px;
    align-items: center;
  }

  &__unsupported {
    color: var(--el-text-color-secondary);
  }
}
</style>
