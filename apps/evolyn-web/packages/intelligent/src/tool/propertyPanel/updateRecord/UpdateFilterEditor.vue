<script setup lang="ts">
import { RiAddLine } from '@remixicon/vue';
import {
  type IntelligentActorOption,
  type IntelligentFieldOption,
  type IntelligentSourceFieldGroup,
  type IntelligentUpdateFilter,
  type IntelligentUpdateMatchMode,
  createIntelligentUpdateFilter,
} from '../../../schema';
import TriggerOptionSelect from '../TriggerOptionSelect.vue';
import UpdateFilterRow from './UpdateFilterRow.vue';

defineOptions({ name: 'UpdateRecordFilterEditor' });

const props = defineProps<{
  filters: readonly IntelligentUpdateFilter[];
  mode: IntelligentUpdateMatchMode;
  fields: readonly IntelligentFieldOption[];
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
  createWhenNoMatch: boolean;
}>();
const emit = defineEmits<{
  'update:filters': [filters: IntelligentUpdateFilter[]];
  'update:mode': [mode: IntelligentUpdateMatchMode];
  'update:createWhenNoMatch': [enabled: boolean];
}>();
const modeOptions = [
  { value: 'all', label: '所有' },
  { value: 'any', label: '任一' },
] as const;

function addFilter(): void {
  emit('update:filters', [...props.filters, createIntelligentUpdateFilter()]);
}

function updateFilter(filter: IntelligentUpdateFilter): void {
  emit(
    'update:filters',
    props.filters.map((item) => (item.id === filter.id ? filter : item)),
  );
}

function removeFilter(filterId: string): void {
  emit(
    'update:filters',
    props.filters.filter((item) => item.id !== filterId),
  );
}
</script>

<template>
  <section class="update-filter-editor">
    <h3><b>*</b>筛选出要修改的数据</h3>
    <div class="update-filter-editor__summary">
      <span>满足</span>
      <TriggerOptionSelect
        :model-value="mode"
        :options="modeOptions"
        control-label="条件匹配方式"
        @update:model-value="emit('update:mode', $event as IntelligentUpdateMatchMode)"
      />
      <span>条件的数据</span>
    </div>
    <button type="button" class="update-filter-editor__add" @click="addFilter">
      <RiAddLine />添加条件
    </button>
    <div class="update-filter-editor__rows">
      <UpdateFilterRow
        v-for="filter in filters"
        :key="filter.id"
        :filter="filter"
        :fields="fields"
        :source-groups="sourceGroups"
        :members="members"
        :departments="departments"
        @change="updateFilter"
        @remove="removeFilter(filter.id)"
      />
    </div>
    <p v-if="filters.length === 0" class="update-filter-editor__error">请设置条件</p>
    <label class="update-filter-editor__upsert">
      <input
        type="checkbox"
        :checked="createWhenNoMatch"
        @change="emit('update:createWhenNoMatch', ($event.target as HTMLInputElement).checked)"
      />
      <span>没有可修改的数据时，向对应表单新增数据</span>
    </label>
  </section>
</template>

<style scoped lang="scss">
.update-filter-editor {
  padding: 28px 32px 30px;

  h3 {
    margin: 0 0 20px;
    font-size: 17px;
  }
  h3 b {
    margin-right: 2px;
    color: #ef5252;
  }
  &__summary {
    display: flex;
    min-height: 38px;
    align-items: center;
    gap: 9px;
    color: #727d8e;

    :deep(.trigger-option-select) {
      width: 90px;
    }
    :deep(.trigger-option-select__button) {
      height: 36px;
      padding: 0 8px;
      border-color: transparent;
      background: #f4f5f7;
    }
  }
  &__add {
    display: inline-flex;
    margin: 12px 0 10px;
    padding: 3px 0;
    align-items: center;
    gap: 5px;
    color: #00a99d;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
    font-size: 16px;

    svg {
      width: 22px;
      height: 22px;
    }
  }
  &__rows {
    display: grid;
    gap: 12px;
  }
  &__error {
    margin: 6px 0 0;
    color: #ef5252;
    font-size: 13px;
  }
  &__upsert {
    display: flex;
    margin-top: 26px;
    align-items: center;
    gap: 10px;
    color: #263247;
    cursor: pointer;
    font-size: 15px;

    input {
      width: 18px;
      height: 18px;
      accent-color: #00afa2;
    }
  }
}
</style>
