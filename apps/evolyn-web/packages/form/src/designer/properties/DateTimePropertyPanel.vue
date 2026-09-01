<script setup lang="ts">
import { ElInput, ElOption, ElSelect } from 'element-plus';
import type { DateTimeWidget } from '../../schema/types';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
defineProps<{ widget: DateTimeWidget }>();
</script>
<template>
  <FormSchemaPropertySection title="格式">
    <el-select v-model="widget.format" aria-label="日期时间格式">
      <el-option label="日期时间（年月日 时分秒）" value="datetime" />
      <el-option label="日期（年月日）" value="date" />
      <el-option label="月份（年月）" value="month" />
      <el-option label="时间（时分）" value="time" />
    </el-select>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="默认值">
    <el-input v-model="widget.defaultValue" :maxlength="32" aria-label="默认值" />
  </FormSchemaPropertySection>
</template>
