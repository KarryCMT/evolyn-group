<script setup lang="ts">
import { RiSearchLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import { formEventFieldTypeLabel, type FormEventFieldOption } from './frontend-events';

const props = withDefaults(
  defineProps<{
    fields: readonly FormEventFieldOption[];
    exclude?: readonly string[];
    placeholder?: string;
  }>(),
  { exclude: () => [], placeholder: '搜索字段' },
);

const emit = defineEmits<{ select: [field: FormEventFieldOption] }>();
const keyword = shallowRef('');
const visibleFields = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  const excluded = new Set(props.exclude);
  return props.fields.filter(
    (field) =>
      !excluded.has(field.key) &&
      (!normalized || `${field.group ?? ''}${field.label}${field.key}`.toLocaleLowerCase().includes(normalized)),
  );
});

function select(field: FormEventFieldOption): void {
  emit('select', field);
  keyword.value = '';
}
</script>

<template>
  <div class="event-field-picker" role="listbox" aria-label="表单字段">
    <label class="event-field-picker__search">
      <RiSearchLine aria-hidden="true" />
      <input v-model="keyword" :placeholder="props.placeholder" type="search" />
    </label>
    <div class="event-field-picker__list">
      <button
        v-for="field in visibleFields"
        :key="`${field.group ?? 'root'}:${field.key}`"
        class="event-field-picker__option"
        type="button"
        role="option"
        @click="select(field)"
      >
        <span class="event-field-picker__label">
          <small v-if="field.group">{{ field.group }} · </small>{{ field.label || field.key }}
        </span>
        <span class="event-field-picker__type">{{ formEventFieldTypeLabel(field.type) }}</span>
      </button>
      <p v-if="visibleFields.length === 0" class="event-field-picker__empty">未找到可用字段</p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.event-field-picker {
  min-width: 300px;
  overflow: hidden;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-light);

  &__search { display: flex; gap: 8px; align-items: center; padding: 10px 12px; border-bottom: 1px solid var(--el-border-color-lighter); color: var(--el-text-color-secondary); }
  &__search input { width: 100%; color: var(--el-text-color-primary); font: inherit; border: 0; outline: 0; background: transparent; }
  &__list { max-height: 260px; overflow-y: auto; padding: 6px; }
  &__option { display: flex; width: 100%; min-height: 36px; align-items: center; justify-content: space-between; gap: 16px; padding: 7px 9px; text-align: left; color: var(--el-text-color-primary); cursor: pointer; background: transparent; border: 0; border-radius: 6px; }
  &__option:hover, &__option:focus-visible { background: var(--el-fill-color-light); outline: 0; }
  &__label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__label small { color: var(--el-text-color-secondary); }
  &__type { flex: 0 0 auto; padding: 1px 6px; color: var(--el-color-primary); font-size: 12px; background: var(--el-color-primary-light-9); border-radius: 999px; }
  &__empty { padding: 16px 8px; margin: 0; color: var(--el-text-color-secondary); text-align: center; }
}
</style>
