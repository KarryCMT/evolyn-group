<script setup lang="ts">
import { RiAddLine, RiSearchLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import { ElButton, ElDialog, ElForm, ElFormItem, ElIcon, ElInput, ElPopover } from 'element-plus';
import type { FormItem } from '../schema/types';
import { SUBMIT_VALIDATOR_SOURCE_TYPES, widgetTypeLabel } from '../schema/dictionary';
import type { FormulaEditorInsertion } from './formula-editor';
import SubmitTemplateEditor from './SubmitTemplateEditor.vue';
import type { PreSubmitConfirmDraft } from './submit-validation-types';

const props = defineProps<{ items: FormItem[] }>();
const open = defineModel<boolean>({ required: true });
const confirm = defineModel<PreSubmitConfirmDraft>('confirm', { required: true });
const activePicker = shallowRef<'title' | 'content'>();
const variableKeyword = shallowRef('');
const titleInsertion = shallowRef<FormulaEditorInsertion>();
const contentInsertion = shallowRef<FormulaEditorInsertion>();
const insertionSequence = shallowRef(0);
const variableItems = computed(() =>
  props.items.filter((item) => SUBMIT_VALIDATOR_SOURCE_TYPES.includes(item.widget.type)),
);
const templateFields = computed(() =>
  variableItems.value.map((item) => ({ widgetName: item.widget.widgetName, label: item.label })),
);
const filteredVariableItems = computed(() => {
  const keyword = variableKeyword.value.trim().toLocaleLowerCase();
  return keyword
    ? variableItems.value.filter((item) =>
        `${item.label} ${widgetTypeLabel(item.widget.type)}`.toLocaleLowerCase().includes(keyword),
      )
    : variableItems.value;
});
const title = computed({
  get: () => confirm.value.title,
  set: (value: string) => (confirm.value = { ...confirm.value, title: value }),
});
const content = computed({
  get: () => confirm.value.content,
  set: (value: string) => (confirm.value = { ...confirm.value, content: value }),
});
const titlePickerOpen = computed({
  get: () => activePicker.value === 'title',
  set: (visible: boolean) => {
    activePicker.value = visible ? 'title' : undefined;
    if (!visible) variableKeyword.value = '';
  },
});
const contentPickerOpen = computed({
  get: () => activePicker.value === 'content',
  set: (visible: boolean) => {
    activePicker.value = visible ? 'content' : undefined;
    if (!visible) variableKeyword.value = '';
  },
});
function insertField(target: 'title' | 'content', widgetName: string): void {
  insertionSequence.value += 1;
  const insertion = { id: insertionSequence.value, text: `\${${widgetName}}` };
  if (target === 'title') titleInsertion.value = insertion;
  else contentInsertion.value = insertion;
  activePicker.value = undefined;
  variableKeyword.value = '';
}
</script>

<template>
  <ElDialog
    v-model="open"
    append-to-body
    width="min(92vw, 640px)"
    class="form-pre-submit-confirm__dialog"
    title="二次确认设置"
  >
    <p class="form-pre-submit-confirm__intro">
      成员点击提交按钮时进行弹窗确认 <span class="form-pre-submit-confirm__preview">预览效果</span>
    </p>
    <ElForm label-position="top" @submit.prevent>
      <ElFormItem label="提示标题" required>
        <div class="form-pre-submit-confirm__template-input">
          <SubmitTemplateEditor
            v-model="title"
            :fields="templateFields"
            :insertion="titleInsertion"
            :max-length="100"
            placeholder="确认继续提交吗？"
          />
          <ElPopover
            v-model:visible="titlePickerOpen"
            placement="bottom-end"
            :width="480"
            :teleported="false"
            trigger="click"
          >
            <template #reference
              ><ElButton class="form-pre-submit-confirm__field-add" aria-label="向标题插入字段"
                ><ElIcon><RiAddLine /></ElIcon></ElButton
            ></template>
            <div
              class="form-pre-submit-confirm__field-picker"
              role="listbox"
              aria-label="可插入字段"
            >
              <ElInput
                v-model="variableKeyword"
                class="form-pre-submit-confirm__field-search"
                placeholder="搜索"
                :prefix-icon="RiSearchLine"
              />
              <button
                v-for="item in filteredVariableItems"
                :key="item.widget.widgetName"
                type="button"
                @click="insertField('title', item.widget.widgetName)"
              >
                <span>{{ item.label }}</span
                ><small>{{ widgetTypeLabel(item.widget.type) }}</small>
              </button>
              <p v-if="filteredVariableItems.length === 0">未找到可插入字段</p>
            </div>
          </ElPopover>
        </div>
      </ElFormItem>
      <ElFormItem label="提示文字">
        <div class="form-pre-submit-confirm__template-input">
          <SubmitTemplateEditor
            v-model="content"
            :fields="templateFields"
            :insertion="contentInsertion"
            :max-length="1000"
            placeholder="请确认填写内容无误后继续提交。"
          />
          <ElPopover
            v-model:visible="contentPickerOpen"
            placement="bottom-end"
            :width="480"
            :teleported="false"
            trigger="click"
          >
            <template #reference
              ><ElButton class="form-pre-submit-confirm__field-add" aria-label="向提示文字插入字段"
                ><ElIcon><RiAddLine /></ElIcon></ElButton
            ></template>
            <div
              class="form-pre-submit-confirm__field-picker"
              role="listbox"
              aria-label="可插入字段"
            >
              <ElInput
                v-model="variableKeyword"
                class="form-pre-submit-confirm__field-search"
                placeholder="搜索"
                :prefix-icon="RiSearchLine"
              />
              <button
                v-for="item in filteredVariableItems"
                :key="item.widget.widgetName"
                type="button"
                @click="insertField('content', item.widget.widgetName)"
              >
                <span>{{ item.label }}</span
                ><small>{{ widgetTypeLabel(item.widget.type) }}</small>
              </button>
              <p v-if="filteredVariableItems.length === 0">未找到可插入字段</p>
            </div>
          </ElPopover>
        </div>
      </ElFormItem>
    </ElForm>
    <template #footer><ElButton @click="open = false">完成</ElButton></template>
  </ElDialog>
</template>
