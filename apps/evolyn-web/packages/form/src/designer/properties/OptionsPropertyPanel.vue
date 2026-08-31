<script setup lang="ts">
import { RiAddFill, RiDeleteBin6Fill } from '@remixicon/vue';
import { ElButton, ElFormItem, ElInput, ElRadioButton, ElRadioGroup, ElSwitch } from 'element-plus';
import type {
  CheckboxGroupWidget,
  ComboCheckWidget,
  ComboWidget,
  RadioGroupWidget,
} from '../../schema/types';

type OptionsWidget = RadioGroupWidget | CheckboxGroupWidget | ComboWidget | ComboCheckWidget;
const props = defineProps<{ widget: OptionsWidget }>();
const hasPlaceholder = ['combo', 'combocheck'].includes(props.widget.type);
const hasLayout = ['radiogroup', 'checkboxgroup'].includes(props.widget.type);
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
  <el-form-item v-if="hasPlaceholder" label="占位提示"
    ><el-input v-model="widget.placeholder" :maxlength="100" placeholder="请选择"
  /></el-form-item>
  <el-form-item v-if="widget.type === 'combo'" label="可搜索"
    ><el-switch v-model="widget.filterable"
  /></el-form-item>
  <el-form-item v-if="hasLayout" label="布局"
    ><el-radio-group v-model="widget.layout"
      ><el-radio-button value="vertical">纵向</el-radio-button
      ><el-radio-button value="horizontal">横向</el-radio-button></el-radio-group
    ></el-form-item
  >
  <el-form-item label="选项（label 与 value 同步维护）"
    ><div class="form-schema-property__options">
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
    </div></el-form-item
  >
</template>
