<script setup lang="ts">
import { ElFormItem, ElOption, ElSelect } from 'element-plus';
import type { DateTimeWidget } from '../../schema/types';
defineProps<{ widget: DateTimeWidget }>();
</script>
<template>
  <el-form-item label="格式"
    ><el-select v-model="widget.format"
      ><el-option label="日期时间（年月日 时分秒）" value="datetime" /><el-option
        label="日期（年月日）"
        value="date" /><el-option label="月份（年月）" value="month" /><el-option
        label="时间（时分）"
        value="time" /></el-select
  ></el-form-item>
</template>
