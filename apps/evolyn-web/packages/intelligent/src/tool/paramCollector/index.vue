<template>
  <div>
    <el-table :data="localParamList" :border="true" type="selection" style="width: 100%">
      <el-table-column prop="key" label="KEY" width="160">
        <template #default="{ row }">
          <div v-if="row.keyType !== 'custom'">
            <span class="required">{{ row.required ? '*' : '' }}</span>
            {{ row.key }}
          </div>
          <el-input
            v-else
            v-model="row.key"
            placeholder="请输入"
            size="small"
            @change="handleKeyChange($event, row)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="value" label="VALUE">
        <template #default="{ row }">
          <ValueCollector
            class="value-select"
            :model-value="param[row.key]"
            @update:model-value="handleValueChange($event, row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="60">
        <template #default="{ row }">
          <el-button
            circle
            type="danger"
            size="small"
            :disabled="!!row.required"
            aria-label="删除参数"
            @click="removeParam(row)"
          >
            ×
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-link type="primary" :underline="false" class="add-button" @click="addCustom">
      <i class="el-icon-circle-plus-outline" />
      添加自定义参数
    </el-link>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue';
import type { IntelligentValueSource } from '../../schema';
import ValueCollector from '../valueCollector/index.vue';

defineOptions({ name: 'IntelligentParamCollector' });

interface ParamRow {
  key: string;
  keyType?: string;
  paramType?: string;
  required?: boolean | number;
  value?: IntelligentValueSource;
}

const props = withDefaults(
  defineProps<{
    paramList?: ParamRow[];
  }>(),
  { paramList: () => [] },
);
const model = defineModel<ParamRow[]>({ default: () => [] });
const emit = defineEmits<{ change: [value: ParamRow[]] }>();
const localParamList = ref<ParamRow[]>([]);
const param = reactive<Record<string, IntelligentValueSource>>({});

function syncParam(value: ParamRow[]): void {
  for (const key of Object.keys(param)) delete param[key];
  for (const item of value) {
    if (item.key && item.value) param[item.key] = { ...item.value };
  }
}

watch(
  () => props.paramList,
  (value) => {
    localParamList.value = value.map((item) => ({ ...item }));
  },
  { immediate: true, deep: true },
);
watch(model, syncParam, { immediate: true, deep: true });

function formatParam(): ParamRow[] {
  return Object.entries(param).map(([key, value]) => {
    const row = localParamList.value.find((item) => item.key === key);
    return {
      key,
      keyType: row?.keyType ?? 'custom',
      paramType: row?.paramType,
      required: row?.required,
      value: { ...value },
    };
  });
}
function publish(): void {
  const nextValue = formatParam();
  model.value = nextValue;
  emit('change', nextValue);
}
function handleValueChange(value: IntelligentValueSource, row: ParamRow): void {
  param[row.key] = { ...value };
  publish();
}
function handleKeyChange(_value: string, row: ParamRow): void {
  if (row.key) param[row.key] = row.value ? { ...row.value } : { type: 'constant' };
  publish();
}
function removeParam(row: ParamRow): void {
  localParamList.value = localParamList.value.filter((item) => item !== row);
  delete param[row.key];
  publish();
}
function addCustom(): void {
  localParamList.value.push({ key: '', required: false, keyType: 'custom' });
}
</script>

<style scoped lang="less">
:deep(.el-select) {
  width: 100%;
}
.required {
  color: #f56c6c;
  width: 6px;
  display: inline-block;
  text-align: right;
}
.add-button {
  margin-top: 10px;
}
</style>
