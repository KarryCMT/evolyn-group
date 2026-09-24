<script setup lang="ts">
import { RiAddLine } from '@remixicon/vue';
import { computed } from 'vue';
import type {
  IntelligentActorOption,
  IntelligentFieldOption,
  IntelligentJsonValue,
} from '../../../schema';

defineOptions({ name: 'IntelligentCustomFieldValueEditor' });

const props = defineProps<{
  field: IntelligentFieldOption;
  modelValue: IntelligentJsonValue;
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
}>();

const emit = defineEmits<{
  'update:modelValue': [value: IntelligentJsonValue];
}>();

const isMultiple = computed(() =>
  ['multi-choice', 'members', 'departments'].includes(props.field.valueKind),
);
const actorOptions = computed(() =>
  props.field.valueKind === 'member' || props.field.valueKind === 'members'
    ? (props.members ?? [])
    : (props.departments ?? []),
);

function inputText(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value);
}

function inputNumber(event: Event): void {
  const raw = (event.target as HTMLInputElement).value;
  // decimal/money/percent 使用 canonical decimal string，只有 number 使用 JS number。
  emit('update:modelValue', raw === '' ? null : props.field.widgetType === 'number' ? Number(raw) : raw);
}

function selectOne(event: Event): void {
  emit('update:modelValue', (event.target as HTMLSelectElement).value);
}

function selectMany(event: Event): void {
  const values = [...(event.target as HTMLSelectElement).selectedOptions].map(
    (option) => option.value,
  );
  emit('update:modelValue', values);
}
</script>

<template>
  <div class="custom-value-editor">
    <input
      v-if="field.valueKind === 'text' || field.valueKind === 'address'"
      :value="typeof modelValue === 'string' ? modelValue : ''"
      :placeholder="field.valueKind === 'address' ? '请输入地址' : '请输入内容'"
      @input="inputText"
    >
    <input
      v-else-if="field.valueKind === 'number'"
      type="number"
      :value="typeof modelValue === 'number' || typeof modelValue === 'string' ? modelValue : ''"
      placeholder="请输入数值"
      @input="inputNumber"
    >
    <input
      v-else-if="field.valueKind === 'date'"
      type="datetime-local"
      :value="typeof modelValue === 'string' ? modelValue : ''"
      @input="inputText"
    >
    <select
      v-else-if="field.valueKind === 'choice'"
      :value="typeof modelValue === 'string' ? modelValue : ''"
      @change="selectOne"
    >
      <option value="">请选择</option>
      <option v-for="choice in field.choices" :key="choice.value" :value="choice.value">
        {{ choice.label }}
      </option>
    </select>
    <select
      v-else-if="field.valueKind === 'multi-choice'"
      multiple
      :value="Array.isArray(modelValue) ? modelValue : []"
      @change="selectMany"
    >
      <option v-for="choice in field.choices" :key="choice.value" :value="choice.value">
        {{ choice.label }}
      </option>
    </select>
    <select
      v-else-if="['member', 'members', 'department', 'departments'].includes(field.valueKind)"
      :multiple="isMultiple"
      :value="isMultiple ? (Array.isArray(modelValue) ? modelValue : []) : String(modelValue ?? '')"
      @change="isMultiple ? selectMany($event) : selectOne($event)"
    >
      <option v-if="!isMultiple" value="">请选择</option>
      <option v-for="option in actorOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
    <button v-else type="button" class="custom-value-editor__unsupported" disabled>
      <RiAddLine />暂不支持自定义值
    </button>
  </div>
</template>

<style scoped lang="scss">
.custom-value-editor {
  min-width: 0;
  flex: 1;

  input,
  select,
  &__unsupported {
    width: 100%;
    min-height: 46px;
    padding: 0 15px;
    color: #263247;
    background: #fff;
    border: 0;
    outline: 0;
    box-sizing: border-box;
    font: inherit;
  }

  select[multiple] { min-height: 76px; padding: 8px 15px; }

  &__unsupported {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    color: #8993a2;
    border-left: 1px dashed #d7dce5;
  }
  &__unsupported svg { width: 20px; height: 20px; }
}
</style>
