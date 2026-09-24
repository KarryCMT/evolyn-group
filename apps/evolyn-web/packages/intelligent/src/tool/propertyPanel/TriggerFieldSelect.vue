<script setup lang="ts">
import {
  RiArrowDownSLine,
  RiBuildingLine,
  RiCalendarLine,
  RiMapPinLine,
  RiPhoneLine,
  RiSearchLine,
  RiText,
  RiUserLine,
} from '@remixicon/vue';
import { type Component, computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import type { IntelligentFormFieldKind, IntelligentFormFieldOption } from '../../mock/nodeTemplates';

defineOptions({ name: 'TriggerFieldSelect' });

const props = defineProps<{
  options: readonly IntelligentFormFieldOption[];
  controlLabel: string;
  placeholder?: string;
}>();

const model = defineModel<string>({ default: '' });
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const keyword = shallowRef('');
const selected = computed(() => props.options.find((option) => option.value === model.value));
// 字段量较大时在前端即时筛选，保留与截图一致的下拉搜索体验。
const visibleOptions = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return props.options;
  return props.options.filter((option) => option.label.toLocaleLowerCase().includes(normalized));
});

const fieldIcons: Record<IntelligentFormFieldKind, Component> = {
  member: RiUserLine,
  text: RiText,
  department: RiBuildingLine,
  choice: RiText,
  date: RiCalendarLine,
  address: RiMapPinLine,
};

function fieldIcon(option?: IntelligentFormFieldOption): Component {
  if (!option) return RiText;
  if (option.value === 'contact_phone') return RiPhoneLine;
  return fieldIcons[option.kind];
}

function choose(value: string): void {
  model.value = value;
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
  <div ref="rootRef" class="trigger-field-select">
    <button
      type="button"
      class="trigger-field-select__button"
      :aria-label="controlLabel"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span :class="{ 'is-placeholder': !selected }">
        <component :is="fieldIcon(selected)" v-if="selected" aria-hidden="true" />
        {{ selected?.label ?? placeholder ?? '请选择字段' }}
      </span>
      <RiArrowDownSLine aria-hidden="true" />
    </button>

    <div v-if="open" class="trigger-field-select__menu" role="dialog" aria-label="选择字段">
      <label class="trigger-field-select__search">
        <RiSearchLine aria-hidden="true" />
        <input v-model="keyword" placeholder="搜索" autocomplete="off">
      </label>
      <div class="trigger-field-select__list" role="listbox">
        <button
          v-for="option in visibleOptions"
          :key="option.value"
          type="button"
          :class="{ 'is-selected': option.value === model }"
          role="option"
          :aria-selected="option.value === model"
          @click="choose(option.value)"
        >
          <component :is="fieldIcon(option)" aria-hidden="true" />
          <span>{{ option.label }}</span>
        </button>
        <p v-if="visibleOptions.length === 0">没有匹配字段</p>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.trigger-field-select {
  position: relative;
  min-width: 0;

  &__button {
    display: flex;
    width: 100%;
    height: 42px;
    padding: 0 13px;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    color: #263247;
    background: #fff;
    border: 1px solid #d8dee8;
    border-radius: 6px;
    cursor: pointer;
    font: inherit;

    > span { display: inline-flex; min-width: 0; align-items: center; gap: 8px; }
    > span.is-placeholder { color: #a2aab6; }
    svg { width: 19px; height: 19px; flex: 0 0 auto; }
  }

  &__menu {
    position: absolute;
    z-index: 35;
    top: calc(100% + 8px);
    left: 0;
    width: min(330px, 80vw);
    padding: 8px;
    background: #fff;
    border: 1px solid #e2e6ec;
    border-radius: 8px;
    box-shadow: 0 12px 30px rgb(31 43 61 / 18%);
  }

  &__search {
    display: flex;
    height: 38px;
    padding: 0 10px;
    align-items: center;
    gap: 8px;
    color: #667286;
    border-bottom: 1px solid #e6e9ef;

    svg { width: 19px; height: 19px; }
    input { min-width: 0; flex: 1; border: 0; outline: 0; font: inherit; }
  }

  &__list {
    display: grid;
    max-height: 330px;
    padding-top: 6px;
    overflow-y: auto;
    gap: 2px;

    button {
      display: flex;
      min-height: 38px;
      padding: 0 10px;
      align-items: center;
      gap: 10px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover,
      &.is-selected { background: #edf1f5; }
      svg { width: 18px; height: 18px; color: #536075; }
    }

    p { margin: 18px 0; color: #98a1ae; text-align: center; }
  }
}
</style>
