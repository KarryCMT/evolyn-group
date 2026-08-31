<script setup lang="ts">
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue';
import type { Editor } from '@tiptap/core';
import Color from '@tiptap/extension-color';
import Image from '@tiptap/extension-image';
import Link from '@tiptap/extension-link';
import TextAlign from '@tiptap/extension-text-align';
import { TextStyle } from '@tiptap/extension-text-style';
import Underline from '@tiptap/extension-underline';
import StarterKit from '@tiptap/starter-kit';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import RichTextToolbar from './RichTextToolbar.vue';
import type {
  EvolynRichTextEditorEmits,
  EvolynRichTextEditorProps,
  EvolynRichTextFormatCommand,
  EvolynRichTextToolbarState,
} from './EvolynRichTextEditor.types';

defineOptions({ name: 'EvolynRichTextEditor' });

const props = withDefaults(defineProps<EvolynRichTextEditorProps>(), {
  editable: true,
  minHeight: 156,
  imageAccept: () => ['image/jpeg', 'image/png', 'image/webp'],
  maxImageSize: 10 * 1024 * 1024,
  ariaLabel: '富文本编辑器',
});
const emit = defineEmits<EvolynRichTextEditorEmits>();
const modelValue = defineModel<string>({ default: '' });

const emptyToolbarState: EvolynRichTextToolbarState = {
  bold: false,
  italic: false,
  underline: false,
  alignLeft: false,
  link: false,
};
const toolbarState = shallowRef<EvolynRichTextToolbarState>({ ...emptyToolbarState });
const showLinkEditor = shallowRef(false);
const linkValue = shallowRef('');
const linkError = shallowRef('');
const imageUploading = shallowRef(false);
const imageInput = useTemplateRef<HTMLInputElement>('imageInput');
const linkInput = useTemplateRef<HTMLInputElement>('linkInput');
const editorContainerStyle = computed(() => ({
  '--evolyn-rich-text-editor-min-height': toCssLength(props.minHeight),
}));
const toolbarDisabled = computed(() => !props.editable || imageUploading.value);

const editor = useEditor({
  content: modelValue.value,
  editable: props.editable,
  extensions: [
    StarterKit.configure({ link: false }),
    Underline,
    TextStyle,
    Color,
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    Link.configure({
      autolink: true,
      openOnClick: false,
      HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
    }),
    Image.configure({
      allowBase64: false,
      HTMLAttributes: { class: 'evolyn-rich-text-editor__image' },
    }),
  ],
  editorProps: {
    attributes: {
      'aria-label': props.ariaLabel,
      'data-placeholder': '请输入内容',
    },
  },
  onTransaction: updateToolbarState,
  onUpdate: ({ editor: updatedEditor }) => {
    const html = updatedEditor.isEmpty ? '' : updatedEditor.getHTML();
    if (modelValue.value !== html) {
      modelValue.value = html;
      emit('change', html);
    }
  },
});

watch(modelValue, (html) => {
  const activeEditor = editor.value;
  if (!activeEditor) return;
  const currentHtml = activeEditor.isEmpty ? '' : activeEditor.getHTML();
  if (html !== currentHtml) {
    activeEditor.commands.setContent(html || '', { emitUpdate: false });
    updateToolbarState();
  }
});

watch(
  () => props.editable,
  (editable) => editor.value?.setEditable(editable),
);

function toCssLength(value: number | string) {
  return typeof value === 'number' ? `${value}px` : value;
}

function updateToolbarState() {
  const activeEditor = editor.value;
  if (!activeEditor) {
    toolbarState.value = { ...emptyToolbarState };
    return;
  }
  toolbarState.value = {
    bold: activeEditor.isActive('bold'),
    italic: activeEditor.isActive('italic'),
    underline: activeEditor.isActive('underline'),
    alignLeft: activeEditor.isActive({ textAlign: 'left' }),
    link: activeEditor.isActive('link'),
    color: activeEditor.getAttributes('textStyle').color as string | undefined,
  };
}

function executeCommand(command: EvolynRichTextFormatCommand) {
  const activeEditor = editor.value;
  if (!activeEditor || toolbarDisabled.value) return;

  switch (command) {
    case 'bold':
      activeEditor.chain().focus().toggleBold().run();
      break;
    case 'italic':
      activeEditor.chain().focus().toggleItalic().run();
      break;
    case 'underline':
      activeEditor.chain().focus().toggleUnderline().run();
      break;
    case 'alignLeft':
      activeEditor.chain().focus().setTextAlign('left').run();
      break;
    case 'clear':
      activeEditor.chain().focus().unsetAllMarks().clearNodes().run();
      break;
    case 'link':
      openLinkEditor(activeEditor);
      break;
    case 'unlink':
      activeEditor.chain().focus().unsetLink().run();
      break;
  }
}

function setColor(color: string) {
  if (!editor.value || toolbarDisabled.value) return;
  editor.value.chain().focus().setColor(color).run();
}

function openLinkEditor(activeEditor: Editor) {
  linkValue.value = (activeEditor.getAttributes('link').href as string | undefined) ?? '';
  linkError.value = '';
  showLinkEditor.value = true;
  void nextTick(() => linkInput.value?.focus());
}

function closeLinkEditor() {
  showLinkEditor.value = false;
  linkError.value = '';
}

function saveLink() {
  const activeEditor = editor.value;
  if (!activeEditor) return;
  const href = normalizeLink(linkValue.value);
  if (href === null) {
    linkError.value = '仅支持 http、https、mailto 和 tel 链接';
    return;
  }
  if (!href) activeEditor.chain().focus().unsetLink().run();
  else activeEditor.chain().focus().extendMarkRange('link').setLink({ href }).run();
  closeLinkEditor();
}

function normalizeLink(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return '';
  const candidate = /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(trimmed) ? trimmed : `https://${trimmed}`;
  try {
    const url = new URL(candidate);
    return ['http:', 'https:', 'mailto:', 'tel:'].includes(url.protocol) ? url.href : null;
  } catch {
    return null;
  }
}

function requestImage() {
  if (!props.uploadImage || toolbarDisabled.value) return;
  imageInput.value?.click();
}

async function handleImageChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file || !props.uploadImage) return;
  if (!isAcceptedImage(file)) return reportImageError('请选择允许格式的图片');
  if (file.size > props.maxImageSize) {
    return reportImageError(`图片大小不能超过 ${formatFileSize(props.maxImageSize)}`);
  }

  imageUploading.value = true;
  try {
    const url = await props.uploadImage(file);
    if (!isSafeImageUrl(url)) throw new Error('图片地址仅支持 HTTPS、HTTP 或站内相对路径');
    editor.value?.chain().focus().setImage({ src: url }).run();
    emit('imageUpload', file, url);
  } catch (error) {
    reportImageError(error instanceof Error ? error.message : '图片上传失败，请稍后重试');
  } finally {
    imageUploading.value = false;
  }
}

function isAcceptedImage(file: File) {
  return props.imageAccept.some(
    (type) => type === 'image/*' || type.toLowerCase() === file.type.toLowerCase(),
  );
}

function isSafeImageUrl(value: string) {
  // 仅允许站内绝对路径，避免把 //host/path 当成相对路径绕过协议校验。
  if (value.startsWith('/') && !value.startsWith('//')) return true;
  try {
    return ['http:', 'https:'].includes(new URL(value).protocol);
  } catch {
    return false;
  }
}

function formatFileSize(size: number) {
  return `${Math.ceil(size / 1024 / 1024)} MB`;
}

function reportImageError(message: string) {
  emit('imageUploadError', message);
}

function focus() {
  editor.value?.commands.focus();
}

function getHTML() {
  return editor.value?.isEmpty ? '' : (editor.value?.getHTML() ?? '');
}

defineExpose({ focus, getHTML });
</script>

<template>
  <section
    class="evolyn-rich-text-editor"
    :class="{ 'is-readonly': !props.editable, 'is-uploading': imageUploading }"
    :style="editorContainerStyle"
  >
    <RichTextToolbar
      v-if="props.editable"
      :disabled="toolbarDisabled"
      :image-enabled="Boolean(props.uploadImage)"
      :state="toolbarState"
      @command="executeCommand"
      @color-change="setColor"
      @image-select="requestImage"
    />

    <form
      v-if="showLinkEditor"
      class="evolyn-rich-text-editor__link-editor"
      @submit.prevent="saveLink"
    >
      <label class="evolyn-rich-text-editor__link-label" for="evolyn-rich-text-link"
        >链接地址</label
      >
      <input
        id="evolyn-rich-text-link"
        ref="linkInput"
        v-model="linkValue"
        class="evolyn-rich-text-editor__link-input"
        type="url"
        placeholder="https://example.com"
        @keydown.esc.prevent="closeLinkEditor"
      />
      <span v-if="linkError" class="evolyn-rich-text-editor__link-error">{{ linkError }}</span>
      <button class="evolyn-rich-text-editor__link-button" type="submit">确认</button>
      <button
        class="evolyn-rich-text-editor__link-button is-secondary"
        type="button"
        @click="closeLinkEditor"
      >
        取消
      </button>
    </form>

    <EditorContent :editor="editor" class="evolyn-rich-text-editor__content" />
    <input
      ref="imageInput"
      class="evolyn-rich-text-editor__image-input"
      type="file"
      :accept="props.imageAccept.join(',')"
      :disabled="toolbarDisabled"
      tabindex="-1"
      aria-hidden="true"
      @change="handleImageChange"
    />
  </section>
</template>

<style lang="scss">
@use './EvolynRichTextEditor.scss' as *;
</style>
