<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Component } from 'vue';
import type { FormItem } from '../../schema/types';
import { isLayoutWidgetType } from '../../schema/codec';
import { sanitizeRichTextDescription } from '../../schema/richTextDescription';
import { fieldDescriptionId, fieldErrorId, fieldInputId, fieldLabelId } from '../field-dom';
import { useFormRendererContext } from '../store/injection';
import type { FormValue } from '../types';
import UnsupportedField from '../widgets/base/UnsupportedField.vue';
import FormFieldError from './FormFieldError.vue';

/**
 * 字段外壳：唯一了解「字段通用展示」的组件，承担标签、必填星号、说明、错误、
 * 隐藏标签与跨列（lineWidth）；字段组件只负责输入控件本身。
 * 通过注入的 widgetName 获取自身 modelValue 与状态，输入一个字段仅重渲染该字段。
 * visible=false 的字段在这里整体不渲染（不校验、不收集）。
 */
const props = defineProps<{ item: FormItem }>();

const { runtime, registry, registerFieldFocus, unregisterFieldFocus, reportUnsupportedField } =
  useFormRendererContext();

const rootRef = ref<HTMLElement | null>(null);

const widgetName = computed(() => props.item.widget.widgetName);
const isLayout = computed(() => isLayoutWidgetType(props.item.widget.type));
const widget = computed<Component | null>(() => registry.resolve(props.item.widget.type));
const state = computed(() => runtime.value?.state.fieldStates[widgetName.value]);
const modelValue = computed<FormValue>(() => runtime.value?.state.values[widgetName.value] ?? null);
/**
 * 分割线等布局项没有填写值，运行时不会为它们建立 fieldState；其显示状态直接
 * 由 Schema 控件配置决定，避免被数据字段的 state?.visible 错误过滤掉。
 */
const isVisible = computed(() =>
  isLayout.value ? props.item.widget.visible : (state.value?.visible ?? false),
);
const showDescription = computed(() => props.item.description !== '');
const descriptionHtml = computed(() => sanitizeRichTextDescription(props.item.description));
const inputId = computed(() => fieldInputId(widgetName.value));
const labelId = computed(() => fieldLabelId(widgetName.value));
// 桌面 12 栅格占列数；非法值回退整行，移动端由样式强制单列。
const span = computed(() => {
  const width = props.item.lineWidth;
  return Number.isInteger(width) && width >= 1 && width <= 12 ? width : 12;
});

function onUpdateModelValue(value: FormValue): void {
  runtime.value?.setValue(widgetName.value, value, 'user');
}

function onBlur(): void {
  runtime.value?.markTouched(widgetName.value);
}

/** 聚焦外壳内第一个可聚焦控件，供错误定位使用。 */
function focusSelf(): void {
  const target = rootRef.value?.querySelector<HTMLElement>('input, select, textarea');
  target?.focus();
}

defineExpose({ focus: focusSelf });

onMounted(() => {
  registerFieldFocus(widgetName.value, focusSelf);
  // 诊断回传仅执行一次，避免重渲染期间重复上报宿主。
  if (!widget.value) {
    reportUnsupportedField({ fieldKey: widgetName.value, type: props.item.widget.type });
  }
});

onBeforeUnmount(() => {
  unregisterFieldFocus(widgetName.value);
});
</script>

<template>
  <div
    v-if="isVisible"
    ref="rootRef"
    class="evf-field"
    :class="{
      'evf-field--error': (state?.errors.length ?? 0) > 0,
      'evf-field--layout': isLayout,
      'evf-field--disabled': state?.disabled ?? false,
      'evf-field--readonly': state?.readonly ?? false,
    }"
    :style="isLayout ? undefined : { '--evf-field-span': span }"
  >
    <!-- 布局字段（分割线等）跳过通用外壳，直接渲染组件。 -->
    <template v-if="isLayout">
      <!-- 分割线等布局项自行决定说明内容的呈现位置。 -->
      <component
        :is="widget"
        :item="item"
        :model-value="modelValue"
        :disabled="false"
        :readonly="false"
        :errors="[]"
      />
    </template>
    <template v-else>
      <label v-if="!item.labelHidden" :id="labelId" class="evf-field__label" :for="inputId">
        <span v-if="!item.widget.allowBlank" class="evf-field__required" aria-hidden="true">*</span>
        <span class="evf-field__label-text">{{ item.label }}</span>
      </label>
      <div
        v-if="showDescription"
        :id="fieldDescriptionId(widgetName)"
        class="evf-field__description"
        v-html="descriptionHtml"
      />
      <div class="evf-field__control">
        <component
          :is="widget ?? UnsupportedField"
          :item="item"
          :model-value="modelValue"
          :disabled="state.disabled"
          :readonly="state.readonly"
          :errors="state.errors"
          @update:model-value="onUpdateModelValue"
          @blur="onBlur"
        />
      </div>
      <FormFieldError :id="fieldErrorId(widgetName)" :messages="state.errors" />
    </template>
  </div>
</template>
