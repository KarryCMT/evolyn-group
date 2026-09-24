<template>
  <div>
    <el-input 
      v-model="localValue" 
      placeholder="请输入值" 
      size="small"
      @change="handleChange"
    />
  </div>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue';

defineOptions({ name: 'UrlParamValue' });

interface UrlParamValueModel {
  dataType: string;
  type: 'urlParam';
  value: string;
}

const props = defineProps<{ context?: unknown; value?: Partial<UrlParamValueModel> }>();
const emit = defineEmits<{ change: [value: UrlParamValueModel] }>();
const localValue = shallowRef('');

watch(
  () => props.value,
  (value) => {
    if (value?.type === 'urlParam') localValue.value = value.value ?? '';
  },
  { immediate: true },
);

function handleChange(value: string): void {
  localValue.value = value;
  emit('change', { type: 'urlParam', value, dataType: 'string' });
}
</script>
