<script setup lang="ts">
import { ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect } from 'element-plus';
import type { TextWidget } from '../../schema/types';
defineProps<{ widget: TextWidget }>();
</script>
<template>
  <el-form-item label="占位提示"
    ><el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入"
  /></el-form-item>
  <el-form-item label="格式"
    ><el-select v-model="widget.format"
      ><el-option label="不限" value="" /><el-option label="邮箱" value="email" /></el-select
  ></el-form-item>
  <div class="form-schema-property__pair">
    <el-form-item label="最小长度"
      ><el-input-number
        :model-value="widget.minLength ?? undefined"
        :min="0"
        :max="1000"
        placeholder="不限"
        @update:model-value="widget.minLength = $event ?? null" /></el-form-item
    ><el-form-item label="最大长度"
      ><el-input-number
        :model-value="widget.maxLength ?? undefined"
        :min="1"
        :max="1000"
        placeholder="不限"
        @update:model-value="widget.maxLength = $event ?? null"
    /></el-form-item>
  </div>
</template>
