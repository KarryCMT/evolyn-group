<script setup lang="ts">
import { RiAddFill, RiDeleteBin6Fill } from '@remixicon/vue';
import {
  ElButton,
  ElInput,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSwitch,
} from 'element-plus';
import type {
  CheckboxGroupWidget,
  ComboCheckWidget,
  ComboWidget,
  RadioGroupWidget,
} from '../../schema/types';
import { computed } from 'vue';
import DefaultValueModeSelect from './DefaultValueModeSelect.vue';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';

type OptionsWidget = RadioGroupWidget | CheckboxGroupWidget | ComboWidget | ComboCheckWidget;
const props = defineProps<{ widget: OptionsWidget }>();
const layoutWidget = computed<RadioGroupWidget | CheckboxGroupWidget | null>(() => {
  switch (props.widget.type) {
    case 'radiogroup':
    case 'checkboxgroup':
      return props.widget;
    default:
      return null;
  }
});
const isMultiple = ['checkboxgroup', 'combocheck'].includes(props.widget.type);
function updateOption(index: number, label: string) {
  props.widget.options[index] = { label, value: label };
}
function addOption() {
  const next = `选项${props.widget.options.length + 1}`;
  props.widget.options.push({ label: next, value: next });
}
function removeOption(index: number) {
  props.widget.options.splice(index, 1);
}
</script>
<template>
  <FormSchemaPropertySection v-if="widget.type === 'combo'" title="显示设置">
    <el-switch v-model="widget.filterable" inline-prompt active-text="可搜索" />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection v-if="layoutWidget" title="布局">
    <el-radio-group v-model="layoutWidget.layout">
      <el-radio-button value="vertical">纵向</el-radio-button>
      <el-radio-button value="horizontal">横向</el-radio-button>
    </el-radio-group>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="默认值">
    <DefaultValueModeSelect />
    <el-select
      v-if="isMultiple"
      v-model="widget.defaultValue"
      multiple
      clearable
      aria-label="默认值"
    >
      <el-option
        v-for="option in widget.options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      />
    </el-select>
    <el-select v-else v-model="widget.defaultValue" clearable aria-label="默认值">
      <el-option
        v-for="option in widget.options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      />
    </el-select>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="选项">
    <div class="form-schema-property__options">
      <div
        v-for="(option, index) in widget.options"
        :key="index"
        class="form-schema-property__option"
      >
        <el-input
          :model-value="option.label"
          :maxlength="100"
          :placeholder="`选项${index + 1}`"
          @update:model-value="updateOption(index, String($event ?? ''))"
        /><el-button
          text
          type="danger"
          :icon="RiDeleteBin6Fill"
          :disabled="widget.options.length <= 1"
          @click="removeOption(index)"
        />
      </div>
      <el-button
        class="form-schema-property__option-add"
        text
        type="primary"
        :icon="RiAddFill"
        :disabled="widget.options.length >= 200"
        @click="addOption"
        >添加选项</el-button
      >
    </div>
  </FormSchemaPropertySection>
</template>
