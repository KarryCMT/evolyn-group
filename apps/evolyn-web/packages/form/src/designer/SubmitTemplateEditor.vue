<script setup lang="ts">
import { Compartment, EditorSelection, EditorState, type Extension } from '@codemirror/state';
import {
  Decoration,
  EditorView,
  ViewPlugin,
  WidgetType,
  placeholder,
  type ViewUpdate,
} from '@codemirror/view';
import { onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import {
  createFormulaEditorBaseExtensions,
  type FormulaEditorInsertion,
} from './formula-editor';

export interface SubmitTemplateEditorField {
  widgetName: string;
  label: string;
}

const template = defineModel<string>({ required: true });
const props = withDefaults(
  defineProps<{
    fields: readonly SubmitTemplateEditorField[];
    insertion?: FormulaEditorInsertion;
    placeholder?: string;
    maxLength?: number;
  }>(),
  { insertion: undefined, placeholder: '请输入提示文字，支持插入字段变量', maxLength: 500 },
);

const editorHost = useTemplateRef<HTMLDivElement>('editorHost');
const editorView = shallowRef<EditorView>();
const tokenCompartment = new Compartment();

function createTokenExtensions(): Extension {
  return createTemplateTokenHighlighter(props.fields);
}

function syncExternalTemplate(nextTemplate: string): void {
  const view = editorView.value;
  if (!view || view.state.doc.toString() === nextTemplate) return;
  const cursor = Math.min(view.state.selection.main.head, nextTemplate.length);
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: nextTemplate },
    selection: EditorSelection.cursor(cursor),
  });
}

function applyInsertion(insertion: FormulaEditorInsertion | undefined): void {
  const view = editorView.value;
  if (!view || !insertion) return;
  const selection = view.state.selection.main;
  const cursor = Math.max(0, selection.from + insertion.text.length + (insertion.cursorOffset ?? 0));
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
    doc: template.value,
    extensions: [
      createFormulaEditorBaseExtensions((nextTemplate) => {
        template.value = nextTemplate;
      }),
      EditorState.transactionFilter.of((transaction) =>
        transaction.newDoc.length <= props.maxLength ? transaction : [],
      ),
      tokenCompartment.of(createTokenExtensions()),
      placeholder(props.placeholder),
    ],
  });
  editorView.value = new EditorView({ state, parent: editorHost.value });
});

onBeforeUnmount(() => editorView.value?.destroy());

watch(template, syncExternalTemplate);
watch(
  () => props.fields,
  () => editorView.value?.dispatch({ effects: tokenCompartment.reconfigure(createTokenExtensions()) }),
);
watch(
  () => props.insertion?.id,
  () => applyInsertion(props.insertion),
);

function createTemplateTokenHighlighter(fields: readonly SubmitTemplateEditorField[]): Extension {
  const labels = new Map(fields.map((field) => [field.widgetName, field.label]));
  return ViewPlugin.fromClass(
    class {
      decorations: ReturnType<typeof createTemplateDecorations>;

      constructor(view: EditorView) {
        this.decorations = createTemplateDecorations(view, labels);
      }

      update(update: ViewUpdate): void {
        if (update.docChanged) this.decorations = createTemplateDecorations(update.view, labels);
      }
    },
    {
      decorations: (plugin) => plugin.decorations,
      provide: (plugin) =>
        EditorView.atomicRanges.of((view) => view.plugin(plugin)?.decorations ?? Decoration.none),
    },
  );
}

function createTemplateDecorations(view: EditorView, labels: ReadonlyMap<string, string>) {
  const source = view.state.doc.toString();
  const ranges = [...source.matchAll(/\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g)].map((match) => {
    const field = match[1];
    const label = field ? labels.get(field) : undefined;
    const from = match.index ?? 0;
    return label
      ? Decoration.replace({ widget: new TemplateFieldWidget(label) }).range(from, from + match[0].length)
      : Decoration.mark({ class: 'cm-submit-template-unknown' }).range(
          from,
          from + match[0].length,
        );
  });
  return Decoration.set(ranges, true);
}

/** 模板源码始终保留 `${widgetName}`，仅以原子标签改善设计器阅读与编辑体验。 */
class TemplateFieldWidget extends WidgetType {
  constructor(private readonly label: string) {
    super();
  }

  override eq(other: TemplateFieldWidget): boolean {
    return this.label === other.label;
  }

  override toDOM(): HTMLElement {
    const element = document.createElement('span');
    element.className = 'cm-submit-template-field';
    element.textContent = this.label;
    element.title = `字段：${this.label}`;
    element.setAttribute('aria-label', `字段 ${this.label}`);
    return element;
  }
}
</script>

<template>
  <div class="submit-template-editor" aria-label="提示文字编辑区">
    <div ref="editorHost" class="submit-template-editor__host" />
  </div>
</template>

<style scoped lang="scss">
.submit-template-editor {
  min-width: 0;
  min-height: 38px;
}

.submit-template-editor__host {
  min-height: inherit;
}

.submit-template-editor__host :deep(.cm-editor) {
  min-height: inherit;
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 14px;
  line-height: 1.5;
}

.submit-template-editor__host :deep(.cm-scroller) {
  min-height: inherit;
  overflow: auto;
  font-family: inherit;
}

.submit-template-editor__host :deep(.cm-content) {
  min-height: 38px;
  padding: 8px 11px;
  caret-color: var(--el-color-primary);
}

.submit-template-editor__host :deep(.cm-focused) {
  outline: none;
}

.submit-template-editor__host :deep(.cm-selectionBackground),
.submit-template-editor__host :deep(::selection) {
  background: var(--el-color-primary-light-8) !important;
}

.submit-template-editor__host :deep(.cm-submit-template-field) {
  display: inline-block;
  padding: 1px 5px;
  margin: 0 1px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
}

.submit-template-editor__host :deep(.cm-submit-template-unknown) {
  color: var(--el-color-danger);
  text-decoration: wavy underline;
}
</style>
