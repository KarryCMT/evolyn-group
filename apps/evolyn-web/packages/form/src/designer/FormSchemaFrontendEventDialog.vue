<script setup lang="ts">
import { RiAddLine, RiArrowLeftLine, RiDeleteBin6Line, RiSettings3Line } from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElPopover,
  ElRadio,
  ElRadioGroup,
  ElSelect,
} from 'element-plus';
import type { FormEvent, FormEventAction, FormEventFieldOption, FormEventRequestEntry } from './frontend-events';
import { createFormEvent, FORM_EVENT_LIMITS, normalizeFormEvent } from './frontend-events';
import FormSchemaEventFieldPicker from './FormSchemaEventFieldPicker.vue';
import FormSchemaEventTokenInput from './FormSchemaEventTokenInput.vue';

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

const step = shallowRef<1 | 2>(1);
const draft = ref<FormEvent>(createFormEvent());
const headerBodyOpen = shallowRef(false);
const actionEditorOpen = shallowRef(false);
const nameError = computed(() => {
  const name = draft.value.name.trim();
  if (!name) return '请输入事件名称';
  if (props.existingNames.some((item) => item === name)) return '事件名称已存在';
  return '';
});
const requestFields = computed(() => props.fields.filter((field) => field.type !== 'separator' && field.type !== 'button'));
const selectedTrigger = computed(() => props.fields.find((field) => field.key === draft.value.trigger));
const selectedActionFields = computed(() => new Set(draft.value.action.map((action) => action.field)));

watch(
  [visible, () => props.event],
  ([open, event]) => {
    if (!open) return;
    draft.value = structuredClone(event ?? createFormEvent());
    step.value = 1;
  },
  { immediate: true },
);

function selectTrigger(field: FormEventFieldOption): void {
  draft.value.trigger = field.key;
}

function nextStep(): void {
  if (nameError.value) return;
  step.value = 2;
}

function addRequestEntry(target: 'header' | 'body'): void {
  const values = draft.value.request[target];
  if (values.length >= FORM_EVENT_LIMITS.maxRequestEntries) return;
  values.push({ key: '', value: '' });
}

function removeRequestEntry(target: 'header' | 'body', index: number): void {
  draft.value.request[target].splice(index, 1);
}

function addAction(): void {
  if (draft.value.action.length >= FORM_EVENT_LIMITS.maxActions) return;
  draft.value.action.push({ field: '', value: '' });
}

function removeAction(index: number): void {
  draft.value.action.splice(index, 1);
}

function updateRequestEntry(entry: FormEventRequestEntry, key: 'key' | 'value', value: string): void {
  entry[key] = value;
}

function updateAction(action: FormEventAction, key: 'field' | 'value', value: string): void {
  action[key] = value;
}

function save(): void {
  if (nameError.value || !draft.value.trigger || !draft.value.request.url.trim()) return;
  const actionsValid = draft.value.action.every((action) => action.field && action.value.trim());
  if (!actionsValid) return;
  emit('save', normalizeFormEvent(draft.value));
  visible.value = false;
}

function close(): void {
  visible.value = false;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="form-event-dialog"
    width="min(92vw, 840px)"
    :close-on-click-modal="false"
    :lock-scroll="true"
    :show-close="true"
    :title="props.event ? '编辑前端事件' : '添加前端事件'"
    append-to-body
    @closed="step = 1"
  >
    <div class="form-event-dialog__steps" aria-label="前端事件配置步骤">
      <div :class="['form-event-dialog__step', { 'is-active': step === 1, 'is-done': step === 2 }]">
        <span>{{ step === 2 ? '✓' : '1' }}</span>名称信息
      </div>
      <i :class="{ 'is-active': step === 2 }" />
      <div :class="['form-event-dialog__step', { 'is-active': step === 2 }]">
        <span>2</span>触发与执行
      </div>
    </div>

    <section v-if="step === 1" class="form-event-dialog__content form-event-dialog__content--intro">
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="事件名称" required :error="nameError">
          <el-input v-model="draft.name" :maxlength="FORM_EVENT_LIMITS.nameMaxLength" autofocus placeholder="例如：自动补全员工信息" />
        </el-form-item>
        <el-form-item label="事件说明">
          <el-input v-model="draft.description" :maxlength="FORM_EVENT_LIMITS.descriptionMaxLength" :rows="3" type="textarea" placeholder="说明此事件的触发时机和执行目的" />
        </el-form-item>
      </el-form>
    </section>

    <section v-else class="form-event-dialog__content">
      <div class="form-event-dialog__section">
        <h3>触发动作</h3>
        <div class="form-event-dialog__trigger-row">
          <el-select model-value="widget" disabled aria-label="触发类型"><el-option label="字段触发" value="widget" /></el-select>
          <el-popover placement="bottom-start" :width="360" trigger="click">
            <template #reference>
              <button class="form-event-dialog__field-select" type="button">
                <span>{{ selectedTrigger?.label || '请选择触发字段' }}</span><span aria-hidden="true">⌄</span>
              </button>
            </template>
            <FormSchemaEventFieldPicker :fields="requestFields" @select="selectTrigger" />
          </el-popover>
        </div>
      </div>

      <div class="form-event-dialog__section">
        <h3>执行动作</h3>
        <el-select model-value="request" disabled aria-label="执行动作"><el-option label="自定义请求" value="request" /></el-select>
      </div>

      <div class="form-event-dialog__section form-event-dialog__request">
        <h3>请求类型</h3>
        <el-radio-group v-model="draft.request.method">
          <el-radio value="get">GET</el-radio><el-radio value="post">POST</el-radio>
        </el-radio-group>
        <label class="form-event-dialog__label">URL <em>*</em></label>
        <FormSchemaEventTokenInput v-model="draft.request.url" :fields="requestFields" :rows="3" placeholder="https://api.example.com/lookup?name=${_widget_name}" />
        <div class="form-event-dialog__inline-action">
          <span>Header / Body</span>
          <button type="button" @click="headerBodyOpen = true"><RiSettings3Line />点击设置</button>
        </div>
      </div>

      <div class="form-event-dialog__section form-event-dialog__response">
        <h3>返回值格式</h3>
        <el-radio-group v-model="draft.request.format"><el-radio value="json">JSON</el-radio><el-radio value="xml">XML</el-radio></el-radio-group>
        <div class="form-event-dialog__inline-action">
          <span>返回值设置</span>
          <button type="button" @click="actionEditorOpen = true"><RiSettings3Line />{{ draft.action.length ? `已设置 ${draft.action.length} 项` : '点击设置' }}</button>
        </div>
        <el-select v-model="draft.subform_fill_rule" class="form-event-dialog__fill-rule" aria-label="子表单写入规则">
          <el-option label="子表单数据合并" value="merge" /><el-option label="子表单数据替换" value="replace" />
        </el-select>
      </div>
    </section>

    <template #footer>
      <div class="form-event-dialog__footer">
        <button class="form-event-dialog__help" type="button">如何添加前端事件？</button>
        <div class="form-event-dialog__footer-actions">
          <el-button v-if="step === 2" :icon="RiArrowLeftLine" @click="step = 1">上一步</el-button>
          <el-button @click="close">取消</el-button>
          <el-button v-if="step === 1" type="primary" :disabled="Boolean(nameError)" @click="nextStep">下一步</el-button>
          <el-button v-else type="primary" :disabled="!draft.trigger || !draft.request.url.trim()" @click="save">保存</el-button>
        </div>
      </div>
    </template>
  </el-dialog>

  <el-dialog v-model="headerBodyOpen" class="form-event-dialog__nested" width="min(92vw, 720px)" title="Header / Body 设置" :lock-scroll="true" append-to-body>
    <div class="form-event-dialog__nested-body">
      <h3>Header</h3>
      <div v-for="(entry, index) in draft.request.header" :key="`header-${index}`" class="form-event-dialog__pair-row">
        <el-input :model-value="entry.key" placeholder="名称" @update:model-value="updateRequestEntry(entry, 'key', $event)" />
        <FormSchemaEventTokenInput :model-value="entry.value" :fields="requestFields" placeholder="值" @update:model-value="updateRequestEntry(entry, 'value', $event)" />
        <button type="button" aria-label="删除 Header" @click="removeRequestEntry('header', index)"><RiDeleteBin6Line /></button>
      </div>
      <button class="form-event-dialog__add-row" type="button" @click="addRequestEntry('header')"><RiAddLine />添加 Header</button>
      <template v-if="draft.request.method === 'post'">
        <h3>Body</h3>
        <div v-for="(entry, index) in draft.request.body" :key="`body-${index}`" class="form-event-dialog__pair-row">
          <el-input :model-value="entry.key" placeholder="字段名" @update:model-value="updateRequestEntry(entry, 'key', $event)" />
          <FormSchemaEventTokenInput :model-value="entry.value" :fields="requestFields" placeholder="字段值" @update:model-value="updateRequestEntry(entry, 'value', $event)" />
          <button type="button" aria-label="删除 Body 字段" @click="removeRequestEntry('body', index)"><RiDeleteBin6Line /></button>
        </div>
        <button class="form-event-dialog__add-row" type="button" @click="addRequestEntry('body')"><RiAddLine />添加 Body 字段</button>
      </template>
    </div>
    <template #footer><el-button @click="headerBodyOpen = false">确定</el-button></template>
  </el-dialog>

  <el-dialog v-model="actionEditorOpen" class="form-event-dialog__nested" width="min(92vw, 720px)" title="返回值设置" :lock-scroll="true" append-to-body>
    <p class="form-event-dialog__mapping-copy">解析请求返回值，并将结果写入指定的表单字段。</p>
    <div v-for="(action, index) in draft.action" :key="`action-${index}`" class="form-event-dialog__mapping-row">
      <el-select :model-value="action.field" placeholder="选择表单字段" @update:model-value="updateAction(action, 'field', $event)">
        <el-option v-for="field in requestFields" :key="field.key" :label="field.group ? `${field.group} · ${field.label}` : field.label" :value="field.key" :disabled="selectedActionFields.has(field.key) && action.field !== field.key" />
      </el-select>
      <span>=</span>
      <FormSchemaEventTokenInput :model-value="action.value" :fields="requestFields" placeholder="返回值路径或字段模板" @update:model-value="updateAction(action, 'value', $event)" />
      <button type="button" aria-label="删除返回值映射" @click="removeAction(index)"><RiDeleteBin6Line /></button>
    </div>
    <button class="form-event-dialog__add-row" type="button" @click="addAction"><RiAddLine />表单字段及对应返回值</button>
    <template #footer><el-button @click="actionEditorOpen = false">确定</el-button></template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.form-event-dialog), :global(.form-event-dialog__nested) { display: flex; max-height: calc(100dvh - 64px); flex-direction: column; margin: 32px auto !important; overflow: hidden; }
:global(.form-event-dialog .el-dialog__body), :global(.form-event-dialog__nested .el-dialog__body) { flex: 1 1 auto; min-height: 0; padding: 10px 28px 16px; overflow-y: auto; overscroll-behavior: contain; }
:global(.form-event-dialog .el-dialog__footer), :global(.form-event-dialog__nested .el-dialog__footer) { flex: 0 0 auto; padding: 14px 28px; border-top: 1px solid var(--el-border-color-lighter); }
.form-event-dialog__steps { display: flex; width: min(100%, 500px); align-items: center; justify-content: center; gap: 14px; margin: 0 auto 20px; }
.form-event-dialog__step { display: inline-flex; gap: 10px; align-items: center; color: var(--el-text-color-secondary); font-size: 16px; }
.form-event-dialog__step span { display: grid; width: 30px; height: 30px; place-items: center; color: var(--el-text-color-secondary); background: var(--el-fill-color); border-radius: 50%; }
.form-event-dialog__step.is-active, .form-event-dialog__step.is-done { color: var(--el-text-color-primary); font-weight: 600; }
.form-event-dialog__step.is-active span, .form-event-dialog__step.is-done span { color: #fff; background: var(--el-color-primary); }
.form-event-dialog__steps i { width: 92px; height: 1px; background: var(--el-border-color); }
.form-event-dialog__steps i.is-active { background: var(--el-color-primary); }
.form-event-dialog__content { min-height: 0; }
.form-event-dialog__content--intro { max-width: 640px; margin: 0 auto; }
.form-event-dialog__section { margin-bottom: 26px; }
.form-event-dialog__section h3 { margin: 0 0 12px; color: var(--el-text-color-primary); font-size: 16px; }
.form-event-dialog__trigger-row { display: grid; grid-template-columns: 240px minmax(0, 1fr); gap: 12px; }
.form-event-dialog__field-select { display: flex; width: 100%; height: 32px; align-items: center; justify-content: space-between; padding: 0 11px; color: var(--el-text-color-secondary); text-align: left; cursor: pointer; background: var(--el-bg-color); border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); }
.form-event-dialog__field-select:hover { border-color: var(--el-color-primary); }
.form-event-dialog__label { display: block; margin: 22px 0 8px; color: var(--el-text-color-primary); font-weight: 600; }
.form-event-dialog__label em { color: var(--el-color-danger); font-style: normal; }
.form-event-dialog__inline-action { display: flex; align-items: center; justify-content: space-between; margin-top: 14px; color: var(--el-text-color-regular); }
.form-event-dialog__inline-action button, .form-event-dialog__add-row, .form-event-dialog__help { display: inline-flex; gap: 5px; align-items: center; padding: 0; color: var(--el-color-primary); font: inherit; cursor: pointer; background: transparent; border: 0; }
.form-event-dialog__fill-rule { width: 220px; margin-top: 16px; }
.form-event-dialog__footer { display: flex; align-items: center; justify-content: space-between; }
.form-event-dialog__footer-actions { display: flex; gap: 10px; }
.form-event-dialog__nested-body { display: flex; flex-direction: column; gap: 10px; min-height: 0; }
.form-event-dialog__nested-body h3 { margin: 12px 0 0; font-size: 15px; }
.form-event-dialog__pair-row { display: grid; grid-template-columns: 160px minmax(0, 1fr) 30px; gap: 10px; align-items: start; }
.form-event-dialog__pair-row > button, .form-event-dialog__mapping-row > button { display: grid; width: 30px; height: 32px; place-items: center; color: var(--el-text-color-secondary); cursor: pointer; background: transparent; border: 0; }
.form-event-dialog__pair-row > button:hover, .form-event-dialog__mapping-row > button:hover { color: var(--el-color-danger); }
.form-event-dialog__add-row { align-self: flex-start; margin-top: 2px; }
.form-event-dialog__mapping-copy { margin: 0 0 18px; color: var(--el-text-color-secondary); }
.form-event-dialog__mapping-row { display: grid; grid-template-columns: 230px 20px minmax(0, 1fr) 30px; gap: 10px; align-items: center; margin-bottom: 10px; }
@media (max-width: 700px) { .form-event-dialog__steps i { width: 30px; } .form-event-dialog__trigger-row, .form-event-dialog__pair-row, .form-event-dialog__mapping-row { grid-template-columns: 1fr; } .form-event-dialog__pair-row > button, .form-event-dialog__mapping-row > button { justify-self: end; } }
</style>
