<script setup lang="ts">
import { ElInput, ElSwitch } from 'element-plus';
import type { TextAreaWidget } from '../../schema/types';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
defineProps<{ widget: TextAreaWidget }>();
</script>
<template>
  <FormSchemaPropertySection title="显示设置">
    <el-switch v-model="widget.autoHeight" inline-prompt active-text="自动增高" />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="默认值">
    <el-input v-model="widget.defaultValue" :maxlength="2000" aria-label="默认值" />
  </FormSchemaPropertySection>
</template>
