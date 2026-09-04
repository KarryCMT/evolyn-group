<script setup lang="ts">
import { Compartment, EditorSelection, EditorState } from '@codemirror/state';
import { EditorView, placeholder } from '@codemirror/view';
import { onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import {
  createFormulaEditorBaseExtensions,
  createFormulaEditorExtensions,
  type FormulaEditorField,
  type FormulaEditorFunction,
  type FormulaEditorInsertion,
} from './formula-editor';

const formula = defineModel<string>({ required: true });
const props = withDefaults(
  defineProps<{
    fields: readonly FormulaEditorField[];
    functions: readonly FormulaEditorFunction[];
    insertion?: FormulaEditorInsertion;
    placeholder?: string;
  }>(),
  { insertion: undefined, placeholder: '请输入公式，或从下方选择字段和函数' },
);

const editorHost = useTemplateRef<HTMLDivElement>('editorHost');
const editorView = shallowRef<EditorView>();
const featureCompartment = new Compartment();
const placeholderCompartment = new Compartment();

function createFeatures() {
  return createFormulaEditorExtensions({ fields: props.fields, functions: props.functions });
}

function createPlaceholder() {
  return placeholder(props.placeholder);
}

function syncExternalFormula(nextFormula: string): void {
  const view = editorView.value;
  if (!view || view.state.doc.toString() === nextFormula) return;
  const cursor = Math.min(view.state.selection.main.head, nextFormula.length);
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: nextFormula },
    selection: EditorSelection.cursor(cursor),
  });
}

function applyInsertion(insertion: FormulaEditorInsertion | undefined): void {
  const view = editorView.value;
  if (!view || !insertion) return;
  const selection = view.state.selection.main;
  const cursor = Math.max(
    0,
    selection.from + insertion.text.length + (insertion.cursorOffset ?? 0),
  );
  view.dispatch({
    changes: { from: selection.from, to: selection.to, insert: insertion.text },
    selection: EditorSelection.cursor(cursor),
    scrollIntoView: true,
  });
  view.focus();
}

onMounted(() => {
  if (!editorHost.value) return;
  const state = EditorState.create({
    doc: formula.value,
    extensions: [
      createFormulaEditorBaseExtensions((nextFormula) => {
        formula.value = nextFormula;
      }),
      featureCompartment.of(createFeatures()),
      placeholderCompartment.of(createPlaceholder()),
    ],
  });
  editorView.value = new EditorView({ state, parent: editorHost.value });
});

onBeforeUnmount(() => editorView.value?.destroy());

watch(formula, syncExternalFormula);
watch(
  () => [props.fields, props.functions],
  () => editorView.value?.dispatch({ effects: featureCompartment.reconfigure(createFeatures()) }),
);
watch(
  () => props.placeholder,
  () =>
    editorView.value?.dispatch({
      effects: placeholderCompartment.reconfigure(createPlaceholder()),
    }),
);
watch(
  () => props.insertion?.id,
  () => applyInsertion(props.insertion),
);
</script>

<template>
  <div class="formula-editor" aria-label="公式编辑区">
    <div ref="editorHost" class="formula-editor__host" />
  </div>
</template>

<style scoped lang="scss">
.formula-editor {
  min-height: 112px;
  background: var(--el-bg-color);
}

.formula-editor__host {
  min-height: inherit;
}

.formula-editor__host :deep(.cm-editor) {
  min-height: inherit;
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 14px;
  line-height: 1.65;
}

.formula-editor__host :deep(.cm-scroller) {
  min-height: inherit;
  max-height: 112px;
  overflow: auto;
  font-family: inherit;
}

.formula-editor__host :deep(.cm-content) {
  min-height: 112px;
  padding: 12px 14px;
  caret-color: var(--el-color-primary);
}

.formula-editor__host :deep(.cm-focused) {
  outline: none;
}

.formula-editor__host :deep(.cm-selectionBackground),
.formula-editor__host :deep(::selection) {
  background: var(--el-color-primary-light-8) !important;
}

.formula-editor__host :deep(.cm-tooltip) {
  z-index: 3;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-light);
}

.formula-editor__host :deep(.cm-tooltip-autocomplete > ul > li) {
  padding: 7px 10px;
}

.formula-editor__host :deep(.cm-tooltip-autocomplete > ul > li[aria-selected]) {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.formula-editor__host :deep(.cm-formula-field),
.formula-editor__host :deep(.cm-formula-field-chip) {
  display: inline-block;
  padding: 2px 4px;
  margin: 0 1px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
}

.formula-editor__host :deep(.cm-formula-function) {
  color: #bb42db;
  font-weight: 600;
}

.formula-editor__host :deep(.cm-diagnostic) {
  padding-bottom: 1px;
}
</style>
