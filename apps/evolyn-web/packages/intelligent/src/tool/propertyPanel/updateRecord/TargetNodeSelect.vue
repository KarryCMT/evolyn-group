<script setup lang="ts">
import { RiArrowDownSLine, RiFileList3Line, RiSearchLine } from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';

defineOptions({ name: 'UpdateRecordTargetNodeSelect' });

interface UpdateRecordTargetNodeOption {
  id: string;
  name: string;
  formName: string;
}

const props = defineProps<{
  modelValue: string;
  options: readonly UpdateRecordTargetNodeOption[];
}>();
const emit = defineEmits<{ 'update:modelValue': [nodeId: string] }>();
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const keyword = shallowRef('');
const selected = computed(() => props.options.find((option) => option.id === props.modelValue));
const visibleOptions = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  return props.options.filter(
    (option) =>
      !normalized || `${option.name}${option.formName}`.toLocaleLowerCase().includes(normalized),
  );
});

function choose(nodeId: string): void {
  emit('update:modelValue', nodeId);
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
  <div ref="rootRef" class="target-node-select">
    <button
      type="button"
      class="target-node-select__control"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span :class="{ 'is-placeholder': !selected }">
        {{ selected ? `${selected.name} · ${selected.formName}` : '请选择前序数据节点' }}
      </span>
      <span><RiFileList3Line /><RiArrowDownSLine /></span>
    </button>
    <div v-if="open" class="target-node-select__menu">
      <label><RiSearchLine /><input v-model="keyword" placeholder="搜索" autocomplete="off" /></label>
      <div>
        <button
          v-for="option in visibleOptions"
          :key="option.id"
          type="button"
          @click="choose(option.id)"
        >
          <RiFileList3Line />
          <span><strong>{{ option.name }}</strong><small>{{ option.formName }}</small></span>
        </button>
        <p v-if="visibleOptions.length === 0">没有可修改的前序数据节点</p>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.target-node-select {
  position: relative;
  min-width: 0;

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

    .is-placeholder {
      color: #9aa3af;
    }
    > span:last-child {
      display: inline-flex;
      gap: 8px;
      color: #536075;
    }
    svg {
      width: 19px;
      height: 19px;
    }
  }

  &__menu {
    position: absolute;
    z-index: 92;
    top: calc(100% + 8px);
    left: 0;
    width: 100%;
    background: #fff;
    border: 1px solid #e1e5eb;
    border-radius: 9px;
    box-shadow: 0 14px 34px rgb(31 43 61 / 17%);

    > label {
      display: flex;
      height: 48px;
      padding: 0 15px;
      align-items: center;
      gap: 10px;
      border-bottom: 1px solid #dfe4eb;
    }
    > label svg {
      width: 21px;
      height: 21px;
    }
    > label input {
      min-width: 0;
      flex: 1;
      border: 0;
      outline: 0;
      font: inherit;
    }
    > div {
      display: grid;
      max-height: 320px;
      padding: 9px;
      overflow-y: auto;
      gap: 2px;
    }
    > div > button {
      display: grid;
      min-height: 48px;
      padding: 6px 11px;
      align-items: center;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 7px;
      cursor: pointer;
      font: inherit;
      grid-template-columns: 22px 1fr;
      gap: 10px;
      text-align: left;
    }
    > div > button:hover {
      background: #e9f7f6;
    }
    > div > button svg {
      width: 19px;
      height: 19px;
      color: #25aee9;
    }
    > div > button span {
      display: grid;
      gap: 2px;
    }
    > div > button strong {
      font-weight: 500;
    }
    > div > button small {
      color: #8d97a5;
    }
    p {
      margin: 24px 0;
      color: #9aa3af;
      text-align: center;
    }
  }
}
</style>
