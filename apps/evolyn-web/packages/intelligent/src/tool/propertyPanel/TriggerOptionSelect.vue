<script setup lang="ts">
import { RiArrowDownSLine } from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';

defineOptions({ name: 'TriggerOptionSelect' });

const props = defineProps<{
  options: readonly { label: string; value: string }[];
  controlLabel: string;
  placeholder?: string;
}>();

const model = defineModel<string>({ default: '' });
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const selectedLabel = computed(
  () => props.options.find((option) => option.value === model.value)?.label ?? props.placeholder ?? '请选择',
);

function choose(value: string): void {
  model.value = value;
  open.value = false;
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) open.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <div ref="rootRef" class="trigger-option-select">
    <button
      type="button"
      class="trigger-option-select__button"
      :aria-label="controlLabel"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span>{{ selectedLabel }}</span>
      <RiArrowDownSLine aria-hidden="true" />
    </button>
    <div v-if="open" class="trigger-option-select__menu" role="listbox">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        :class="{ 'is-selected': option.value === model }"
        role="option"
        :aria-selected="option.value === model"
        @click="choose(option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.trigger-option-select {
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
    text-align: left;

    &:focus-visible { border-color: #00afa2; outline: none; }
    svg { width: 20px; height: 20px; flex: 0 0 auto; }
  }

  &__menu {
    position: absolute;
    z-index: 30;
    top: calc(100% + 8px);
    left: 0;
    display: grid;
    width: max(100%, 210px);
    padding: 8px;
    background: #fff;
    border: 1px solid #e3e7ed;
    border-radius: 8px;
    box-shadow: 0 10px 28px rgb(30 43 62 / 16%);
    gap: 2px;

    button {
      padding: 9px 12px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover,
      &.is-selected { background: #eaf8f6; }
    }
  }
}
</style>
