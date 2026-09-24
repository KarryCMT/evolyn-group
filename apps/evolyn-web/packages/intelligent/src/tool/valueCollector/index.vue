<script setup lang="ts">
import { computed } from 'vue';
import type { IntelligentValueSource, IntelligentValueSourceType } from '../../schema';
import { mockFormFields } from '../../mock/nodeTemplates';
import IntelligentFieldSelector from '../fieldSelector/index.vue';

defineOptions({ name: 'IntelligentValueCollector' });

const model = defineModel<IntelligentValueSource>({
  default: () => ({ type: 'constant' }),
});

const sourceType = computed({
  get: () => model.value.type,
  set: (type: IntelligentValueSourceType) => {
    model.value = { type };
  },
});

function updateField(field: string): void {
  model.value = { ...model.value, field };
}

function updateValue(event: Event): void {
  model.value = { ...model.value, value: (event.target as HTMLInputElement).value };
}
</script>

<template>
  <div class="intelligent-value-collector">
    <select v-model="sourceType">
      <option value="constant">固定值</option>
      <option value="trigger-field">触发数据字段</option>
      <option value="node-output">前序节点输出</option>
    </select>
    <input
      v-if="sourceType === 'constant'"
      :value="model.value ?? ''"
      placeholder="请输入固定值"
      @input="updateValue"
    >
    <IntelligentFieldSelector
      v-else-if="sourceType === 'trigger-field'"
      :model-value="model.field ?? ''"
      :options="mockFormFields"
      @update:model-value="updateField"
    />
    <input
      v-else
      :value="model.nodeId ?? ''"
      placeholder="请输入前序节点 ID"
      @input="model = { ...model, nodeId: ($event.target as HTMLInputElement).value }"
    >
  </div>
</template>

<style scoped lang="scss">
.intelligent-value-collector {
  display: grid;
  grid-template-columns: 128px minmax(0, 1fr);
  gap: 8px;

  select,
  input {
    min-width: 0;
    height: 38px;
    padding: 0 10px;
    color: #172033;
    background: #fff;
    border: 1px solid #d8dee8;
    border-radius: 6px;
    outline: none;

    &:focus { border-color: #00afa2; }
  }
}
</style>
