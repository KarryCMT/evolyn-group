<script setup lang="ts">
import { RiSearchLine } from '@remixicon/vue';
import { ElIcon } from 'element-plus';
import { computed, shallowRef } from 'vue';
import type { FormulaEditorField } from '../../formula';

const props = defineProps<{
  fields: readonly (FormulaEditorField & {
    listKey?: string;
    disabled?: boolean;
    disabledReason?: string;
  })[];
}>();

const emit = defineEmits<{
  insert: [field: FormulaEditorField];
}>();

const keyword = shallowRef('');
const filteredFields = computed(() => {
  const query = keyword.value.trim().toLocaleLowerCase();
  return query
    ? props.fields.filter(
        (field) =>
          field.label.toLocaleLowerCase().includes(query) ||
          field.widgetName.toLocaleLowerCase().includes(query),
      )
    : props.fields;
});
</script>

<template>
  <section class="formula-variable-panel" aria-label="公式变量">
    <label class="formula-variable-panel__search">
      <el-icon><RiSearchLine /></el-icon>
      <input v-model="keyword" type="search" placeholder="搜索变量" />
    </label>
    <div class="formula-variable-panel__scope">当前表单</div>
    <div class="formula-variable-panel__list">
      <button
        v-for="field in filteredFields"
        :key="field.listKey ?? field.widgetName"
        type="button"
        :disabled="field.disabled"
        :title="field.disabledReason"
        class="formula-variable-panel__field"
        @click="emit('insert', field)"
      >
        <span>{{ field.label }}</span>
        <small>{{ field.displayType ?? field.valueType }}</small>
      </button>
      <p v-if="filteredFields.length === 0" class="formula-variable-panel__empty">未找到变量</p>
    </div>
  </section>
</template>

<style scoped lang="scss">
.formula-variable-panel {
  min-width: 0;
  border-right: 1px solid var(--el-border-color-lighter);
}

.formula-variable-panel__search {
  display: flex;
  gap: 8px;
  height: 46px;
  padding: 0 14px;
  align-items: center;
  color: var(--el-text-color-placeholder);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.formula-variable-panel__search input {
  width: 100%;
  color: var(--el-text-color-primary);
  font: inherit;
  background: transparent;
  border: 0;
  outline: 0;
}

.formula-variable-panel__scope {
  padding: 10px 16px 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.formula-variable-panel__list {
  max-height: 230px;
  padding: 0 9px 10px;
  overflow: auto;
}

.formula-variable-panel__field {
  display: flex;
  width: 100%;
  min-height: 40px;
  padding: 0 10px;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 7px;
}

.formula-variable-panel__field:hover,
.formula-variable-panel__field:focus-visible {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  outline: none;
}

.formula-variable-panel__field:disabled {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
  background: transparent;
}

.formula-variable-panel__field small {
  padding: 2px 7px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 999px;
}

.formula-variable-panel__empty {
  padding: 18px 8px;
  margin: 0;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
