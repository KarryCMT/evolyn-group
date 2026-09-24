<template>
  <div class="action-item-wrap">
    <div class="header">
      {{ title }}
      <span class="delete" style="cursor: pointer" @click="handleDelete">
        <i class="el-icon-delete"></i>
        删除
      </span>
    </div>
    <el-row :gutter="8">
      <el-col :span="10">
        <ValueCollector
          :model-value="condition.key"
          @update:model-value="handleKeyChange"
        />
      </el-col>
      <el-col :span="4">
        <el-select
          v-model="condition.operator"
          class="use-property"
          size="small"
          placeholder="请选择"
          @change="handleOperatorChange"
        >
          <el-option v-for="(item, index) in comparisonOperators" :key="index" :label="item.value" :value="item.value">
          </el-option>
        </el-select>
      </el-col>
      <el-col :span="10">
        <ValueCollector
          :model-value="condition.value"
          @update:model-value="handleValueChange"
        />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { IntelligentValueSource } from '../../schema';
import { comparisonOperators } from '../../util/expression';
import ValueCollector from '../valueCollector/index.vue';

defineOptions({ name: 'IntelligentConditionItem' });

interface LegacyConditionValue {
  operator: string;
  key: IntelligentValueSource;
  value: IntelligentValueSource;
}

defineProps<{ lf?: unknown; context?: unknown; current?: unknown; title?: string }>();
const model = defineModel<LegacyConditionValue>({
  default: () => ({
    operator: '==',
    key: { type: 'trigger-field' },
    value: { type: 'constant' },
  }),
});
const emit = defineEmits<{ change: [value: LegacyConditionValue]; delete: [] }>();
const condition = ref<LegacyConditionValue>({ ...model.value });

watch(
  model,
  (value) => {
    condition.value = {
      ...value,
      key: { ...value.key },
      value: { ...value.value },
    };
  },
  { immediate: true, deep: true },
);

function publish(): void {
  const value = {
    ...condition.value,
    key: { ...condition.value.key },
    value: { ...condition.value.value },
  };
  model.value = value;
  emit('change', value);
}

function handleKeyChange(value: IntelligentValueSource): void {
  condition.value.key = value;
  publish();
}

function handleValueChange(value: IntelligentValueSource): void {
  condition.value.value = value;
  publish();
}

function handleOperatorChange(): void {
  publish();
}

function handleDelete(): void {
  emit('delete');
}
</script>

<style scoped lang="less">
.action-item-wrap {
  background: #f3f6fa;
  border-radius: 4px;
  font-family: PingFangSC-Regular;
  font-size: 12px;
  color: #303a51;
  line-height: 16px;
  font-weight: 400;
  padding: 9px 12px;
}

.header {
  border-bottom: 1px solid #dcdfe6;
  margin-bottom: 8px;
  padding: 0 0 4px 0;
}

.delete {
  float: right;
  color: #a8adbd;
  font-weight: 400;
  &:hover {
    color: #2961ef;
  }
}
:deep(.el-col) {
  display: flex;
}
:deep(.el-select) {
  flex: 1;
}
.value-select {
  flex: 1;
}
.delete-button {
  height: 32px;
  font-size: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
}
.action-label {
  margin-right: 10px;
  height: 32px;
  line-height: 32px;
}
</style>
