<script setup lang="ts">
import { RiSettings3Line } from '@remixicon/vue';
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElPopover,
  ElRadio,
  ElRadioGroup,
  ElSelect,
} from 'element-plus';
import type { FormEvent, FormEventFieldOption } from './frontend-events';
import { FORM_EVENT_LIMITS } from './frontend-events';
import FormSchemaEventFieldPicker from './FormSchemaEventFieldPicker.vue';
import FormSchemaEventTokenInput from './FormSchemaEventTokenInput.vue';

defineProps<{
  fields: readonly FormEventFieldOption[];
  nameError: string;
  requestFields: readonly FormEventFieldOption[];
  selectedTrigger?: FormEventFieldOption;
}>();

const draft = defineModel<FormEvent>('draft', { required: true });
const step = defineModel<1 | 2>('step', { required: true });
const emit = defineEmits<{
  selectTrigger: [field: FormEventFieldOption];
  openRequestSettings: [];
  openActionSettings: [];
}>();
</script>

<template>
  <div class="form-event-dialog__steps" aria-label="前端事件配置步骤">
    <div :class="['form-event-dialog__step', { 'is-active': step === 1, 'is-done': step === 2 }]">
      <span>{{ step === 2 ? '✓' : '1' }}</span
      >名称信息
    </div>
    <i :class="{ 'is-active': step === 2 }" />
    <div :class="['form-event-dialog__step', { 'is-active': step === 2 }]">
      <span>2</span>触发与执行
    </div>
  </div>

  <section v-if="step === 1" class="form-event-dialog__content form-event-dialog__content--intro">
    <ElForm label-position="top" @submit.prevent>
      <ElFormItem label="事件名称" required :error="nameError">
        <ElInput
          v-model="draft.name"
          :maxlength="FORM_EVENT_LIMITS.nameMaxLength"
          autofocus
          placeholder="例如：自动补全员工信息"
        />
      </ElFormItem>
      <ElFormItem label="事件说明">
        <ElInput
          v-model="draft.description"
          :maxlength="FORM_EVENT_LIMITS.descriptionMaxLength"
          :rows="3"
          type="textarea"
          placeholder="说明此事件的触发时机和执行目的"
        />
      </ElFormItem>
    </ElForm>
  </section>

  <section v-else class="form-event-dialog__content">
    <div class="form-event-dialog__section">
      <h3>触发动作</h3>
      <div class="form-event-dialog__trigger-row">
        <ElSelect model-value="widget" disabled aria-label="触发类型"
          ><ElOption label="字段触发" value="widget" /></ElSelect
        ><ElPopover placement="bottom-start" :width="360" trigger="click"
          ><template #reference
            ><button class="form-event-dialog__field-select" type="button">
              <span>{{ selectedTrigger?.label || '请选择触发字段' }}</span
              ><span aria-hidden="true">⌄</span>
            </button></template
          ><FormSchemaEventFieldPicker
            :fields="requestFields"
            @select="emit('selectTrigger', $event)"
        /></ElPopover>
      </div>
    </div>
    <div class="form-event-dialog__section">
      <h3>执行动作</h3>
      <ElSelect model-value="request" disabled aria-label="执行动作"
        ><ElOption label="自定义请求" value="request"
      /></ElSelect>
    </div>
    <div class="form-event-dialog__section form-event-dialog__request">
      <h3>请求类型</h3>
      <ElRadioGroup v-model="draft.request.method"
        ><ElRadio value="get">GET</ElRadio><ElRadio value="post">POST</ElRadio></ElRadioGroup
      ><label class="form-event-dialog__label">URL <em>*</em></label
      ><FormSchemaEventTokenInput
        v-model="draft.request.url"
        :fields="requestFields"
        :rows="3"
        placeholder="https://api.example.com/lookup?name=${_widget_name}"
      />
      <div class="form-event-dialog__inline-action">
        <span>Header / Body</span
        ><button type="button" @click="emit('openRequestSettings')">
          <RiSettings3Line />点击设置
        </button>
      </div>
    </div>
    <div class="form-event-dialog__section form-event-dialog__response">
      <h3>返回值格式</h3>
      <ElRadioGroup v-model="draft.request.format"
        ><ElRadio value="json">JSON</ElRadio><ElRadio value="xml">XML</ElRadio></ElRadioGroup
      >
      <div class="form-event-dialog__inline-action">
        <span>返回值设置</span
        ><button type="button" @click="emit('openActionSettings')">
          <RiSettings3Line />{{
            draft.action.length ? `已设置 ${draft.action.length} 项` : '点击设置'
          }}
        </button>
      </div>
      <ElSelect
        v-model="draft.subform_fill_rule"
        class="form-event-dialog__fill-rule"
        aria-label="子表单写入规则"
        ><ElOption label="子表单数据合并" value="merge" /><ElOption
          label="子表单数据替换"
          value="replace"
      /></ElSelect>
    </div>
  </section>
</template>
