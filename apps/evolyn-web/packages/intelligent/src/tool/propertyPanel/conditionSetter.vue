<template>
  <div class="setter-wrapper">
    <condition-item
      v-for="(item, index) in conditions"
      :key="index"
      v-model="conditions[index]"
      :title="`条件${index + 1}（C${index + 1}）`"
      :context="context"
      :current="current"
      :lf="lf"
      @change="handleConditionChange($event, index)"
      @delete="handleConditionDelete(index)"
      class="setter-item"
    ></condition-item>
    <el-link 
      type="primary" 
      :underline="false"
      class="add-button"
      @click="addCondition"
    >
      <i class="el-icon-circle-plus-outline"></i>
      添加条件
    </el-link>
    <el-row class="setter-footer">
      <el-radio-group v-model="combineType" size="small" v-if="conditions.length" @change="handleCombineTypeChange">
        <el-radio :label="1" size="small">满足所有条件</el-radio>
        <el-radio :label="2">满足任意条件</el-radio>
        <el-radio :label="3">自定义</el-radio>
        <el-input 
          class="input"
          v-model="combineRule" 
          placeholder="例如：C1&&(C2||C3)" 
          size="small"
          :disabled="combineType!==3"
          @change="handleCombineTypeChange"
        >
        </el-input>
      </el-radio-group>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import type { IntelligentValueSource } from '../../schema';
import ConditionItem from './conditionItem.vue';

defineOptions({ name: 'IntelligentConditionSetter' });

interface ConditionGroup {
  combineType: 1 | 2 | 3;
  combineRule: string;
  conditions: LegacyConditionValue[];
}

interface LegacyConditionValue {
  operator: string;
  key: IntelligentValueSource;
  value: IntelligentValueSource;
}

defineProps<{ lf?: unknown; context?: unknown; current?: unknown }>();
const model = defineModel<ConditionGroup>({
  default: () => ({ combineType: 1, combineRule: '', conditions: [] }),
});
const emit = defineEmits<{ change: [value: ConditionGroup] }>();
const combineType = ref<ConditionGroup['combineType']>(1);
const combineRule = ref('');
const conditions = ref<LegacyConditionValue[]>([]);

watch(
  model,
  (value) => {
    combineType.value = value.combineType ?? 1;
    combineRule.value = value.combineRule ?? '';
    conditions.value = (value.conditions ?? []).map((item) => ({
      ...item,
      key: { ...item.key },
      value: { ...item.value },
    }));
    updateGeneratedRule();
  },
  { immediate: true, deep: true },
);

function updateGeneratedRule(): void {
  if (combineType.value === 1) {
    combineRule.value = conditions.value.map((_item, index) => `C${index + 1}`).join('&&');
  } else if (combineType.value === 2) {
    combineRule.value = conditions.value.map((_item, index) => `C${index + 1}`).join('||');
  }
}

function publish(): void {
  const value: ConditionGroup = {
    combineType: combineType.value,
    combineRule: combineRule.value,
    conditions: conditions.value.map((item) => ({
      ...item,
      key: { ...item.key },
      value: { ...item.value },
    })),
  };
  model.value = value;
  emit('change', value);
}

function handleConditionChange(value: LegacyConditionValue, index: number): void {
  conditions.value[index] = value;
  publish();
}

function handleConditionDelete(index: number): void {
  conditions.value.splice(index, 1);
  updateGeneratedRule();
  publish();
}

function handleCombineTypeChange(): void {
  updateGeneratedRule();
  publish();
}

function addCondition(): void {
  if (conditions.value.length >= 6) {
    ElMessage.warning('一条线最多允许添加6个判断条件！');
    return;
  }
  conditions.value.push({
    operator: '==',
    key: { type: 'trigger-field' },
    value: { type: 'constant' },
  });
  updateGeneratedRule();
  publish();
}
</script>

<style scoped lang="less">
.setter-wrapper {
  width: 100%;
  font-size: 12px;
}

.setter-item {
  margin-bottom: 10px;
}

.setter-header {
  margin-bottom: 20px;
}
.setter-footer {
  margin-top: 20px;
}

:deep(.el-radio__label) {
  font-size: 12px;
  padding-left: 4px;
}
:deep(.el-radio) {
  &:not(:last-of-type){
    margin-right: 20px;
  }
  margin-right: 4px;
}
.input {
  margin-top: 10px;
}
</style>
