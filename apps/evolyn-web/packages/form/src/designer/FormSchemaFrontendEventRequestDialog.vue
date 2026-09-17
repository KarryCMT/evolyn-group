<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line } from '@remixicon/vue';
import { ElButton, ElDialog, ElInput } from 'element-plus';
import type { FormEvent, FormEventFieldOption } from './frontend-events';
import { FORM_EVENT_LIMITS } from './frontend-events';
import FormSchemaEventTokenInput from './FormSchemaEventTokenInput.vue';

const props = defineProps<{ fields: readonly FormEventFieldOption[] }>();
const open = defineModel<boolean>({ required: true });
const request = defineModel<FormEvent['request']>('request', { required: true });
function updateEntry(
  target: 'header' | 'body',
  index: number,
  key: 'key' | 'value',
  value: string,
): void {
  const entries = request.value[target].map((entry, entryIndex) =>
    entryIndex === index ? { ...entry, [key]: value } : entry,
  );
  request.value = { ...request.value, [target]: entries };
}
function addEntry(target: 'header' | 'body'): void {
  if (request.value[target].length >= FORM_EVENT_LIMITS.maxRequestEntries) return;
  request.value = {
    ...request.value,
    [target]: [...request.value[target], { key: '', value: '' }],
  };
}
function removeEntry(target: 'header' | 'body', index: number): void {
  request.value = {
    ...request.value,
    [target]: request.value[target].filter((_entry, entryIndex) => entryIndex !== index),
  };
}
</script>

<template>
  <ElDialog
    v-model="open"
    class="form-event-dialog__nested"
    width="min(92vw, 720px)"
    title="Header / Body 设置"
    :lock-scroll="true"
    append-to-body
  >
    <div class="form-event-dialog__nested-body">
      <h3>Header</h3>
      <div
        v-for="(entry, index) in request.header"
        :key="`header-${index}`"
        class="form-event-dialog__pair-row"
      >
        <ElInput
          :model-value="entry.key"
          placeholder="名称"
          @update:model-value="updateEntry('header', index, 'key', $event)"
        /><FormSchemaEventTokenInput
          :model-value="entry.value"
          :fields="props.fields"
          placeholder="值"
          @update:model-value="updateEntry('header', index, 'value', $event)"
        /><button type="button" aria-label="删除 Header" @click="removeEntry('header', index)">
          <RiDeleteBin6Line />
        </button>
      </div>
      <button class="form-event-dialog__add-row" type="button" @click="addEntry('header')">
        <RiAddLine />添加 Header</button
      ><template v-if="request.method === 'post'"
        ><h3>Body</h3>
        <div
          v-for="(entry, index) in request.body"
          :key="`body-${index}`"
          class="form-event-dialog__pair-row"
        >
          <ElInput
            :model-value="entry.key"
            placeholder="字段名"
            @update:model-value="updateEntry('body', index, 'key', $event)"
          /><FormSchemaEventTokenInput
            :model-value="entry.value"
            :fields="props.fields"
            placeholder="字段值"
            @update:model-value="updateEntry('body', index, 'value', $event)"
          /><button type="button" aria-label="删除 Body 字段" @click="removeEntry('body', index)">
            <RiDeleteBin6Line />
          </button>
        </div>
        <button class="form-event-dialog__add-row" type="button" @click="addEntry('body')">
          <RiAddLine />添加 Body 字段
        </button></template
      >
    </div>
    <template #footer><ElButton @click="open = false">确定</ElButton></template>
  </ElDialog>
</template>
