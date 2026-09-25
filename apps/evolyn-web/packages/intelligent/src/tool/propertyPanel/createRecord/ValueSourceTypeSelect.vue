<script setup lang="ts">
import { RiArrowDownSFill, RiCloseLine, RiEditBoxLine, RiText } from '@remixicon/vue';
import {
  type Component,
  computed,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
} from 'vue';

defineOptions({ name: 'IntelligentValueSourceTypeSelect' });

type AssignmentSourceMode = 'node-field' | 'custom' | 'empty';

const props = withDefaults(
  defineProps<{
    modelValue: AssignmentSourceMode;
    allowEmpty?: boolean;
  }>(),
  {
    allowEmpty: true,
  },
);
const emit = defineEmits<{ 'update:modelValue': [mode: AssignmentSourceMode] }>();
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);

const allOptions: readonly { value: AssignmentSourceMode; label: string; icon: Component }[] = [
  { value: 'node-field', label: '节点字段值', icon: RiText },
  { value: 'custom', label: '自定义', icon: RiEditBoxLine },
  { value: 'empty', label: '空值', icon: RiCloseLine },
];
const options = computed(() =>
  props.allowEmpty ? allOptions : allOptions.filter((item) => item.value !== 'empty'),
);

function selectedIcon(): Component {
  return options.value.find((item) => item.value === props.modelValue)?.icon ?? RiText;
}

function choose(mode: AssignmentSourceMode): void {
  emit('update:modelValue', mode);
  open.value = false;
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) open.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <div ref="rootRef" class="source-type-select">
    <button type="button" :aria-expanded="open" aria-label="选择值来源" @click="open = !open">
      <component :is="selectedIcon()" aria-hidden="true" />
      <RiArrowDownSFill aria-hidden="true" />
    </button>
    <div v-if="open" class="source-type-select__menu" role="listbox">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        role="option"
        :class="{ 'is-selected': option.value === modelValue }"
        :aria-selected="option.value === modelValue"
        @click="choose(option.value)"
      >
        <component :is="option.icon" aria-hidden="true" />
        <span>{{ option.label }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.source-type-select {
  position: relative;
  width: 106px;
  flex: 0 0 auto;

  > button {
    display: flex;
    width: 100%;
    height: 46px;
    padding: 0 13px;
    align-items: center;
    justify-content: space-between;
    color: #5b6678;
    background: #fff;
    border: 0;
    border-right: 1px solid #d8dee8;
    cursor: pointer;
  }
  svg {
    width: 20px;
    height: 20px;
  }

  &__menu {
    position: absolute;
    z-index: 90;
    top: calc(100% + 8px);
    left: -2px;
    display: grid;
    width: 178px;
    padding: 8px;
    background: #fff;
    border: 1px solid #e0e5ec;
    border-radius: 8px;
    box-shadow: 0 12px 28px rgb(31 43 61 / 18%);
    gap: 2px;

    button {
      display: flex;
      min-height: 40px;
      padding: 0 10px;
      align-items: center;
      gap: 10px;
      color: #273247;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;

      &:hover,
      &.is-selected {
        background: #e9f7f6;
      }
      svg {
        width: 20px;
        height: 20px;
      }
    }
  }
}
</style>
