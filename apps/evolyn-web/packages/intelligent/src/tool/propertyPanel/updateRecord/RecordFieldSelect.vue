<script setup lang="ts">
import {
  RiArrowDownSLine,
  RiBuildingLine,
  RiCalendarLine,
  RiMapPinLine,
  RiSearchLine,
  RiText,
  RiUserLine,
} from '@remixicon/vue';
import {
  type Component,
  computed,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
} from 'vue';
import type { IntelligentFieldOption, IntelligentFieldValueKind } from '../../../schema';

defineOptions({ name: 'UpdateRecordFieldSelect' });

const props = defineProps<{
  modelValue: string;
  options: readonly IntelligentFieldOption[];
  placeholder?: string;
  disabledFieldIds?: readonly string[];
}>();
const emit = defineEmits<{
  'update:modelValue': [fieldId: string];
}>();

const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const keyword = shallowRef('');
const selected = computed(() => props.options.find((field) => field.fieldId === props.modelValue));
const disabledIds = computed(() => new Set(props.disabledFieldIds ?? []));
const visibleOptions = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  return props.options.filter(
    (field) =>
      !normalized || `${field.label}${field.widgetName}`.toLocaleLowerCase().includes(normalized),
  );
});

const fieldIcons: Partial<Record<IntelligentFieldValueKind, Component>> = {
  member: RiUserLine,
  members: RiUserLine,
  department: RiBuildingLine,
  departments: RiBuildingLine,
  date: RiCalendarLine,
  address: RiMapPinLine,
};

function icon(field?: IntelligentFieldOption): Component {
  return field ? (fieldIcons[field.valueKind] ?? RiText) : RiText;
}

function choose(field: IntelligentFieldOption): void {
  if (disabledIds.value.has(field.fieldId) && field.fieldId !== props.modelValue) return;
  emit('update:modelValue', field.fieldId);
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
  <div ref="rootRef" class="record-field-select">
    <button
      type="button"
      class="record-field-select__control"
      :class="{ 'is-open': open, 'is-invalid': !selected }"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span :class="{ 'is-placeholder': !selected }">
        <component :is="icon(selected)" v-if="selected" aria-hidden="true" />
        {{ selected?.label ?? placeholder ?? '请选择字段' }}
      </span>
      <RiArrowDownSLine aria-hidden="true" />
    </button>

    <div v-if="open" class="record-field-select__menu" role="dialog" aria-label="选择字段">
      <label class="record-field-select__search">
        <RiSearchLine aria-hidden="true" />
        <input v-model="keyword" placeholder="搜索" autocomplete="off" />
      </label>
      <div class="record-field-select__list" role="listbox">
        <button
          v-for="field in visibleOptions"
          :key="field.fieldId"
          type="button"
          :disabled="disabledIds.has(field.fieldId) && field.fieldId !== modelValue"
          :class="{ 'is-selected': field.fieldId === modelValue }"
          @click="choose(field)"
        >
          <component :is="icon(field)" aria-hidden="true" />
          <span>{{ field.label }}</span>
        </button>
        <p v-if="visibleOptions.length === 0">没有匹配字段</p>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.record-field-select {
  position: relative;
  min-width: 0;

  &__control {
    display: flex;
    width: 100%;
    height: 46px;
    padding: 0 14px;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    color: #263247;
    background: #fff;
    border: 1px solid #d8dee8;
    border-radius: 7px;
    cursor: pointer;
    font: inherit;
    text-align: left;

    &.is-open {
      border-color: #11b8ad;
    }
    &.is-invalid {
      border-color: #ef5858;
    }
    > span {
      display: inline-flex;
      min-width: 0;
      align-items: center;
      gap: 9px;
    }
    > span.is-placeholder {
      color: #a0a8b5;
    }
    svg {
      width: 20px;
      height: 20px;
      flex: 0 0 auto;
    }
  }

  &__menu {
    position: absolute;
    z-index: 94;
    top: calc(100% + 8px);
    left: 0;
    width: max(100%, 320px);
    background: #fff;
    border: 1px solid #e1e5eb;
    border-radius: 9px;
    box-shadow: 0 14px 34px rgb(31 43 61 / 17%);
  }

  &__search {
    display: flex;
    height: 48px;
    padding: 0 14px;
    align-items: center;
    gap: 10px;
    color: #536075;
    border-bottom: 1px solid #e1e5eb;

    svg {
      width: 21px;
      height: 21px;
    }
    input {
      min-width: 0;
      flex: 1;
      border: 0;
      outline: 0;
      font: inherit;
    }
  }

  &__list {
    display: grid;
    max-height: 340px;
    padding: 9px;
    overflow-y: auto;
    gap: 2px;

    button {
      display: flex;
      min-height: 40px;
      padding: 0 11px;
      align-items: center;
      gap: 10px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 7px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover,
      &.is-selected {
        background: #edf1f5;
      }
      &:disabled {
        color: #b0b7c2;
        cursor: not-allowed;
      }
      svg {
        width: 19px;
        height: 19px;
        color: #536075;
      }
    }

    p {
      margin: 24px 0;
      color: #9aa3af;
      text-align: center;
    }
  }
}
</style>
