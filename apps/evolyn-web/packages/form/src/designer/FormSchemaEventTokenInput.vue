<script setup lang="ts">
import { RiAddLine } from '@remixicon/vue';
import { nextTick, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import { ElPopover } from 'element-plus';
import type { FormEventFieldOption } from './frontend-events';
import FormSchemaEventFieldPicker from './FormSchemaEventFieldPicker.vue';

const model = defineModel<string>({ required: true });
const props = withDefaults(
  defineProps<{
    fields: readonly FormEventFieldOption[];
    rows?: number;
    placeholder?: string;
    insertLabel?: string;
  }>(),
  { rows: 1, placeholder: '', insertLabel: '插入字段' },
);

const pickerOpen = shallowRef(false);
const editor = useTemplateRef<HTMLElement>('editor');
const selectionOffset = shallowRef<number | null>(null);
const renderedSignature = shallowRef('');

interface TokenSegment {
  kind: 'text' | 'token';
  value: string;
  key?: string;
}

function splitTemplate(template: string, fields: readonly FormEventFieldOption[]): TokenSegment[] {
  const segments: TokenSegment[] = [];
  const labels = new Map(fields.map((field) => [field.key, field.label || field.key]));
  let cursor = 0;
  for (const match of template.matchAll(/\$\{([A-Za-z0-9_]+)\}/g)) {
    const index = match.index ?? 0;
    if (index > cursor) segments.push({ kind: 'text', value: template.slice(cursor, index) });
    const key = match[1]!;
    segments.push({ kind: 'token', key, value: labels.get(key) ?? key });
    cursor = index + match[0].length;
  }
  if (cursor < template.length) segments.push({ kind: 'text', value: template.slice(cursor) });
  return segments;
}

/**
 * contenteditable 内部会被浏览器在输入时直接改写。不能让 Vue 同时管理这些子节点，
 * 否则打开字段选择浮层触发重渲染时，旧 VNode 与实际 DOM 脱节并导致 patchElement 报错。
 */
function renderTemplate(): void {
  const root = editor.value;
  if (!root) return;
  const labels = new Map(props.fields.map((field) => [field.key, field.label || field.key]));
  const fragment = document.createDocumentFragment();
  for (const segment of splitTemplate(model.value, props.fields)) {
    if (segment.kind === 'text') {
      const text = document.createElement('span');
      text.className = 'event-token-input__text';
      text.textContent = segment.value;
      fragment.append(text);
      continue;
    }
    const token = document.createElement('span');
    token.className = 'event-token-input__token';
    token.dataset.token = segment.key;
    token.contentEditable = 'false';
    token.textContent = labels.get(segment.key ?? '') ?? segment.value;
    fragment.append(token);
  }
  root.replaceChildren(fragment);
  renderedSignature.value = templateSignature();
}

function templateSignature(template = model.value): string {
  return `${template}\u0000${props.fields.map((field) => `${field.key}\u0000${field.label}`).join('\u0001')}`;
}

function serialize(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) return node.textContent ?? '';
  if (!(node instanceof HTMLElement)) return Array.from(node.childNodes).map(serialize).join('');
  const token = node.dataset.token;
  if (token) return `\${${token}}`;
  if (node.tagName === 'BR') return '\n';
  const value = Array.from(node.childNodes).map(serialize).join('');
  return ['DIV', 'P'].includes(node.tagName) ? `${value}\n` : value;
}

function readSelectionOffset(): number | null {
  const root = editor.value;
  const range = window.getSelection()?.rangeCount ? window.getSelection()!.getRangeAt(0) : null;
  if (!root || !range || !root.contains(range.startContainer)) return null;
  const prefix = range.cloneRange();
  prefix.selectNodeContents(root);
  prefix.setEnd(range.startContainer, range.startOffset);
  return serialize(prefix.cloneContents()).length;
}

function restoreSelection(offset: number): void {
  const root = editor.value;
  if (!root) return;
  let remaining = offset;
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT);
  let current: Node | null = walker.nextNode();
  while (current) {
    if (current instanceof HTMLElement && current.dataset.token) {
      const length = `\${${current.dataset.token}}`.length;
      if (remaining <= length) {
        const parent = current.parentNode;
        if (!parent) return;
        const index = Array.prototype.indexOf.call(parent.childNodes, current);
        setCaret(parent, remaining === 0 ? index : index + 1);
        return;
      }
      remaining -= length;
      current = walker.nextSibling();
      continue;
    }
    if (current.nodeType === Node.TEXT_NODE) {
      const length = current.textContent?.length ?? 0;
      if (remaining <= length) {
        setCaret(current, remaining);
        return;
      }
      remaining -= length;
    }
    current = walker.nextNode();
  }
  setCaret(root, root.childNodes.length);
}

function setCaret(node: Node, offset: number): void {
  const range = document.createRange();
  range.setStart(node, offset);
  range.collapse(true);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

function captureSelection(): void {
  selectionOffset.value = readSelectionOffset();
}

function updateTemplate(): void {
  const nextValue = editor.value ? serialize(editor.value).replace(/\n$/, '') : '';
  // model 写回父组件后，props 通常会在下一个渲染周期才回传新值。先用本次输入
  // 的文本更新签名，避免回传时被误判为外部修改而 replaceChildren，导致光标跳到开头。
  renderedSignature.value = templateSignature(nextValue);
  model.value = nextValue;
}

function insertField(field: FormEventFieldOption): void {
  const token = `\${${field.key}}`;
  const start = selectionOffset.value ?? readSelectionOffset() ?? model.value.length;
  const end = start;
  model.value = `${model.value.slice(0, start)}${token}${model.value.slice(end)}`;
  pickerOpen.value = false;
  nextTick(() => {
    renderTemplate();
    editor.value?.focus();
    restoreSelection(start + token.length);
  });
}

onMounted(renderTemplate);

watch(
  templateSignature,
  (signature) => {
    if (signature !== renderedSignature.value) renderTemplate();
  },
  { flush: 'post' },
);
</script>

<template>
  <div :class="['event-token-input', { 'event-token-input--single-line': props.rows === 1 }]">
    <div
      ref="editor"
      :class="[
        'event-token-input__control',
        { 'event-token-input__control--textarea': props.rows > 1 },
      ]"
      :data-placeholder="props.placeholder"
      contenteditable="true"
      role="textbox"
      :aria-multiline="props.rows > 1"
      @input="updateTemplate"
      @keyup="captureSelection"
      @mouseup="captureSelection"
      @focus="captureSelection"
    ></div>
    <el-popover
      v-model:visible="pickerOpen"
      placement="bottom-end"
      :width="320"
      trigger="click"
    >
      <template #reference>
        <button
          class="event-token-input__insert"
          type="button"
          @mousedown.prevent="captureSelection"
        >
          <RiAddLine aria-hidden="true" />{{ props.insertLabel }}
        </button>
      </template>
      <FormSchemaEventFieldPicker :fields="props.fields" @select="insertField" />
    </el-popover>
  </div>
</template>

<style scoped lang="scss">
.event-token-input { position: relative; width: 100%; }
.event-token-input__control { box-sizing: border-box; width: 100%; min-height: 40px; padding: 8px 112px 8px 11px; overflow: auto; color: var(--el-text-color-primary); font: inherit; line-height: 22px; white-space: pre-wrap; overflow-wrap: anywhere; background: var(--el-bg-color); border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); outline: 0; }
.event-token-input__control:empty::before { color: var(--el-text-color-placeholder); pointer-events: none; content: attr(data-placeholder); }
.event-token-input__control:focus { border-color: var(--el-color-primary); box-shadow: 0 0 0 1px var(--el-color-primary-light-7); }
.event-token-input__control--textarea { min-height: 92px; }
/* 令牌节点由 renderTemplate 手动创建，不会带上 Vue scoped attribute，需穿透作用域。 */
:deep(.event-token-input__token) { display: inline-flex; max-width: 100%; align-items: center; padding: 0 5px; margin: 0 2px; overflow: hidden; color: var(--el-text-color-regular); line-height: 24px; vertical-align: baseline; text-overflow: ellipsis; white-space: nowrap; background: var(--el-fill-color); border-radius: 3px; user-select: all; }
:deep(.event-token-input__text) { white-space: pre-wrap; }
.event-token-input__insert { position: absolute; top: 7px; right: 8px; display: inline-flex; gap: 3px; align-items: center; padding: 4px 5px; color: var(--el-color-primary); font: inherit; cursor: pointer; background: var(--el-bg-color); border: 0; border-radius: 4px; }
/* 单行控件与 Element Plus large 输入框同为 40px，高度变化时仍保持按钮居中。 */
.event-token-input--single-line .event-token-input__insert { top: 50%; transform: translateY(-50%); }
.event-token-input__insert:hover { background: var(--el-color-primary-light-9); }
</style>
