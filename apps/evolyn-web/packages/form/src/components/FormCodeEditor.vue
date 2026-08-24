<template>
  <div ref="editorRef" class="plugin-code-editor"></div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
// 注册 Monaco 默认编辑器功能，包含右键菜单、格式化、剪切复制粘贴等命令。
import 'monaco-editor/esm/vs/editor/editor.all';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import TsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution';
import 'monaco-editor/esm/vs/basic-languages/java/java.contribution';
import 'monaco-editor/esm/vs/basic-languages/python/python.contribution';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import 'monaco-editor/esm/vs/language/typescript/monaco.contribution';
import type { PluginCodeDiagnostic } from '../types';

type MonacoEditorLanguage = 'javascript' | 'python' | 'java' | 'json' | 'plaintext';
type EditorLanguage = MonacoEditorLanguage | 'nodejs' | 'node.js' | 'python3';
type CompletionState = {
  count: number;
  disposables: monaco.IDisposable[];
};

const props = withDefaults(
  defineProps<{
    modelValue: string;
    language?: EditorLanguage;
    readOnly?: boolean;
    fontSize?: number;
    hideHorizontalScrollbar?: boolean;
    diagnostics?: PluginCodeDiagnostic[];
    diagnosticFocusKey?: number;
  }>(),
  {
    language: 'javascript',
    readOnly: false,
    fontSize: 16,
    hideHorizontalScrollbar: false,
    diagnostics: () => [],
    diagnosticFocusKey: 0,
  },
);

const emits = defineEmits<{
  (event: 'update:modelValue', value: string): void;
}>();

const editorRef = ref<HTMLDivElement>();
const editorInstance = shallowRef<monaco.editor.IStandaloneCodeEditor>();
const editorModel = shallowRef<monaco.editor.ITextModel>();
const markerOwner = 'plugin-code-editor';

const normalizeEditorLanguage = (language?: EditorLanguage): MonacoEditorLanguage => {
  // Monaco 只识别 javascript/python 等语言 id，运行时枚举需要先归一后再创建 model。
  if (language === 'nodejs' || language === 'node.js') return 'javascript';
  if (language === 'python3') return 'python';
  return language || 'javascript';
};

const getCompletionState = () => {
  const host = self as typeof self & {
    __pluginCodeEditorCompletionState?: CompletionState;
  };
  if (!host.__pluginCodeEditorCompletionState) {
    host.__pluginCodeEditorCompletionState = {
      count: 0,
      disposables: [],
    };
  }
  return host.__pluginCodeEditorCompletionState;
};

self.MonacoEnvironment = {
  getWorker(_: string, label: string) {
    if (label === 'json') return new JsonWorker();
    if (['typescript', 'javascript'].includes(label)) return new TsWorker();
    return new EditorWorker();
  },
};

type CompletionOption = {
  label: string;
  insertText: string;
  detail: string;
  documentation?: string;
  kind: monaco.languages.CompletionItemKind;
  insertTextRules?: monaco.languages.CompletionItemInsertTextRule;
};

const runtimeCompletionMap: Record<MonacoEditorLanguage, CompletionOption[]> = {
  javascript: [
    {
      label: 'triggerConf',
      insertText: 'triggerConf',
      detail: '插件请求参数',
      documentation: '当前函数请求参数对象。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'agentConf',
      insertText: 'agentConf',
      detail: '插件身份认证配置',
      documentation: '身份验证中配置的授权参数对象。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'async handler',
      insertText: ['async function handler() {', '  ${1:// TODO}', '  return ${2:{}}', '}'].join(
        '\n',
      ),
      detail: 'Node.js 异步函数模板',
      kind: monaco.languages.CompletionItemKind.Snippet,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
    {
      label: 'fetch',
      insertText: 'fetch(${1:url}, ${2:options})',
      detail: '发起 HTTP 请求',
      kind: monaco.languages.CompletionItemKind.Function,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
  ],
  python: [
    {
      label: 'triggerConf',
      insertText: 'triggerConf',
      detail: '插件请求参数',
      documentation: '当前函数请求参数字典。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'agentConf',
      insertText: 'agentConf',
      detail: '插件身份认证配置',
      documentation: '身份验证中配置的授权参数字典。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'requests.post',
      insertText: 'requests.post(${1:url}, json=${2:payload}, headers=${3:headers})',
      detail: 'Python HTTP POST 请求',
      kind: monaco.languages.CompletionItemKind.Function,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
    {
      label: 'try except',
      insertText: ['try:', '    ${1:pass}', 'except Exception as error:', '    raise error'].join(
        '\n',
      ),
      detail: 'Python 异常处理模板',
      kind: monaco.languages.CompletionItemKind.Snippet,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
  ],
  java: [
    {
      label: 'triggerConf',
      insertText: 'triggerConf',
      detail: '插件请求参数',
      documentation: '当前函数请求参数 Map。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'agentConf',
      insertText: 'agentConf',
      detail: '插件身份认证配置',
      documentation: '身份验证中配置的授权参数 Map。',
      kind: monaco.languages.CompletionItemKind.Variable,
    },
    {
      label: 'public method',
      insertText: ['public ${1:Object} ${2:handler}() {', '    ${3:return null;}', '}'].join('\n'),
      detail: 'Java 方法模板',
      kind: monaco.languages.CompletionItemKind.Snippet,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
    {
      label: 'try catch',
      insertText: [
        'try {',
        '    ${1:// TODO}',
        '} catch (Exception error) {',
        '    throw error;',
        '}',
      ].join('\n'),
      detail: 'Java 异常处理模板',
      kind: monaco.languages.CompletionItemKind.Snippet,
      insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    },
  ],
  json: [],
  plaintext: [],
};

const getCompletionRange = (model: monaco.editor.ITextModel, position: monaco.Position) => {
  const word = model.getWordUntilPosition(position);
  return {
    startLineNumber: position.lineNumber,
    endLineNumber: position.lineNumber,
    startColumn: word.startColumn,
    endColumn: word.endColumn,
  };
};

const getSafeFormatEdits = (model: monaco.editor.ITextModel) => {
  // Python/Java 暂无内置格式化器，仅清理行尾空白，避免破坏缩进语义。
  const nextValue = model
    .getLinesContent()
    .map((line) => line.replace(/[ \t]+$/u, ''))
    .join(model.getEOL());
  if (nextValue === model.getValue()) return [];
  return [
    {
      range: model.getFullModelRange(),
      text: nextValue,
    },
  ];
};

const registerCompletionProvider = () => {
  const completionState = getCompletionState();
  completionState.count += 1;
  if (completionState.disposables.length) return;

  monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
    noSemanticValidation: true,
    noSyntaxValidation: false,
  });
  monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
    target: monaco.languages.typescript.ScriptTarget.ES2020,
    allowNonTsExtensions: true,
  });

  completionState.disposables = [
    ...(['javascript', 'python', 'java'] as const).map((language) =>
      monaco.languages.registerCompletionItemProvider(language, {
        triggerCharacters: ['.', '"', "'", '/', '@'],
        provideCompletionItems(model, position) {
          const range = getCompletionRange(model, position);
          // Python/Java 只有基础高亮，这里补充插件运行时变量和常用代码片段提示。
          const suggestions = runtimeCompletionMap[language].map((item) => ({
            ...item,
            range,
          }));
          return { suggestions };
        },
      }),
    ),
    ...(['python', 'java', 'plaintext'] as const).map((language) =>
      monaco.languages.registerDocumentFormattingEditProvider(language, {
        provideDocumentFormattingEdits: getSafeFormatEdits,
      }),
    ),
  ];
};

const disposeCompletionProvider = () => {
  const completionState = getCompletionState();
  completionState.count = Math.max(completionState.count - 1, 0);
  if (completionState.count > 0) return;
  completionState.disposables.forEach((item) => item.dispose());
  completionState.disposables = [];
};

const disposeEditor = () => {
  if (editorModel.value) monaco.editor.setModelMarkers(editorModel.value, markerOwner, []);
  editorInstance.value?.dispose();
  editorModel.value?.dispose();
  editorInstance.value = undefined;
  editorModel.value = undefined;
};

const syncDiagnostics = () => {
  const model = editorModel.value;
  if (!model) return;
  const diagnostics = props.diagnostics || [];
  monaco.editor.setModelMarkers(
    model,
    markerOwner,
    diagnostics.map((item) => ({
      startLineNumber: item.line,
      startColumn: item.column,
      endLineNumber: item.endLine || item.line,
      endColumn: item.endColumn || item.column + 1,
      message: item.message,
      severity:
        item.severity === 'warning' ? monaco.MarkerSeverity.Warning : monaco.MarkerSeverity.Error,
    })),
  );
};

const focusFirstDiagnostic = () => {
  const diagnostic = props.diagnostics?.[0];
  const editor = editorInstance.value;
  if (!diagnostic || !editor) return;
  // 保存校验失败时，把光标放到第一处错误，让用户不用自己找红线位置。
  editor.revealPositionInCenter({
    lineNumber: diagnostic.line,
    column: diagnostic.column,
  });
  editor.setPosition({
    lineNumber: diagnostic.line,
    column: diagnostic.column,
  });
  editor.focus();
};

/**
 * 读取 Monaco 语言服务生成的语法错误，保存前由设计器统一提示和定位。
 * 自定义业务诊断使用独立 owner 写入，需要排除以避免重复返回。
 */
const getSyntaxDiagnostics = (): PluginCodeDiagnostic[] => {
  const model = editorModel.value;
  if (!model) return [];
  return monaco.editor
    .getModelMarkers({ resource: model.uri })
    .filter(
      (marker) => marker.owner !== markerOwner && marker.severity === monaco.MarkerSeverity.Error,
    )
    .map((marker) => ({
      line: marker.startLineNumber,
      column: marker.startColumn,
      endLine: marker.endLineNumber,
      endColumn: marker.endColumn,
      message: marker.message,
      severity: 'error',
    }));
};

const mountEditor = () => {
  if (!editorRef.value) return;
  disposeEditor();
  // 使用独立 model，避免多个插件函数切换时共享 Monaco 全局模型内容。
  const model = monaco.editor.createModel(
    props.modelValue,
    normalizeEditorLanguage(props.language),
  );
  const editor = monaco.editor.create(editorRef.value, {
    model,
    theme: 'vs-dark',
    automaticLayout: true,
    fontSize: props.fontSize,
    minimap: { enabled: false },
    readOnly: props.readOnly,
    scrollbar: {
      horizontal: props.hideHorizontalScrollbar ? 'hidden' : 'auto',
    },
    scrollBeyondLastLine: false,
    tabSize: 2,
    wordWrap: 'on',
    contextmenu: true,
  });
  editor.onDidChangeModelContent(() => {
    const value = editor.getValue();
    if (value !== props.modelValue) emits('update:modelValue', value);
  });
  editorModel.value = model;
  editorInstance.value = editor;
  syncDiagnostics();
  focusFirstDiagnostic();
};

watch(
  () => props.modelValue,
  (value) => {
    const editor = editorInstance.value;
    if (!editor || value === editor.getValue()) return;
    // 外部切换函数或加载详情时，仅同步内容，不重建编辑器实例。
    editor.setValue(value);
  },
);

watch(
  () => props.language,
  (language) => {
    if (!editorModel.value) return;
    // 运行环境切换后切换当前 model 语言，保证高亮跟随当前代码类型。
    monaco.editor.setModelLanguage(editorModel.value, normalizeEditorLanguage(language));
  },
);

watch(
  () => props.readOnly,
  (readOnly) => {
    editorInstance.value?.updateOptions({ readOnly });
  },
);

watch(
  () => props.fontSize,
  (fontSize) => {
    // 支持不同业务场景单独调整 Monaco 字号，不影响其它编辑器默认观感。
    editorInstance.value?.updateOptions({ fontSize });
  },
);

watch(
  () => props.hideHorizontalScrollbar,
  (hideHorizontalScrollbar) => {
    editorInstance.value?.updateOptions({
      scrollbar: {
        horizontal: hideHorizontalScrollbar ? 'hidden' : 'auto',
      },
    });
  },
);

watch(
  () => props.diagnostics,
  () => {
    syncDiagnostics();
  },
  { deep: true },
);

watch(
  () => props.diagnosticFocusKey,
  () => {
    focusFirstDiagnostic();
  },
);

onMounted(() => {
  registerCompletionProvider();
  mountEditor();
});

onBeforeUnmount(() => {
  disposeCompletionProvider();
  disposeEditor();
});

defineExpose({ getSyntaxDiagnostics });
</script>

<style lang="scss" scoped>
.plugin-code-editor {
  width: 100%;
  height: 100%;
  min-height: 0;
  background-color: #1e1e1e;
}
</style>
