<script setup lang="ts">
import { RiArrowLeftLine } from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import { ElButton, ElDialog } from 'element-plus';
import type { FormEvent, FormEventFieldOption } from './frontend-events';
import { createFormEvent, normalizeFormEvent } from './frontend-events';
import FormSchemaFrontendEventActionDialog from './FormSchemaFrontendEventActionDialog.vue';
import FormSchemaFrontendEventEditor from './FormSchemaFrontendEventEditor.vue';
import FormSchemaFrontendEventRequestDialog from './FormSchemaFrontendEventRequestDialog.vue';

const props = withDefaults(
  defineProps<{
    event?: FormEvent;
    fields: readonly FormEventFieldOption[];
    existingNames?: readonly string[];
  }>(),
  { event: undefined, existingNames: () => [] },
);
const emit = defineEmits<{ save: [event: FormEvent] }>();
const visible = defineModel<boolean>({ required: true });
const step = shallowRef<1 | 2>(1);
const draft = ref<FormEvent>(createFormEvent());
const requestDialogOpen = shallowRef(false);
const actionDialogOpen = shallowRef(false);
const nameError = computed(() => {
  const name = draft.value.name.trim();
  if (!name) return '请输入事件名称';
  return props.existingNames.some((item) => item === name) ? '事件名称已存在' : '';
});
const requestFields = computed(() =>
  props.fields.filter((field) => field.type !== 'separator' && field.type !== 'button'),
);
const selectedTrigger = computed(() =>
  props.fields.find((field) => field.key === draft.value.trigger),
);

watch(
  [visible, () => props.event],
  ([open, event]) => {
    if (!open) return;
    draft.value = structuredClone(event ?? createFormEvent());
    step.value = 1;
  },
  { immediate: true },
);

function save(): void {
  if (nameError.value || !draft.value.trigger || !draft.value.request.url.trim()) return;
  if (!draft.value.action.every((action) => action.field && action.value.trim())) return;
  emit('save', normalizeFormEvent(draft.value));
  visible.value = false;
}
</script>

<template>
  <ElDialog
    v-model="visible"
    class="form-event-dialog"
    width="min(92vw, 840px)"
    :close-on-click-modal="false"
    :lock-scroll="true"
    :title="props.event ? '编辑前端事件' : '添加前端事件'"
    append-to-body
    @closed="step = 1"
  >
    <FormSchemaFrontendEventEditor
      v-model:draft="draft"
      v-model:step="step"
      :fields="props.fields"
      :name-error="nameError"
      :request-fields="requestFields"
      :selected-trigger="selectedTrigger"
      @select-trigger="draft.trigger = $event.key"
      @open-request-settings="requestDialogOpen = true"
      @open-action-settings="actionDialogOpen = true"
    />
    <template #footer>
      <div class="form-event-dialog__footer">
        <button class="form-event-dialog__help" type="button">如何添加前端事件？</button>
        <div class="form-event-dialog__footer-actions">
          <ElButton v-if="step === 2" :icon="RiArrowLeftLine" @click="step = 1">上一步</ElButton>
          <ElButton @click="visible = false">取消</ElButton>
          <ElButton
            v-if="step === 1"
            type="primary"
            :disabled="Boolean(nameError)"
            @click="step = 2"
            >下一步</ElButton
          >
          <ElButton
            v-else
            type="primary"
            :disabled="!draft.trigger || !draft.request.url.trim()"
            @click="save"
            >保存</ElButton
          >
        </div>
      </div>
    </template>
  </ElDialog>
  <FormSchemaFrontendEventRequestDialog
    v-model="requestDialogOpen"
    v-model:request="draft.request"
    :fields="requestFields"
  />
  <FormSchemaFrontendEventActionDialog
    v-model="actionDialogOpen"
    v-model:actions="draft.action"
    :fields="requestFields"
  />
</template>
