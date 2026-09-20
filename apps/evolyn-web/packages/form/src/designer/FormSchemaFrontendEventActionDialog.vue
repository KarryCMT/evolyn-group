<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line } from '@remixicon/vue';
import { computed } from 'vue';
import { ElButton, ElDialog, ElOption, ElSelect } from 'element-plus';
import type { FormEventAction, FormEventFieldOption } from './frontend-events';
import { FORM_EVENT_LIMITS } from './frontend-events';
import FormSchemaEventTokenInput from './FormSchemaEventTokenInput.vue';

const props = defineProps<{ fields: readonly FormEventFieldOption[] }>();
const open = defineModel<boolean>({ required: true });
const actions = defineModel<FormEventAction[]>('actions', { required: true });
const selectedFields = computed(() => new Set(actions.value.map((action) => action.field)));
function updateAction(index: number, key: 'field' | 'value', value: string): void {
  actions.value = actions.value.map((action, actionIndex) =>
    actionIndex === index ? { ...action, [key]: value } : action,
  );
}
function addAction(): void {
  if (actions.value.length >= FORM_EVENT_LIMITS.maxActions) return;
  actions.value = [...actions.value, { field: '', value: '' }];
}
function removeAction(index: number): void {
  actions.value = actions.value.filter((_action, actionIndex) => actionIndex !== index);
}
</script>

<template>
  <ElDialog
    v-model="open"
    class="form-event-dialog__nested"
    width="min(92vw, 720px)"
    title="返回值设置"
    :lock-scroll="true"
    append-to-body
  >
    <p class="form-event-dialog__mapping-copy">解析请求返回值，并将结果写入指定的表单字段。</p>
    <div
      v-for="(action, index) in actions"
      :key="`action-${index}`"
      class="form-event-dialog__mapping-row"
    >
      <ElSelect
        :model-value="action.field"
        size="large"
        placeholder="选择表单字段"
        @update:model-value="updateAction(index, 'field', $event)"
        ><ElOption
          v-for="field in props.fields"
          :key="field.key"
          :label="field.group ? `${field.group} · ${field.label}` : field.label"
          :value="field.key"
          :disabled="selectedFields.has(field.key) && action.field !== field.key" /></ElSelect
      ><span>=</span
      ><FormSchemaEventTokenInput
        :model-value="action.value"
        :fields="props.fields"
        placeholder="返回值路径或字段模板"
        @update:model-value="updateAction(index, 'value', $event)"
      /><button type="button" aria-label="删除返回值映射" @click="removeAction(index)">
        <RiDeleteBin6Line />
      </button>
    </div>
    <button class="form-event-dialog__add-row" type="button" @click="addAction">
      <RiAddLine />表单字段及对应返回值
    </button>
    <template #footer><ElButton @click="open = false">确定</ElButton></template>
  </ElDialog>
</template>
