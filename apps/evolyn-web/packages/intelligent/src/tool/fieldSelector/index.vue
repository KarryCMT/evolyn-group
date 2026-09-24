<script setup lang="ts">
defineOptions({ name: 'IntelligentFieldSelector' });

defineProps<{
  options: readonly { value: string; label: string }[];
  placeholder?: string;
}>();

const model = defineModel<string>({ default: '' });
</script>

<template>
  <select v-model="model" class="intelligent-field-selector">
    <option value="" disabled>{{ placeholder ?? '请选择字段' }}</option>
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>

<style scoped>
.intelligent-field-selector {
  width: 100%;
  height: 38px;
  padding: 0 10px;
  color: #172033;
  background: #fff;
  border: 1px solid #d8dee8;
  border-radius: 6px;
  outline: none;
}

.intelligent-field-selector:focus { border-color: #00afa2; }
</style>
