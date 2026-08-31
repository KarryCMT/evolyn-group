<script setup lang="ts">
import { ElFormItem, ElInput, ElInputNumber, ElSwitch } from 'element-plus';
import type { TextAreaWidget } from '../../schema/types';
defineProps<{ widget: TextAreaWidget }>();
</script>
<template>
  <el-form-item label="占位提示"
    ><el-input v-model="widget.placeholder" :maxlength="100" placeholder="请输入"
  /></el-form-item>
  <el-form-item label="自动增高"><el-switch v-model="widget.autoHeight" /></el-form-item>
  <div class="form-schema-property__pair">
    <el-form-item label="最小长度"
      ><el-input-number
        :model-value="widget.minLength ?? undefined"
        :min="0"
        :max="2000"
        placeholder="不限"
        @update:model-value="widget.minLength = $event ?? null" /></el-form-item
    ><el-form-item label="最大长度"
      ><el-input-number
        :model-value="widget.maxLength ?? undefined"
        :min="1"
        :max="2000"
        placeholder="不限"
        @update:model-value="widget.maxLength = $event ?? null"
    /></el-form-item>
  </div>
</template>
