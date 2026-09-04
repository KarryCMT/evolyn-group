<script setup lang="ts">
import {
  RiAddLine,
  RiCloseLine,
  RiErrorWarningLine,
  RiFileCopyLine,
  RiFullscreenLine,
  RiFunctionLine,
  RiInformationLine,
  RiSearchLine,
} from '@remixicon/vue';
import { computed, reactive, shallowRef, watch } from 'vue';
import {
  ElButton,
  ElCheckbox,
  ElDialog,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElPopover,
  ElRadio,
  ElRadioGroup,
  ElTooltip,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import {
  FORMULA_FUNCTIONS,
  collectFormulaDiagnostics,
  FormulaFunctionLibrary,
  projectFormulaContext,
  type FormulaEditorField,
  type FormulaEditorFunction,
  type FormulaEditorInsertion,
} from '../formula';
import FormulaEditor from './FormulaEditor.vue';
import { createSubmitValidatorDraft, type SubmitValidatorDraft } from './submit-validation-types';

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    items: FormItem[];
    validator?: SubmitValidatorDraft;
  }>(),
  { modelValue: false, items: () => [], validator: undefined },
);

const emit = defineEmits<{
  'update:model-value': [open: boolean];
  save: [validator: SubmitValidatorDraft];
}>();

const draft = reactive<SubmitValidatorDraft>(createSubmitValidatorDraft());
const formulaDialogOpen = shallowRef(false);
const formulaDialogExpanded = shallowRef(false);
const formulaInsertion = shallowRef<FormulaEditorInsertion>();
const formulaInsertionSequence = shallowRef(0);
const fieldPickerOpen = shallowRef(false);
const reminderIssue = shallowRef('');
const formulaIssue = shallowRef('');

/** 提示文案插值可引用所有有业务值的顶层字段。 */
const fieldItems = computed(() =>
  props.items.filter((item) => item.widget.type !== 'separator' && item.widget.type !== 'button'),
);
/** 变量面板始终基于当前草稿投影，避免未保存字段被服务端旧版本覆盖。 */
const formulaFields = computed<FormulaEditorField[]>(() => projectFormulaContext(props.items));
const formulaFieldTypeByName = computed(
  () => new Map(formulaFields.value.map((field) => [field.widgetName, field.displayType])),
);
const formulaFieldLabelByName = computed(
  () => new Map(formulaFields.value.map((field) => [field.widgetName, field.label])),
);
const formulaFunctions = FORMULA_FUNCTIONS;
// 与编辑器内联标记共用同一分析器，让横幅提示、确认按钮和保存校验保持一致。
const formulaDiagnostics = computed(() =>
  draft.formula.trim()
    ? collectFormulaDiagnostics(draft.formula, formulaFields.value, formulaFunctions)
    : [],
);
const formulaSyntaxIssue = computed(
  () => formulaDiagnostics.value.find((diagnostic) => diagnostic.severity === 'error')?.message,
);
const formulaSyntaxErrorLabel = computed(() =>
  formulaSyntaxIssue.value && isCharacterSequenceIssue(formulaSyntaxIssue.value)
    ? '字符错误'
    : '语法错误',
);
const formulaPreview = computed(() => (draft.formula.trim() ? draft.formula : '编辑公式'));
const formulaPreviewSegments = computed(() =>
  formulaSegments(formulaPreview.value, formulaFieldLabelByName.value),
);

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return;
    const source = props.validator ?? createSubmitValidatorDraft();
    Object.assign(draft, structuredClone(source));
    formulaIssue.value = '';
    reminderIssue.value = '';
  },
);

function close(): void {
  emit('update:model-value', false);
}

function save(): void {
  formulaIssue.value = !draft.formula.trim()
    ? '请先设置校验条件'
    : formulaSyntaxIssue.value
      ? `公式${formulaSyntaxErrorLabel.value}：${formulaSyntaxIssue.value}`
      : '';
  reminderIssue.value = draft.remind.trim() ? '' : '请填写不满足条件时的提示文字';
  if (formulaIssue.value || reminderIssue.value) return;
  emit('save', structuredClone(draft));
  close();
}

function appendTemplateField(widgetName: string): void {
  draft.remind += `${draft.remind ? ' ' : ''}\${${widgetName}}`;
  fieldPickerOpen.value = false;
}

function appendFormulaField(widgetName: string): void {
  requestFormulaInsertion(`$${widgetName}#`);
}

function appendFunction(functionSpec: FormulaEditorFunction): void {
  requestFormulaInsertion(`${functionSpec.name}()`, -1);
}

function requestFormulaInsertion(text: string, cursorOffset = 0): void {
  formulaInsertionSequence.value += 1;
  formulaInsertion.value = {
    id: formulaInsertionSequence.value,
    text,
    cursorOffset,
  };
}

function setFailAction(value: string | number | boolean | undefined): void {
  draft.failAction = Number(value) === 1 ? 1 : 0;
}

function copyFormula(): void {
  void navigator.clipboard?.writeText(draft.formula);
}

function closeFormulaEditor(): void {
  formulaDialogOpen.value = false;
  formulaDialogExpanded.value = false;
}

function confirmFormulaEditor(): void {
  // 取消和关闭允许保留未完成输入；确认只接受语法通过的公式。
  if (formulaSyntaxIssue.value) return;
  closeFormulaEditor();
}

/** 连续输入字段或函数、以及输入非法符号，统一归为面向用户的“字符错误”。 */
function isCharacterSequenceIssue(message: string): boolean {
  return message.startsWith('不支持的字符') || message === '此处不应继续出现公式内容';
}

function formulaSegments(
  formula: string,
  fieldLabelByName: ReadonlyMap<string, string>,
): Array<{ text: string; className: string }> {
  return (
    formula.match(/\$[A-Za-z_][A-Za-z0-9_]*#|[A-Z][A-Z0-9_]*(?=\()|[^A-Z$]+|[A-Z$]+/g) ?? []
  ).map((segment) => {
    const fieldMatch = segment.match(/^\$([A-Za-z_][A-Za-z0-9_]*)#$/);
    if (fieldMatch?.[1]) {
      return {
        text: fieldLabelByName.get(fieldMatch[1]) ?? segment,
        className: 'is-field',
      };
    }
    return {
      text: segment,
      className: /^[A-Z][A-Z0-9_]*(?=\()/.test(segment) ? 'is-function' : 'is-text',
    };
  });
}
</script>

<template>
  <el-dialog
    :model-value="props.modelValue"
    append-to-body
    destroy-on-close
    lock-scroll
    :show-close="false"
    class="form-submit-validator-dialog"
    aria-label="数据校验设置"
    @update:model-value="emit('update:model-value', $event)"
  >
    <template #header>
      <header class="form-submit-validator-dialog__header">
        <h2>数据校验设置</h2>
        <button type="button" aria-label="关闭数据校验设置" @click="close">
          <el-icon><RiCloseLine /></el-icon>
        </button>
      </header>
    </template>

    <div class="form-submit-validator-dialog__body">
      <section class="form-submit-validator-dialog__formula-section">
        <h3>校验条件</h3>
        <button
          type="button"
          class="form-submit-validator-dialog__formula-box"
          :class="{ 'is-empty': !draft.formula }"
          @click="formulaDialogOpen = true"
        >
          <span v-if="!draft.formula" class="form-submit-validator-dialog__formula-placeholder">
            <el-icon><RiFunctionLine /></el-icon>
            编辑公式
          </span>
          <code v-else class="form-submit-validator-dialog__formula-preview">
            <span
              v-for="(segment, index) in formulaPreviewSegments"
              :key="`${segment.text}-${index}`"
              :class="segment.className"
              >{{ segment.text }}</span
            >
          </code>
        </button>
        <p v-if="formulaIssue" class="form-submit-validator-dialog__error">{{ formulaIssue }}</p>
      </section>

      <el-form class="form-submit-validator-dialog__form" label-position="top" @submit.prevent>
        <el-form-item required>
          <template #label>不满足校验条件时的提示文字</template>
          <el-input
            v-model="draft.remind"
            :maxlength="500"
            placeholder="请输入提示文字"
            @input="reminderIssue = ''"
          >
            <template #append>
              <el-popover v-model:visible="fieldPickerOpen" :width="360" placement="bottom-end">
                <template #reference>
                  <el-button class="form-submit-validator-dialog__field-add" aria-label="插入字段">
                    <el-icon><RiAddLine /></el-icon>
                  </el-button>
                </template>
                <div class="form-submit-validator-dialog__field-picker" role="listbox">
                  <button
                    v-for="item in fieldItems"
                    :key="item.widget.widgetName"
                    type="button"
                    @click="appendTemplateField(item.widget.widgetName)"
                  >
                    <span>{{ item.label }}</span>
                    <small>{{ item.widget.type }}</small>
                  </button>
                  <p v-if="fieldItems.length === 0">请先在画布中添加字段</p>
                </div>
              </el-popover>
            </template>
          </el-input>
          <p v-if="reminderIssue" class="form-submit-validator-dialog__error">
            {{ reminderIssue }}
          </p>
        </el-form-item>

        <el-checkbox v-model="draft.realtime" class="form-submit-validator-dialog__realtime">
          字段修改时实时校验
          <el-tooltip content="变更校验条件引用字段时，立即提示校验结果" placement="top">
            <el-icon aria-label="实时校验说明"><RiInformationLine /></el-icon>
          </el-tooltip>
        </el-checkbox>

        <section class="form-submit-validator-dialog__failure">
          <h3>不满足校验条件后</h3>
          <el-radio-group :model-value="draft.failAction" @update:model-value="setFailAction">
            <el-radio :value="0">阻止提交</el-radio>
            <el-radio :value="1">允许忽略并继续提交</el-radio>
          </el-radio-group>
        </section>
      </el-form>
    </div>

    <template #footer>
      <footer class="form-submit-validator-dialog__footer">
        <el-button @click="close">取消</el-button>
        <el-button type="primary" @click="save">确定</el-button>
      </footer>
    </template>
  </el-dialog>

  <el-dialog
    v-model="formulaDialogOpen"
    append-to-body
    destroy-on-close
    lock-scroll
    :show-close="false"
    class="form-submit-formula-dialog"
    :class="{ 'is-expanded': formulaDialogExpanded }"
    aria-label="提交校验公式编辑器"
  >
    <template #header>
      <header class="form-submit-formula-dialog__header">
        <div>
          <h2>提交校验</h2>
          <span>使用数学运算符编辑公式</span>
        </div>
        <div class="form-submit-formula-dialog__actions">
          <button
            type="button"
            :aria-label="formulaDialogExpanded ? '还原公式编辑器尺寸' : '展开公式编辑器'"
            @click="formulaDialogExpanded = !formulaDialogExpanded"
          >
            <el-icon><RiFullscreenLine /></el-icon>
          </button>
          <button type="button" aria-label="关闭公式编辑器" @click="closeFormulaEditor">
            <el-icon><RiCloseLine /></el-icon>
          </button>
        </div>
      </header>
    </template>

    <div class="form-submit-formula-dialog__body">
      <section class="form-submit-formula-dialog__editor">
        <header>
          <span>公式&nbsp;=</span>
          <div>
            <button type="button" @click="copyFormula">
              <el-icon><RiFileCopyLine /></el-icon>复制
            </button>
            <button type="button">
              <el-icon><RiInformationLine /></el-icon>备注
            </button>
          </div>
        </header>
        <FormulaEditor
          v-model="draft.formula"
          :fields="formulaFields"
          :functions="formulaFunctions"
          :insertion="formulaInsertion"
        />
      </section>

      <p
        v-if="formulaSyntaxIssue"
        class="form-submit-formula-dialog__syntax-error"
        role="alert"
        aria-live="polite"
      >
        <el-icon><RiErrorWarningLine /></el-icon>
        <strong>{{ formulaSyntaxErrorLabel }}</strong>
        <span class="form-submit-formula-dialog__syntax-error-detail">
          {{ formulaSyntaxIssue }}
        </span>
      </p>

      <section class="form-submit-formula-dialog__library">
        <div class="form-submit-formula-dialog__fields">
          <div class="form-submit-formula-dialog__search">
            <el-icon><RiSearchLine /></el-icon>搜索变量
          </div>
          <button
            v-for="field in formulaFields"
            :key="field.widgetName"
            type="button"
            :disabled="!field.formulaAllowed"
            :title="field.formulaAllowed ? undefined : '该字段类型暂不支持参与公式计算'"
            @click="appendFormulaField(field.widgetName)"
          >
            <span>{{ field.label }}</span>
            <small>{{ formulaFieldTypeByName.get(field.widgetName) }}</small>
          </button>
        </div>
        <FormulaFunctionLibrary
          class="form-submit-formula-dialog__function-library"
          :functions="formulaFunctions"
          @insert="appendFunction"
        />
      </section>
    </div>
    <template #footer>
      <footer class="form-submit-formula-dialog__footer">
        <el-button @click="closeFormulaEditor">取消</el-button>
        <el-button
          type="primary"
          :disabled="Boolean(formulaSyntaxIssue)"
          @click="confirmFormulaEditor"
          >确定</el-button
        >
      </footer>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.form-submit-validator-dialog {
  width: min(840px, calc(100vw - 40px)) !important;
  height: min(520px, calc(100dvh - 24px));
  max-height: calc(100dvh - 24px);
  margin: 12px auto !important;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;

  .el-dialog__header {
    padding: 0;
    margin: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
  .el-dialog__body {
    padding: 0;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }
  .el-dialog__footer {
    padding: 0;
    border-top: 1px solid var(--el-border-color);
  }

  &__header {
    display: flex;
    height: 56px;
    padding: 0 26px;
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    font-size: 19px;
    font-weight: 700;
    letter-spacing: -0.3px;
  }
  &__header button,
  .form-submit-formula-dialog__header button {
    display: inline-grid;
    width: 30px;
    height: 30px;
    padding: 0;
    place-items: center;
    font-size: 21px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 8px;
  }
  &__header button:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
  &__body {
    height: 100%;
    padding: 20px 26px;
    box-sizing: border-box;
    overflow: hidden;
  }
  &__formula-section h3,
  &__failure h3 {
    margin: 0 0 9px;
    font-size: 16px;
    font-weight: 650;
    color: var(--el-text-color-primary);
  }
  &__formula-box {
    display: grid;
    width: 100%;
    min-height: 104px;
    padding: 16px;
    place-items: center;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-fill-color);
    border-radius: 10px;
  }
  &__formula-box:hover,
  &__formula-box:focus-visible {
    border-color: var(--el-color-primary-light-5);
    outline: none;
  }
  &__formula-placeholder {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    font-size: 15px;
  }
  &__formula-placeholder .el-icon {
    font-size: 18px;
    color: var(--el-color-primary);
  }
  &__formula-preview {
    display: block;
    width: 100%;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 14px;
    line-height: 1.6;
    text-align: left;
    overflow-wrap: anywhere;
  }
  &__formula-preview .is-field {
    padding: 2px 7px;
    margin: 0 2px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 4px;
  }
  &__formula-preview .is-function {
    color: #bb42db;
  }
  &__formula-preview .is-text {
    color: var(--el-text-color-primary);
  }
  &__form {
    margin-top: 16px;
  }
  &__form .el-form-item {
    margin-bottom: 14px;
  }
  &__form .el-form-item__label {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  &__field-add.el-button {
    height: 30px;
    padding: 0 8px;
    margin: 0 -9px 0 0;
    color: var(--el-color-primary);
    border: 0;
  }
  &__field-picker {
    max-height: 300px;
    overflow-y: auto;
  }
  &__field-picker button {
    display: flex;
    width: 100%;
    min-height: 42px;
    padding: 0 8px;
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
    padding: 2px 8px;
    font-size: 12px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 99px;
  }
  &__field-picker p {
    margin: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
  &__realtime {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    margin-bottom: 16px;
    font-size: 14px;
  }
  &__realtime .el-icon {
    color: var(--el-text-color-secondary);
  }
  &__failure .el-radio-group {
    display: grid;
    gap: 9px;
  }
  &__failure .el-radio {
    height: auto;
    margin-right: 0;
    font-size: 14px;
  }
  &__footer,
  .form-submit-formula-dialog__footer {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
    padding: 10px 26px;
  }
  &__footer .el-button {
    min-width: 72px;
    height: 34px;
    font-size: 14px;
  }
  &__error {
    margin: 7px 0 0;
    font-size: 13px;
    color: var(--el-color-danger);
  }
}

.form-submit-formula-dialog {
  // 弹窗本身始终落在视口内；长列表仅在各自分栏内滚动，不能撑出页面滚动条。
  width: min(920px, calc(100vw - 40px)) !important;
  height: min(560px, calc(100dvh - 24px));
  max-height: calc(100dvh - 24px);
  margin: 12px auto !important;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;

  &.is-expanded {
    width: min(1280px, calc(100vw - 48px)) !important;
    height: min(760px, calc(100dvh - 24px));
  }

  .el-dialog__header {
    padding: 0;
    margin: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
  .el-dialog__body {
    padding: 0;
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }
  .el-dialog__footer {
    padding: 0;
    border-top: 1px solid var(--el-border-color);
  }
  &__header {
    display: flex;
    height: 56px;
    padding: 0 20px;
    align-items: center;
    justify-content: space-between;
  }
  &__header div {
    display: flex;
    gap: 14px;
    align-items: baseline;
  }
  &__header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 700;
  }
  &__header span {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  &__actions {
    display: inline-flex;
    gap: 4px;
    align-items: center;
  }
  &__actions button {
    display: inline-grid;
    width: 28px;
    height: 28px;
    padding: 0;
    place-items: center;
    font-size: 18px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 6px;
  }
  &__actions button:hover,
  &__actions button:focus-visible {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
    outline: none;
  }
  &__body {
    height: 100%;
    padding: 16px 20px;
    box-sizing: border-box;
    overflow: hidden;
  }
  &__editor {
    overflow: hidden;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
  }
  &__editor > header {
    display: flex;
    height: 44px;
    padding: 0 14px;
    align-items: center;
    justify-content: space-between;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  &__editor > header > span {
    font-size: 15px;
  }
  &__editor > header div {
    display: flex;
    gap: 12px;
  }
  &__editor > header button {
    display: inline-flex;
    gap: 5px;
    align-items: center;
    padding: 0;
    font: inherit;
    font-size: 13px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
  }
  &__editor > header button:hover {
    color: var(--el-color-primary);
  }
  &__editor .el-textarea__inner {
    min-height: 112px !important;
    padding: 12px 14px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 14px;
    line-height: 1.65;
    border: 0;
    box-shadow: none;
    resize: none;
  }
  &__library {
    display: grid;
    grid-template-columns: 1.15fr 2.2fr;
    height: 216px;
    min-height: 0;
    margin-top: 10px;
    overflow: hidden;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
  }
  &__syntax-error {
    display: flex;
    min-height: 34px;
    gap: 7px;
    padding: 0 12px;
    margin: 8px 0 0;
    align-items: center;
    box-sizing: border-box;
    font-size: 13px;
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
    border: 1px solid var(--el-color-danger-light-7);
    border-radius: 7px;
  }
  &__syntax-error .el-icon {
    flex: 0 0 auto;
    font-size: 16px;
  }
  &__syntax-error strong {
    flex: 0 0 auto;
    font-weight: 600;
  }
  &__syntax-error-detail {
    color: var(--el-text-color-secondary);
  }
  // 标准尺寸在出现横幅后收缩函数库高度，始终由内部区域消化内容高度。
  &:not(.is-expanded) &__syntax-error + &__library {
    height: 202px;
  }
  &__fields {
    overflow: hidden auto;
    max-height: 216px;
    border-right: 1px solid var(--el-border-color);
  }
  &__search {
    position: sticky;
    top: 0;
    z-index: 2;
    display: flex;
    gap: 8px;
    height: 42px;
    padding: 0 14px;
    align-items: center;
    font-size: 13px;
    color: var(--el-text-color-placeholder);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  &__fields button {
    display: grid;
    width: 100%;
    min-height: 40px;
    gap: 2px;
    padding: 5px 14px;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
  }
  &__fields button:hover {
    background: var(--el-fill-color-light);
  }
  &__fields small {
    justify-self: start;
    padding: 1px 7px;
    font-size: 11px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 99px;
  }
  &__function-library {
    min-width: 0;
  }
  &__footer {
    padding: 10px 20px;
  }
  &__footer .el-button {
    min-width: 72px;
    height: 34px;
    font-size: 14px;
  }
}

// Element Plus 的 overlay 容器默认可滚动。两个弹窗的长内容都在内部区域处理，
// 因此显式收口外层滚动，避免滚轮带动被遮罩的设计器页面。
.el-overlay-dialog:has(.form-submit-validator-dialog),
.el-overlay-dialog:has(.form-submit-formula-dialog) {
  overflow: hidden;
}

@media (width <= 760px) {
  .form-submit-validator-dialog {
    width: calc(100vw - 24px) !important;
    height: calc(100dvh - 16px);
    margin: 8px auto !important;
  }
  .form-submit-validator-dialog__header,
  .form-submit-validator-dialog__body {
    padding-right: 20px;
    padding-left: 20px;
  }
  .form-submit-formula-dialog {
    width: calc(100vw - 16px) !important;
    height: calc(100dvh - 16px);
    margin: 8px auto !important;
  }
  .form-submit-formula-dialog__header {
    height: auto;
    min-height: 54px;
    padding: 10px 14px;
  }
  .form-submit-formula-dialog__header div {
    display: grid;
    gap: 3px;
  }
  .form-submit-formula-dialog__body {
    padding: 18px;
  }
  .form-submit-formula-dialog__library {
    grid-template-columns: 1fr;
  }
  .form-submit-formula-dialog__fields {
    max-height: 180px;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
}
</style>
