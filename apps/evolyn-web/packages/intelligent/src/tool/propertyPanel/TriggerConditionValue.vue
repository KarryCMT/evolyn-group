<script setup lang="ts">
import { RiAddLine, RiCloseLine } from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import type { IntelligentFormFieldOption } from '../../mock/nodeTemplates';

defineOptions({ name: 'TriggerConditionValue' });

const props = defineProps<{
  field?: IntelligentFormFieldOption;
  disabled?: boolean;
}>();

const values = defineModel<string[]>({ default: () => [] });
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const input = shallowRef('');
const menuOpen = shallowRef(false);
// 成员与选项字段使用受控候选值，其余字段允许直接录入并形成标签。
const selectableValues = computed(() => {
  if (props.field?.kind === 'member') return ['张三', '李四', '王五'];
  return [...(props.field?.choices ?? [])];
});
const buttonLabel = computed(() => (props.field?.kind === 'member' ? '选择成员' : '添加筛选值'));

function addValue(value = input.value): void {
  const normalized = value.trim();
  if (!normalized || values.value.includes(normalized)) return;
  values.value = [...values.value, normalized];
  input.value = '';
  menuOpen.value = false;
}

function removeValue(value: string): void {
  values.value = values.value.filter((item) => item !== value);
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  addValue();
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) menuOpen.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <div ref="rootRef" class="trigger-condition-value" :class="{ 'is-disabled': disabled }">
    <span v-for="value in values" :key="value" class="trigger-condition-value__tag">
      {{ value }}
      <button type="button" :aria-label="`移除${value}`" @click="removeValue(value)"><RiCloseLine /></button>
    </span>

    <template v-if="!disabled">
      <button
        v-if="selectableValues.length > 0"
        type="button"
        class="trigger-condition-value__add"
        :aria-expanded="menuOpen"
        @click="menuOpen = !menuOpen"
      >
        <RiAddLine />{{ buttonLabel }}
      </button>
      <input
        v-else
        v-model="input"
        :placeholder="buttonLabel"
        @keydown="handleKeydown"
        @blur="addValue()"
      >
    </template>

    <div v-if="menuOpen" class="trigger-condition-value__menu">
      <button
        v-for="option in selectableValues"
        :key="option"
        type="button"
        :disabled="values.includes(option)"
        @click="addValue(option)"
      >
        {{ option }}
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.trigger-condition-value {
  position: relative;
  display: flex;
  min-height: 42px;
  padding: 4px 8px;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  background: #fff;
  border: 1px dashed #d6dce6;
  border-radius: 6px;

  &.is-disabled { background: #f5f7fa; }

  > input {
    min-width: 100px;
    height: 30px;
    flex: 1;
    border: 0;
    outline: 0;
    font: inherit;
  }

  &__add {
    display: inline-flex;
    padding: 5px 4px;
    align-items: center;
    gap: 4px;
    color: #526077;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;

    svg { width: 18px; height: 18px; }
  }

  &__tag {
    display: inline-flex;
    min-height: 28px;
    padding: 0 6px 0 9px;
    align-items: center;
    gap: 4px;
    color: #344157;
    background: #f0f3f7;
    border-radius: 4px;
    font-size: 13px;

    button { display: inline-flex; padding: 2px; background: transparent; border: 0; cursor: pointer; }
    svg { width: 14px; height: 14px; }
  }

  &__menu {
    position: absolute;
    z-index: 36;
    top: calc(100% + 7px);
    left: 0;
    display: grid;
    min-width: 190px;
    padding: 7px;
    background: #fff;
    border: 1px solid #e1e5eb;
    border-radius: 8px;
    box-shadow: 0 10px 26px rgb(31 43 61 / 16%);

    button {
      padding: 8px 10px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 5px;
      cursor: pointer;
      text-align: left;

      &:hover:not(:disabled) { background: #eaf8f6; }
      &:disabled { color: #a5adba; cursor: not-allowed; }
    }
  }
}
</style>
