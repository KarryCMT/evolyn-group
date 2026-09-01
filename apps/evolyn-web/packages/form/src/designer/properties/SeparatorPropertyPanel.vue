<script setup lang="ts">
import { ElInput, ElOption, ElRadioButton, ElRadioGroup, ElSelect } from 'element-plus';
import type { SeparatorWidget } from '../../schema/types';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
const props = defineProps<{ widget: SeparatorWidget }>();

const borderStyles = [
  ['solid', '实线'],
  ['dashed', '虚线'],
  ['dotted', '点线'],
  ['double', '双线'],
  ['groove', '凹槽'],
  ['ridge', '脊线'],
  ['inset', '内嵌'],
  ['outset', '外凸'],
  ['none', '无边框'],
  ['hidden', '隐藏'],
] as const;
</script>
<template>
  <FormSchemaPropertySection title="分割线文案">
    <el-input v-model="props.widget.content" :maxlength="64" placeholder="请输入文案（可留空）" />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="方向">
    <el-radio-group v-model="props.widget.direction">
      <el-radio-button value="horizontal">水平</el-radio-button>
      <el-radio-button value="vertical">垂直</el-radio-button>
    </el-radio-group>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="线条样式">
    <el-select v-model="props.widget.borderStyle" placeholder="实线">
      <el-option
        v-for="[value, label] in borderStyles"
        :key="value"
        :label="label"
        :value="value"
      />
    </el-select>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection v-if="props.widget.direction !== 'vertical'" title="文案位置">
    <el-radio-group v-model="props.widget.contentPosition">
      <el-radio-button value="left">左侧</el-radio-button>
      <el-radio-button value="center">居中</el-radio-button>
      <el-radio-button value="right">右侧</el-radio-button>
    </el-radio-group>
  </FormSchemaPropertySection>
</template>
