<script setup lang="ts">
import { RiAddLine, RiInformationLine, RiSearchLine, RiSettings3Line } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElPopover,
  ElSwitch,
  ElTooltip,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import { SUBMIT_VALIDATOR_SOURCE_TYPES, widgetTypeLabel } from '../schema/dictionary';
import type { FormulaEditorInsertion } from './formula-editor';
import SubmitTemplateEditor from './SubmitTemplateEditor.vue';
import type { PreSubmitConfirmDraft } from './submit-validation-types';

const confirm = defineModel<PreSubmitConfirmDraft>({ required: true });
const props = defineProps<{ items: FormItem[] }>();
const dialogOpen = shallowRef(false);
const activePicker = shallowRef<'title' | 'content'>();
const variableKeyword = shallowRef('');
const titleInsertion = shallowRef<FormulaEditorInsertion>();
const contentInsertion = shallowRef<FormulaEditorInsertion>();
const insertionSequence = shallowRef(0);

const variableItems = computed(() =>
  props.items.filter((item) => SUBMIT_VALIDATOR_SOURCE_TYPES.includes(item.widget.type)),
);
const templateFields = computed(() =>
  variableItems.value.map((item) => ({
    widgetName: item.widget.widgetName,
    label: item.label,
  })),
);
const filteredVariableItems = computed(() => {
  const keyword = variableKeyword.value.trim().toLocaleLowerCase();
  if (!keyword) return variableItems.value;
  return variableItems.value.filter((item) =>
    `${item.label} ${widgetTypeLabel(item.widget.type)}`.toLocaleLowerCase().includes(keyword),
  );
});
const enabled = computed({
  get: () => confirm.value.enable,
  set: (enable: boolean) => {
    confirm.value = { ...confirm.value, enable };
  },
});
const title = computed({
  get: () => confirm.value.title,
  set: (nextTitle: string) => {
    confirm.value = { ...confirm.value, title: nextTitle };
  },
});
const content = computed({
  get: () => confirm.value.content,
  set: (nextContent: string) => {
    confirm.value = { ...confirm.value, content: nextContent };
  },
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
  const insertion: FormulaEditorInsertion = {
    id: insertionSequence.value,
    text: `\${${widgetName}}`,
  };
  if (target === 'title') titleInsertion.value = insertion;
  else contentInsertion.value = insertion;
  activePicker.value = undefined;
  variableKeyword.value = '';
}
</script>

<template>
  <section class="form-pre-submit-confirm" aria-label="提交时二次确认">
    <div class="form-pre-submit-confirm__heading">
      <span class="form-pre-submit-confirm__title">二次确认</span>
      <el-tooltip content="提交前提示填写人再次核对关键信息" placement="top">
        <el-icon class="form-pre-submit-confirm__help" aria-label="二次确认说明">
          <RiInformationLine />
        </el-icon>
      </el-tooltip>
    </div>
    <div class="form-pre-submit-confirm__control">
      <span class="form-pre-submit-confirm__caption">提交时弹出确认提示</span>
      <el-switch v-model="enabled" aria-label="开启提交二次确认" />
    </div>
    <button
      v-if="enabled"
      type="button"
      class="form-pre-submit-confirm__configure"
      @click="dialogOpen = true"
    >
      <span>设置提示文案</span>
      <el-icon><RiSettings3Line /></el-icon>
    </button>

    <el-dialog
      v-model="dialogOpen"
      append-to-body
      width="min(92vw, 640px)"
      class="form-pre-submit-confirm__dialog"
      title="二次确认设置"
    >
      <p class="form-pre-submit-confirm__intro">
        成员点击提交按钮时进行弹窗确认
        <span class="form-pre-submit-confirm__preview">预览效果</span>
      </p>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="提示标题" required>
          <div class="form-pre-submit-confirm__template-input">
            <SubmitTemplateEditor
              v-model="title"
              :fields="templateFields"
              :insertion="titleInsertion"
              :max-length="100"
              placeholder="确认继续提交吗？"
            />
            <el-popover
              v-model:visible="titlePickerOpen"
              placement="bottom-end"
              :width="480"
              :teleported="false"
              trigger="click"
            >
              <template #reference>
                <el-button class="form-pre-submit-confirm__field-add" aria-label="向标题插入字段">
                  <el-icon><RiAddLine /></el-icon>
                </el-button>
              </template>
              <div
                class="form-pre-submit-confirm__field-picker"
                role="listbox"
                aria-label="可插入字段"
              >
                <el-input
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
                  <span>{{ item.label }}</span>
                  <small>{{ widgetTypeLabel(item.widget.type) }}</small>
                </button>
                <p v-if="filteredVariableItems.length === 0">未找到可插入字段</p>
              </div>
            </el-popover>
          </div>
        </el-form-item>
        <el-form-item label="提示文字">
          <div class="form-pre-submit-confirm__template-input">
            <SubmitTemplateEditor
              v-model="content"
              :fields="templateFields"
              :insertion="contentInsertion"
              :max-length="1000"
              placeholder="请确认填写内容无误后继续提交。"
            />
            <el-popover
              v-model:visible="contentPickerOpen"
              placement="bottom-end"
              :width="480"
              :teleported="false"
              trigger="click"
            >
              <template #reference>
                <el-button
                  class="form-pre-submit-confirm__field-add"
                  aria-label="向提示文字插入字段"
                >
                  <el-icon><RiAddLine /></el-icon>
                </el-button>
              </template>
              <div
                class="form-pre-submit-confirm__field-picker"
                role="listbox"
                aria-label="可插入字段"
              >
                <el-input
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
                  <span>{{ item.label }}</span>
                  <small>{{ widgetTypeLabel(item.widget.type) }}</small>
                </button>
                <p v-if="filteredVariableItems.length === 0">未找到可插入字段</p>
              </div>
            </el-popover>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogOpen = false">完成</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.form-pre-submit-confirm {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;

  &__heading,
  &__control {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  &__heading {
    justify-content: flex-start;
    gap: 6px;
  }
  &__title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  &__help {
    font-size: 16px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }
  &__caption {
    font-size: 14px;
    color: var(--el-text-color-regular);
  }

  &__configure {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    align-self: flex-start;
    padding: 0;
    font: inherit;
    font-size: 13px;
    color: var(--el-color-primary);
    cursor: pointer;
    background: none;
    border: 0;

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary-light-3);
      outline: none;
    }
  }

  &__intro {
    margin: 0 0 20px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.65;
  }
  &__preview {
    padding: 0;
    margin-left: 8px;
    font: inherit;
    color: var(--el-color-primary);
  }
  &__template-input {
    display: flex;
    width: 100%;
    min-width: 0;
    min-height: 38px;
    overflow: visible;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    transition: border-color var(--el-transition-duration);
  }
  &__template-input:focus-within {
    border-color: var(--el-color-primary);
  }
  &__template-input .submit-template-editor {
    flex: 1 1 auto;
    min-width: 0;
  }
  &__field-add.el-button {
    flex: 0 0 42px;
    width: 42px;
    height: 38px;
    padding: 0 8px;
    margin: 0;
    color: var(--el-color-primary);
    border-top: 0;
    border-right: 0;
    border-bottom: 0;
    border-left-color: var(--el-border-color);
    border-radius: 0 var(--el-border-radius-base) var(--el-border-radius-base) 0;
  }

  &__field-picker {
    max-height: 360px;
    overflow-y: auto;
  }
  &__field-search {
    position: sticky;
    top: 0;
    z-index: 1;
    padding: 8px;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  &__field-picker button {
    display: flex;
    width: 100%;
    min-height: 48px;
    padding: 0 14px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 6px;
  }
  &__field-picker button:hover {
    background: var(--el-fill-color-light);
  }
  &__field-picker small {
    padding: 2px 10px;
    font-size: 12px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 999px;
  }
  &__field-picker p {
    margin: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}
</style>
