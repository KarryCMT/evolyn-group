<template>
  <div class="action-item-wrap">
    <div class="header">
      {{ title }}
      <span class="delete" style="cursor: pointer" @click="handleDelete">
        <i class="el-icon-delete"></i>
        删除
      </span>
    </div>
    <el-row>
      <el-col :span="10">
        <div class="action-label">属性:</div>
        <el-select v-model="propName" class="use-property" size="small" placeholder="请选择属性" @change="handlePropChange">
          <el-option v-for="item in propOptions" :key="item.value" :label="item.label" size="small" :value="item.value"> </el-option>
        </el-select>
      </el-col>
      <el-col :span="14">
        <div class="action-label" style="margin-left: 30px">设置为:</div>
        <ValueCollector
          class="value-select"
          :model-value="val"
          @update:model-value="handleValueChange"
        />
      </el-col>
      <!-- <el-col :span="2" class="delete-button">
        <i 
          class="el-icon-delete" 
          style="cursor:pointer;"
          @click="handleDelete"
        >
        </i>
      </el-col> -->
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { IntelligentValueSource } from '../../schema';
import ValueCollector from '../valueCollector/index.vue';

defineOptions({ name: 'IntelligentActionItem' });

interface LogicProperty {
  name: string;
  description: string;
  propType: string;
  optionValue?: Array<{ description: string; value: unknown }>;
}

interface ActionValue {
  key: string;
  keyDefine: string;
  keyType: string;
  value: IntelligentValueSource;
  valueDefine: string;
}

const props = defineProps<{
  lf?: unknown;
  context?: unknown;
  current?: { getLogic: () => { props?: LogicProperty[] } };
  title?: string;
}>();
const model = defineModel<ActionValue>({
  default: () => ({
    key: '',
    keyDefine: '',
    keyType: '',
    value: { type: 'constant' },
    valueDefine: '',
  }),
});
const emit = defineEmits<{ change: [value: ActionValue]; delete: [] }>();
const propName = ref('');
const val = ref<IntelligentValueSource>({ type: 'constant' });

const propOptions = computed(() =>
  (props.current?.getLogic().props ?? []).map((item) => ({
    label: item.description,
    value: item.name,
  })),
);
const currentProp = computed(() =>
  (props.current?.getLogic().props ?? []).find((item) => item.name === propName.value),
);

watch(
  model,
  (value) => {
    propName.value = value.key;
    val.value = { ...value.value };
  },
  { immediate: true, deep: true },
);

function publish(value: ActionValue): void {
  model.value = value;
  emit('change', value);
}

function handlePropChange(value: string): void {
  propName.value = value;
  val.value = { type: 'constant' };
  const property = currentProp.value;
  if (!property) return;
  publish({
    key: property.name,
    keyDefine: property.description,
    keyType: property.propType,
    value: val.value,
    valueDefine: '',
  });
}

function handleValueChange(value: IntelligentValueSource): void {
  val.value = value;
  const property = currentProp.value;
  if (!property) return;
  publish({
    key: property.name,
    keyDefine: property.description,
    keyType: property.propType,
    value,
    valueDefine: '',
  });
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
