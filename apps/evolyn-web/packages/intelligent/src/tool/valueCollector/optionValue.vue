<template>
  <div>
    <el-select 
      v-model="localValue"
      filterable
      class="use-property" 
      size="small" 
      placeholder="请选择"
      @change="handleChange"
    >
      <el-option
        v-for="item in options"
        :key="item.value"
        :label="item.label"
        size="small"
        :value="item.value"
      />
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue';

defineOptions({ name: 'OptionValue' });

interface OptionItem {
  dataType?: string;
  label: string;
  value: string;
}

interface OptionValueModel {
  dataType?: string;
  type: 'option';
  value: string;
  valueDesc?: string;
}

const props = withDefaults(
  defineProps<{ options?: OptionItem[]; value?: Partial<OptionValueModel> }>(),
  { options: () => [], value: () => ({}) },
);
const emit = defineEmits<{ change: [value: OptionValueModel] }>();
const localValue = shallowRef('');

watch(
  () => props.value,
  (value) => {
    if (value?.type === 'option') localValue.value = value.value ?? '';
  },
  { immediate: true },
);

function handleChange(value: string): void {
  localValue.value = value;
  const option = props.options.find((item) => item.value === value);
  emit('change', {
    type: 'option',
    value,
    dataType: option?.dataType,
    valueDesc: option?.label,
  });
}
</script>

<!-- <style scoped lang="less">
:deep(.el-select) {
  width: 100%;
}
:deep(.el-input__inner) {
 border-radius: 0 4px 4px 0;
}
</style> -->
