<template>
  <div>
    <el-input 
      class="input"
      v-model="localValue" 
      placeholder="请输入值" 
      size="small"
      type="textarea"
      :rows="1"
      @change="handleChange"
    >
    </el-input>
    <el-select v-model="dataType" size="small" placeholder="请选择数据类型" filterable @change="handleTypeChange">
      <el-option v-for="item in typeOptions" :key="item.value" :label="item.label" size="small" :value="item.value">
        <span style="float: left">
          {{ item.label }}
        </span>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue';
import { TypeOptions } from '../../util/dataType';

defineOptions({ name: 'InputValue' });

interface InputValueModel {
  dataType: string;
  type: 'input';
  value: unknown;
}

const props = defineProps<{ context?: unknown; value?: Partial<InputValueModel> }>();
const emit = defineEmits<{ change: [value: InputValueModel] }>();
const localValue = shallowRef<unknown>('');
const dataType = shallowRef('string');
const typeOptions = TypeOptions as Array<{ label: string; value: string }>;

watch(
  () => props.value,
  (value) => {
    if (value?.type !== 'input') return;
    localValue.value = value.value ?? '';
    dataType.value = value.dataType ?? 'string';
  },
  { immediate: true },
);

function emitValue(): void {
  emit('change', { type: 'input', value: localValue.value, dataType: dataType.value || 'string' });
}

function handleTypeChange(value: string): void {
  dataType.value = value;
  emitValue();
}

function handleChange(value: unknown): void {
  localValue.value = value;
  emitValue();
}
</script>

<style scoped lang="less">
.input {
  margin-bottom: 2px;
}
</style>
