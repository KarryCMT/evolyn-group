<template>
  <div>
    <el-select 
      v-model="selectedValue" 
      class="use-property" 
      size="small" 
      placeholder="请选择页面组件"
      filterable
      @change="handleChange"
    >
      <el-option
        v-for="item in options"
        :key="item.value"
        size="small"
        :label="item.name"
        :value="item.value"
      >
        <span 
          style="display: block"
          @mouseenter.stop="handleMouseEnter(item.modelId)"
        >
          <b>{{ item.name }}</b> 
        </span>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';

defineOptions({ name: 'IntelligentConditionKey' });

type ConditionKeyValue = string | number | boolean | Record<string, unknown> | unknown[];

interface ConditionKeyOption {
  modelId: string;
  name: string;
  value: ConditionKeyValue;
}

const props = defineProps<{
  context?: unknown;
  defaultValue?: ConditionKeyValue;
}>();
const emit = defineEmits<{ change: [option: ConditionKeyOption | undefined] }>();
const selectedValue = shallowRef<ConditionKeyValue | undefined>(props.defaultValue);
const options = computed<ConditionKeyOption[]>(() => []);

watch(
  () => props.defaultValue,
  (value) => {
    selectedValue.value = value;
  },
);

// 预留画布高亮入口，组件目录接入后在这里发送 hover 事件。
function handleMouseEnter(_modelId: string): void {}

function handleChange(value: ConditionKeyValue): void {
  emit('change', options.value.find((item) => item.value === value));
}
</script>

<style scoped lang="less">
:deep(.el-select) {
  width: 100%;
}
</style>
