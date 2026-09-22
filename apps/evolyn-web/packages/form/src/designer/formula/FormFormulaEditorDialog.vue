<script setup lang="ts">
import {
  RiCloseLine,
  RiFileCopyLine,
  RiFullscreenExitLine,
  RiFullscreenLine,
} from '@remixicon/vue';
import { ElButton, ElDialog, ElIcon, ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import type { FormulaEditorField, FormulaEditorFunction, FormulaEditorInsertion } from '../../formula';
import FormulaFunctionLibrary from '../../formula/FormulaFunctionLibrary.vue';
import FormulaEditor from '../FormulaEditor.vue';
import FormulaVariablePanel from './FormulaVariablePanel.vue';

type FormulaVariableOption = FormulaEditorField & {
  listKey?: string;
  disabled?: boolean;
  disabledReason?: string;
};

const open = defineModel<boolean>({ required: true });
const formula = defineModel<string>('formula', { required: true });
const props = withDefaults(
  defineProps<{
    title: string;
    subtitle: string;
    editorLabel: string;
    fields: readonly FormulaEditorField[];
    functions: readonly FormulaEditorFunction[];
    variableFields?: readonly FormulaVariableOption[];
    placeholder?: string;
    error?: string;
    errorLabel?: string;
    confirmDisabled?: boolean;
    footerHint?: string;
    ariaLabel?: string;
  }>(),
  {
    variableFields: undefined,
    placeholder: '请输入公式，或从下方选择字段和函数',
    error: '',
    errorLabel: '语法错误',
    confirmDisabled: false,
    footerHint: '',
    ariaLabel: '公式编辑器',
  },
);

const emit = defineEmits<{
  cancel: [];
  confirm: [];
  closed: [];
}>();

const expanded = shallowRef(false);
const insertion = shallowRef<FormulaEditorInsertion>();
const insertionSequence = shallowRef(0);

function requestInsertion(text: string, cursorOffset = 0): void {
  insertionSequence.value += 1;
  insertion.value = { id: insertionSequence.value, text, cursorOffset };
}

function insertField(field: FormulaEditorField): void {
  requestInsertion(`$${field.widgetName}#`);
}

function insertFunction(entry: FormulaEditorFunction): void {
  requestInsertion(`${entry.name}()`, -1);
}

function copyFormula(): void {
  void navigator.clipboard?.writeText(formula.value);
  ElMessage.success('公式已复制');
}

function cancel(): void {
  emit('cancel');
  open.value = false;
}

function confirm(): void {
  emit('confirm');
}

function handleClosed(): void {
  expanded.value = false;
  emit('closed');
}
</script>

<template>
  <ElDialog
    v-model="open"
    append-to-body
    destroy-on-close
    lock-scroll
    width="min(1600px, 90vw)"
    :show-close="false"
    class="form-formula-dialog"
    :class="{ 'is-expanded': expanded }"
    :aria-label="props.ariaLabel"
    @closed="handleClosed"
  >
    <template #header>
      <header class="form-formula-dialog__header">
        <div class="form-formula-dialog__heading">
          <h2>{{ props.title }}</h2>
          <span>{{ props.subtitle }}</span>
        </div>
        <div class="form-formula-dialog__header-actions">
          <button
            type="button"
            :aria-label="expanded ? '退出全屏' : '全屏'"
            @click="expanded = !expanded"
          >
            <el-icon><RiFullscreenExitLine v-if="expanded" /><RiFullscreenLine v-else /></el-icon>
          </button>
          <button type="button" aria-label="关闭公式编辑器" @click="cancel">
            <el-icon><RiCloseLine /></el-icon>
          </button>
        </div>
      </header>
    </template>

    <div class="form-formula-dialog__workspace">
      <section class="form-formula-dialog__editor-shell">
        <header class="form-formula-dialog__editor-header">
          <strong>{{ props.editorLabel }} =</strong>
          <div class="form-formula-dialog__toolbar">
            <button type="button" @click="copyFormula">
              <el-icon><RiFileCopyLine /></el-icon>复制
            </button>
            <slot name="toolbar" />
          </div>
        </header>
        <slot name="editor-extra" />
        <FormulaEditor
          v-model="formula"
          :fields="props.fields"
          :functions="props.functions"
          :insertion="insertion"
          :placeholder="props.placeholder"
        />
        <p v-if="props.error" class="form-formula-dialog__error" role="alert" aria-live="polite">
          <strong>{{ props.errorLabel }}</strong>
          <span>{{ props.error }}</span>
        </p>
      </section>

      <section class="form-formula-dialog__library">
        <FormulaVariablePanel
          :fields="props.variableFields ?? props.fields"
          @insert="insertField"
        />
        <FormulaFunctionLibrary
          class="form-formula-dialog__functions"
          :functions="props.functions"
          @insert="insertFunction"
        />
      </section>
    </div>

    <template #footer>
      <footer class="form-formula-dialog__footer">
        <span>{{ props.footerHint }}</span>
        <div>
          <el-button @click="cancel">取消</el-button>
          <el-button type="primary" :disabled="props.confirmDisabled" @click="confirm">
            确定
          </el-button>
        </div>
      </footer>
    </template>
  </ElDialog>
</template>

<style lang="scss">
.form-formula-dialog {
  // 公式编辑器是设计器中的工作台级弹窗：宽度统一按视口计算，避免业务弹窗
  // 各自回退到 Element Plus 默认 50% 宽度而挤压函数说明栏。
  width: min(1600px, 90vw) !important;
  height: min(900px, calc(100dvh - 48px));
  max-height: calc(100dvh - 48px);
  margin: 24px auto !important;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 14px;

  &.is-expanded {
    width: calc(100vw - 32px) !important;
    height: calc(100dvh - 32px);
    max-height: calc(100dvh - 32px);
    margin: 16px auto !important;
  }

  .el-dialog__header,
  .el-dialog__body,
  .el-dialog__footer {
    padding: 0;
    margin: 0;
  }

  .el-dialog__body {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }

  &__header {
    display: flex;
    min-height: 72px;
    padding: 0 28px;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__heading {
    display: flex;
    gap: 18px;
    align-items: baseline;
  }

  &__heading h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 22px;
    letter-spacing: -0.02em;
  }

  &__heading span,
  &__footer > span {
    color: var(--el-text-color-secondary);
  }

  &__header-actions,
  &__toolbar,
  &__footer > div {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  &__header-actions button,
  &__toolbar button {
    display: inline-flex;
    gap: 5px;
    align-items: center;
    padding: 0;
    color: var(--el-text-color-regular);
    font: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  &__header-actions button {
    display: inline-grid;
    width: 32px;
    height: 32px;
    place-items: center;
    border-radius: 7px;
  }

  &__header-actions button:hover,
  &__header-actions button:focus-visible,
  &__toolbar button:hover,
  &__toolbar button:focus-visible {
    color: var(--el-color-primary);
    outline: none;
  }

  &__header-actions button:hover,
  &__header-actions button:focus-visible {
    background: var(--el-fill-color-light);
  }

  &__workspace {
    display: flex;
    height: 100%;
    min-height: 0;
    padding: 28px 32px 24px;
    flex-direction: column;
    background: var(--el-fill-color-extra-light);
  }

  &__editor-shell,
  &__library {
    min-width: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
  }

  &__editor-shell {
    flex: 1 1 auto;
    min-height: 260px;
    border-radius: 9px 9px 0 0;
  }

  &__editor-header {
    display: flex;
    min-height: 48px;
    padding: 0 16px;
    align-items: center;
    justify-content: space-between;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__editor-shell .formula-editor,
  &__editor-shell .formula-editor__host,
  &__editor-shell .cm-editor,
  &__editor-shell .cm-content {
    min-height: 250px;
  }

  &__editor-shell .cm-scroller {
    min-height: 210px;
    max-height: calc(100dvh - 590px);
  }

  &__error {
    display: flex;
    gap: 8px;
    padding: 8px 14px;
    margin: 0;
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
    border-top: 1px solid var(--el-color-danger-light-7);
  }

  &__error span {
    color: var(--el-text-color-secondary);
  }

  &__library {
    display: grid;
    grid-template-columns: minmax(240px, 320px) minmax(0, 1fr);
    flex: 0 0 278px;
    min-height: 0;
    border-top: 0;
    border-radius: 0 0 9px 9px;
  }

  &__functions.formula-function-library {
    min-width: 0;
    grid-template-columns: minmax(260px, 0.8fr) minmax(0, 1.2fr);
  }

  &__functions .formula-function-library__groups,
  &__functions .formula-function-library__guide {
    min-width: 0;
  }

  &__functions .formula-function-library__guide {
    overflow: auto;
  }

  &__footer {
    display: flex;
    min-height: 76px;
    padding: 0 32px;
    align-items: center;
    justify-content: space-between;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  &__footer .el-button {
    min-width: 72px;
  }
}

.el-overlay-dialog:has(.form-formula-dialog) {
  overflow: hidden;
}

@media (width <= 960px) {
  .form-formula-dialog {
    width: calc(100vw - 24px) !important;
    height: calc(100dvh - 24px);
    max-height: calc(100dvh - 24px);
    margin: 12px auto !important;

    &__workspace {
      padding: 18px;
    }

    &__library {
      grid-template-columns: 1fr;
      flex-basis: 300px;
    }

    .formula-variable-panel {
      display: none;
    }
  }
}
</style>
