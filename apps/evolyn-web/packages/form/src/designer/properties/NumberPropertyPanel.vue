<script setup lang="ts">
import { ElFormItem, ElInput, ElInputNumber } from 'element-plus';
import type { NumberWidget } from '../../schema/types';
defineProps<{ widget: NumberWidget }>();
</script>
<template>
  <el-form-item label="占位提示"
    ><el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入"
  /></el-form-item>
  <div class="form-schema-property__pair">
    <el-form-item label="最小值"
      ><el-input-number
        :model-value="widget.min ?? undefined"
        placeholder="不限"
        @update:model-value="widget.min = $event ?? null" /></el-form-item
    ><el-form-item label="最大值"
      ><el-input-number
        :model-value="widget.max ?? undefined"
        placeholder="不限"
        @update:model-value="widget.max = $event ?? null"
    /></el-form-item>
  </div>
  <el-form-item label="小数位数"
    ><el-input-number
      :model-value="widget.precision ?? undefined"
      :min="0"
      :max="8"
      placeholder="不限"
      @update:model-value="widget.precision = $event ?? null"
  /></el-form-item>
</template>
