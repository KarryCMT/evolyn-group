<script setup lang="ts">
import { RiArrowDownSLine, RiFileList3Line, RiSearchLine } from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import type { IntelligentFormOption } from '../../../schema';

defineOptions({ name: 'IntelligentTargetFormSelect' });

const props = defineProps<{
  options: readonly IntelligentFormOption[];
  modelValue: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [formCode: string];
}>();

const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const keyword = shallowRef('');
const selected = computed(() => props.options.find((item) => item.code === props.modelValue));
const visibleOptions = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return props.options;
  return props.options.filter((item) =>
    `${item.name}${item.code}`.toLocaleLowerCase().includes(normalized),
  );
});

function choose(option: IntelligentFormOption): void {
  if (option.disabled) return;
  emit('update:modelValue', option.code);
  open.value = false;
  keyword.value = '';
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) open.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <div ref="rootRef" class="target-form-select">
    <button
      type="button"
      class="target-form-select__control"
      :class="{ 'is-open': open }"
      :aria-expanded="open"
      aria-label="选择目标表单"
      @click="open = !open"
    >
      <span :class="{ 'is-placeholder': !selected }">
        {{ selected?.name ?? '请选择目标表单' }}
      </span>
      <span class="target-form-select__trailing">
        <RiFileList3Line aria-hidden="true" />
        <RiArrowDownSLine aria-hidden="true" />
      </span>
    </button>

    <div v-if="open" class="target-form-select__menu" role="dialog" aria-label="目标表单列表">
      <label class="target-form-select__search">
        <RiSearchLine aria-hidden="true" />
        <input v-model="keyword" placeholder="搜索" autocomplete="off">
      </label>
      <div class="target-form-select__list" role="listbox">
        <button
          v-for="option in visibleOptions"
          :key="option.code"
          type="button"
          role="option"
          :disabled="option.disabled"
          :aria-selected="option.code === modelValue"
          :class="{ 'is-selected': option.code === modelValue }"
          @click="choose(option)"
        >
          <RiFileList3Line aria-hidden="true" />
          <span>{{ option.name }}</span>
          <small v-if="option.disabled">未发布</small>
        </button>
        <p v-if="visibleOptions.length === 0">没有匹配的表单</p>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.target-form-select {
  position: relative;

  &__control {
    display: flex;
    width: 100%;
    height: 58px;
    padding: 0 16px;
    align-items: center;
    justify-content: space-between;
    color: #172033;
    background: #fff;
    border: 1px solid #d6dce6;
    border-radius: 8px;
    cursor: pointer;
    font: inherit;
    font-size: 16px;
    text-align: left;

    &.is-open,
    &:focus-visible { border-color: #12b8ad; outline: none; }
    .is-placeholder { color: #9aa3af; }
  }

  &__trailing { display: inline-flex; align-items: center; gap: 8px; color: #536075; }
  &__trailing svg { width: 19px; height: 19px; }

  &__menu {
    position: absolute;
    z-index: 80;
    top: calc(100% + 8px);
    left: 0;
    width: 100%;
    background: #fff;
    border: 1px solid #e1e5eb;
    border-radius: 9px;
    box-shadow: 0 14px 34px rgb(31 43 61 / 17%);
  }

  &__search {
    display: flex;
    height: 48px;
    padding: 0 15px;
    align-items: center;
    gap: 10px;
    color: #536075;
    border-bottom: 1px solid #dfe4eb;
  }
  &__search svg { width: 21px; height: 21px; }
  &__search input { min-width: 0; flex: 1; border: 0; outline: 0; font: inherit; font-size: 15px; }

  &__list {
    display: grid;
    max-height: 390px;
    padding: 9px 14px 12px;
    overflow-y: auto;
    gap: 3px;

    button {
      display: grid;
      min-height: 43px;
      padding: 0 13px;
      align-items: center;
      color: #273247;
      background: transparent;
      border: 0;
      border-radius: 8px;
      cursor: pointer;
      font: inherit;
      grid-template-columns: 22px minmax(0, 1fr) auto;
      gap: 9px;
      text-align: left;

      &:hover,
      &.is-selected { background: #e9f7f6; }
      &:disabled { color: #9aa3af; cursor: not-allowed; }
      svg { width: 19px; height: 19px; color: #25aee9; }
      small { color: #9aa3af; }
    }

    p { margin: 28px 0; color: #9aa3af; text-align: center; }
  }
}
</style>
