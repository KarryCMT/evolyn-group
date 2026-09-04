import {
  autocompletion,
  closeBrackets,
  pickedCompletion,
  type Completion,
  type CompletionContext,
} from '@codemirror/autocomplete';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { bracketMatching } from '@codemirror/language';
import { linter } from '@codemirror/lint';
import type { Extension } from '@codemirror/state';
import {
  Decoration,
  EditorView,
  keymap,
  ViewPlugin,
  WidgetType,
  type ViewUpdate,
} from '@codemirror/view';
import { collectFormulaDiagnostics } from '../formula/analyzer';
import type { FormulaEditorField, FormulaEditorFunction } from '../formula/types';

export { collectFormulaDiagnostics } from '../formula/analyzer';
export { FORMULA_FUNCTIONS } from '../formula/catalog';
export type {
  FormulaEditorField,
  FormulaEditorFunction,
  FormulaEditorInsertion,
} from '../formula/types';

/**
 * 组装公式 DSL 的编辑器扩展。该层只负责输入体验与前端预校验，
 * 最终公式语义仍必须以统一的服务端校验器为准。
 */
export function createFormulaEditorExtensions(options: {
  fields: readonly FormulaEditorField[];
  functions: readonly FormulaEditorFunction[];
}): Extension {
  return [
    autocompletion({
      activateOnTyping: true,
      override: [createFormulaCompletionSource(options.fields, options.functions)],
    }),
    linter(
      (view) =>
        collectFormulaDiagnostics(view.state.doc.toString(), options.fields, options.functions),
      { delay: 180 },
    ),
    createFormulaTokenHighlighter(options.fields),
  ];
}

/** 基础编辑器能力与公式 DSL 扩展解耦，便于后续复用到默认值公式等场景。 */
export function createFormulaEditorBaseExtensions(onUpdate: (value: string) => void): Extension {
  return [
    history(),
    keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
    EditorView.lineWrapping,
    closeBrackets(),
    bracketMatching(),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) onUpdate(update.state.doc.toString());
    }),
  ];
}

function createFormulaCompletionSource(
  fields: readonly FormulaEditorField[],
  functions: readonly FormulaEditorFunction[],
) {
  const options: Completion[] = [
    ...fields.map((field) => ({
      label: field.label,
      detail: `字段 · ${field.displayType ?? field.valueType}`,
      type: 'variable',
      apply: `$${field.widgetName}#`,
    })),
    ...functions.map(
      (item): Completion => ({
        label: item.name,
        detail: item.description,
        type: 'function',
        apply: (view, completion, from, to) => {
          const text = `${item.name}()`;
          view.dispatch({
            changes: { from, to, insert: text },
            selection: { anchor: from + item.name.length + 1 },
            annotations: pickedCompletion.of(completion),
          });
        },
      }),
    ),
  ];

  return (context: CompletionContext) => {
    const token = context.matchBefore(/[\$A-Za-z_][A-Za-z0-9_$]*/);
    if (!context.explicit && !token) return null;
    return { from: token?.from ?? context.pos, options };
  };
}

function createFormulaDecorations(view: EditorView, fieldLabelByName: ReadonlyMap<string, string>) {
  const source = view.state.doc.toString();
  const ranges = [...source.matchAll(/\$([A-Za-z_][A-Za-z0-9_]*)#|[A-Z][A-Z0-9_]*(?=\()/g)].map(
    (match) => {
      const from = match.index ?? 0;
      const widgetName = match[1];
      const label = widgetName ? fieldLabelByName.get(widgetName) : undefined;
      if (label) {
        return Decoration.replace({ widget: new FormulaFieldWidget(label) }).range(
          from,
          from + match[0].length,
        );
      }
      const className = widgetName ? 'cm-formula-field' : 'cm-formula-function';
      return Decoration.mark({ class: className }).range(from, from + match[0].length);
    },
  );
  return Decoration.set(ranges, true);
}

function createFormulaTokenHighlighter(fields: readonly FormulaEditorField[]): Extension {
  const fieldLabelByName = new Map(fields.map((field) => [field.widgetName, field.label]));
  return ViewPlugin.fromClass(
    class {
      decorations: ReturnType<typeof createFormulaDecorations>;

      constructor(view: EditorView) {
        this.decorations = createFormulaDecorations(view, fieldLabelByName);
      }

      update(update: ViewUpdate): void {
        if (update.docChanged) {
          this.decorations = createFormulaDecorations(update.view, fieldLabelByName);
        }
      }
    },
    {
      decorations: (plugin) => plugin.decorations,
      provide: (plugin) =>
        EditorView.atomicRanges.of((view) => view.plugin(plugin)?.decorations ?? Decoration.none),
    },
  );
}

/**
 * 公式正文仍保存稳定 key；仅将其替换成可读标签的原子化展示，
 * 因此删除或复制时依旧作用于完整的 `$widgetName#` 引用。
 */
class FormulaFieldWidget extends WidgetType {
  constructor(private readonly label: string) {
    super();
  }

  override eq(other: FormulaFieldWidget): boolean {
    return this.label === other.label;
  }

  override toDOM(): HTMLElement {
    const element = document.createElement('span');
    element.className = 'cm-formula-field-chip';
    element.textContent = this.label;
    element.title = `字段：${this.label}`;
    element.setAttribute('aria-label', `字段 ${this.label}`);
    return element;
  }
}
