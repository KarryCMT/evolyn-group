<template>
  <div class="action-panel-wrapper">
    <action-item
      v-for="(item, index) in actions"
      :key="`${item.key}-${index}`"
      v-model="actions[index]"
      :title="`行为${index + 1}`"
      :context="context"
      :current="current"
      :lf="lf"
      @change="handleActionChange($event, index)"
      @delete="handleActionDelete(index)"
      class="action-item"
    ></action-item>
    <el-link 
      type="primary" 
      :underline="false"
      class="add-button"
      @click="addAction"
    >
      <i class="el-icon-circle-plus-outline"></i>
      添加行为
    </el-link>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import type { IntelligentValueSource } from '../../schema';
import ActionItem from './actionItem.vue';

defineOptions({ name: 'IntelligentActionSetter' });

interface ActionValue {
  key: string;
  keyDefine: string;
  keyType: string;
  value: IntelligentValueSource;
  valueDefine: string;
}

const props = defineProps<{ lf?: unknown; context?: unknown; current?: unknown }>();
const model = defineModel<ActionValue[]>({ default: () => [] });
const emit = defineEmits<{ change: [value: ActionValue[]] }>();
const actions = ref<ActionValue[]>([]);

function createEmptyAction(): ActionValue {
  return {
    key: '',
    keyDefine: '',
    keyType: '',
    valueDefine: '',
    value: { type: 'constant' },
  };
}

watch(
  model,
  (value) => {
    actions.value = value.length
      ? value.map((item) => ({ ...item, value: { ...item.value } }))
      : [createEmptyAction()];
  },
  { immediate: true, deep: true },
);

function publish(): void {
  const value = actions.value.map((item) => ({ ...item, value: { ...item.value } }));
  model.value = value;
  emit('change', value);
}

function handleActionChange(value: ActionValue, index: number): void {
  actions.value[index] = value;
  publish();
}

function handleActionDelete(index: number): void {
  actions.value.splice(index, 1);
  publish();
}

function addAction(): void {
  if (actions.value.length >= 6) {
    ElMessage.warning('一个节点最多允许添加6个响应行为！');
    return;
  }
  actions.value.push(createEmptyAction());
  publish();
}
</script>

<style scoped lang="less">
.action-panel-wrapper {
  width: 100%;
}
.action-item {
  margin-bottom: 10px;
}
</style>
