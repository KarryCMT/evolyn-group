<script setup lang="ts">
import type { FormEvent, FormEventFieldOption } from './frontend-events';
import FormSchemaFrontendEventEditorDialog from './FormSchemaFrontendEventEditorDialog.vue';

const visible = defineModel<boolean>({ required: true });
const props = withDefaults(
  defineProps<{
    event?: FormEvent;
    fields: readonly FormEventFieldOption[];
    existingNames?: readonly string[];
  }>(),
  { event: undefined, existingNames: () => [] },
);
const emit = defineEmits<{ save: [event: FormEvent] }>();
</script>

<template>
  <FormSchemaFrontendEventEditorDialog
    v-model="visible"
    :event="props.event"
    :fields="props.fields"
    :existing-names="props.existingNames"
    @save="emit('save', $event)"
  />
</template>

<style lang="scss">
.form-event-dialog,
.form-event-dialog__nested {
  display: flex;
  max-height: calc(100dvh - 64px);
  flex-direction: column;
  margin: 32px auto !important;
  overflow: hidden;
}
.form-event-dialog .el-dialog__body,
.form-event-dialog__nested .el-dialog__body {
  flex: 1 1 auto;
  min-height: 0;
  padding: 10px 28px 16px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
.form-event-dialog .el-dialog__footer,
.form-event-dialog__nested .el-dialog__footer {
  flex: 0 0 auto;
  padding: 14px 28px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.form-event-dialog__steps,
.form-event-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.form-event-dialog__steps {
  width: min(100%, 500px);
  justify-content: center;
  gap: 14px;
  margin: 0 auto 20px;
}
.form-event-dialog__step {
  display: inline-flex;
  gap: 10px;
  align-items: center;
  color: var(--el-text-color-secondary);
  font-size: 16px;
}
.form-event-dialog__step span {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  background: var(--el-fill-color);
  border-radius: 50%;
}
.form-event-dialog__step.is-active,
.form-event-dialog__step.is-done {
  color: var(--el-text-color-primary);
  font-weight: 600;
}
.form-event-dialog__step.is-active span,
.form-event-dialog__step.is-done span {
  color: #fff;
  background: var(--el-color-primary);
}
.form-event-dialog__steps i {
  width: 92px;
  height: 1px;
  background: var(--el-border-color);
}
.form-event-dialog__steps i.is-active {
  background: var(--el-color-primary);
}
.form-event-dialog__content--intro {
  max-width: 640px;
  margin: 0 auto;
}
.form-event-dialog__section {
  margin-bottom: 26px;
}
.form-event-dialog__section h3 {
  margin: 0 0 12px;
  font-size: 16px;
}
.form-event-dialog__trigger-row {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 12px;
}
.form-event-dialog__field-select {
  display: flex;
  width: 100%;
  height: 32px;
  padding: 0 11px;
  align-items: center;
  justify-content: space-between;
  text-align: left;
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
}
.form-event-dialog__label {
  display: block;
  margin: 22px 0 8px;
  font-weight: 600;
}
.form-event-dialog__label em {
  color: var(--el-color-danger);
  font-style: normal;
}
.form-event-dialog__inline-action {
  display: flex;
  margin-top: 14px;
  align-items: center;
  justify-content: space-between;
}
.form-event-dialog__inline-action button,
.form-event-dialog__add-row,
.form-event-dialog__help {
  display: inline-flex;
  gap: 5px;
  padding: 0;
  align-items: center;
  color: var(--el-color-primary);
  font: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
}
.form-event-dialog__fill-rule {
  width: 220px;
  margin-top: 16px;
}
.form-event-dialog__footer-actions {
  display: flex;
  gap: 10px;
}
.form-event-dialog__nested-body {
  display: flex;
  gap: 10px;
  flex-direction: column;
}
.form-event-dialog__nested-body h3 {
  margin: 12px 0 0;
  font-size: 15px;
}
.form-event-dialog__pair-row {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr) 30px;
  gap: 10px;
  align-items: start;
}
.form-event-dialog__pair-row > button,
.form-event-dialog__mapping-row > button {
  display: grid;
  width: 30px;
  height: 32px;
  place-items: center;
  cursor: pointer;
  background: transparent;
  border: 0;
}
.form-event-dialog__add-row {
  align-self: flex-start;
  margin-top: 2px;
}
.form-event-dialog__mapping-copy {
  margin: 0 0 18px;
  color: var(--el-text-color-secondary);
}
.form-event-dialog__mapping-row {
  display: grid;
  grid-template-columns: 230px 20px minmax(0, 1fr) 30px;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
</style>
