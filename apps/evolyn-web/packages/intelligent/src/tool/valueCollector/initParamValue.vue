<template>
  <div>
    <el-input 
      v-model="localValue" 
      placeholder="请输入值" 
      size="small"
      @change="handleChange"
    >
    </el-input>
  </div>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue';

defineOptions({ name: 'InitParamValue' });

interface InitParamValueModel {
  dataType: string;
  type: 'initParam';
  value: string;
}

const props = defineProps<{ context?: unknown; value?: Partial<InitParamValueModel> }>();
const emit = defineEmits<{ change: [value: InitParamValueModel] }>();
const localValue = shallowRef('');

watch(
  () => props.value,
  (value) => {
    if (value?.type === 'initParam') localValue.value = value.value ?? '';
  },
  { immediate: true },
);

function handleChange(value: string): void {
  localValue.value = value;
  emit('change', { type: 'initParam', value, dataType: 'string' });
}
</script>
