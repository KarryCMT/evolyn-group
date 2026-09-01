<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus';

type DefaultValueMode = 'custom' | 'data-linkage' | 'formula';

const mode = defineModel<DefaultValueMode>({ default: 'custom' });

const options: ReadonlyArray<{ label: string; value: DefaultValueMode }> = [
  { label: '自定义', value: 'custom' },
  { label: '数据联动', value: 'data-linkage' },
  { label: '公式编辑', value: 'formula' },
];
</script>

<template>
  <el-select v-model="mode" class="default-value-mode-select" aria-label="默认值类型">
    <el-option
      v-for="option in options"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </el-select>
</template>

<style scoped lang="scss">
.default-value-mode-select {
  width: 100%;
}
</style>
