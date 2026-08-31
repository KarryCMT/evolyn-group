<script setup lang="ts">
import {
  RiAlignLeft,
  RiBold,
  RiFontColor,
  RiFormatClear,
  RiImageAddLine,
  RiItalic,
  RiLink,
  RiLinkUnlink,
  RiUnderline,
} from '@remixicon/vue';
import type {
  EvolynRichTextFormatCommand,
  EvolynRichTextToolbarState,
} from './EvolynRichTextEditor.types';

interface RichTextToolbarProps {
  disabled?: boolean;
  imageEnabled?: boolean;
  state: EvolynRichTextToolbarState;
}

const props = withDefaults(defineProps<RichTextToolbarProps>(), {
  disabled: false,
  imageEnabled: false,
});
const emit = defineEmits<{
  command: [command: EvolynRichTextFormatCommand];
  colorChange: [color: string];
  imageSelect: [];
}>();

function execute(command: EvolynRichTextFormatCommand) {
  if (!props.disabled) emit('command', command);
}

function changeColor(event: Event) {
  emit('colorChange', (event.target as HTMLInputElement).value);
}
</script>

<template>
  <div class="evolyn-rich-text-toolbar" role="toolbar" aria-label="富文本工具栏">
    <button
      class="evolyn-rich-text-toolbar__button"
      :class="{ 'is-active': props.state.bold }"
      type="button"
      title="加粗"
      aria-label="加粗"
      :aria-pressed="props.state.bold"
      :disabled="props.disabled"
      @click="execute('bold')"
    >
      <RiBold aria-hidden="true" />
    </button>
    <button
      class="evolyn-rich-text-toolbar__button"
      :class="{ 'is-active': props.state.italic }"
      type="button"
      title="斜体"
      aria-label="斜体"
      :aria-pressed="props.state.italic"
      :disabled="props.disabled"
      @click="execute('italic')"
    >
      <RiItalic aria-hidden="true" />
    </button>
    <button
      class="evolyn-rich-text-toolbar__button"
      :class="{ 'is-active': props.state.underline }"
      type="button"
      title="下划线"
      aria-label="下划线"
      :aria-pressed="props.state.underline"
      :disabled="props.disabled"
      @click="execute('underline')"
    >
      <RiUnderline aria-hidden="true" />
    </button>
    <button
      class="evolyn-rich-text-toolbar__button"
      :class="{ 'is-active': props.state.alignLeft }"
      type="button"
      title="左对齐"
      aria-label="左对齐"
      :aria-pressed="props.state.alignLeft"
      :disabled="props.disabled"
      @click="execute('alignLeft')"
    >
      <RiAlignLeft aria-hidden="true" />
    </button>
    <label class="evolyn-rich-text-toolbar__color" title="文字颜色">
      <RiFontColor aria-hidden="true" />
      <input
        type="color"
        :value="props.state.color ?? '#4b5563'"
        aria-label="文字颜色"
        :disabled="props.disabled"
        @input="changeColor"
      />
    </label>
    <button
      class="evolyn-rich-text-toolbar__button"
      type="button"
      title="清除格式"
      aria-label="清除格式"
      :disabled="props.disabled"
      @click="execute('clear')"
    >
      <RiFormatClear aria-hidden="true" />
    </button>
    <span class="evolyn-rich-text-toolbar__divider" aria-hidden="true" />
    <button
      class="evolyn-rich-text-toolbar__button"
      :class="{ 'is-active': props.state.link }"
      type="button"
      title="编辑链接"
      aria-label="编辑链接"
      :aria-pressed="props.state.link"
      :disabled="props.disabled"
      @click="execute('link')"
    >
      <RiLink aria-hidden="true" />
    </button>
    <button
      class="evolyn-rich-text-toolbar__button"
      type="button"
      title="取消链接"
      aria-label="取消链接"
      :disabled="props.disabled || !props.state.link"
      @click="execute('unlink')"
    >
      <RiLinkUnlink aria-hidden="true" />
    </button>
    <button
      class="evolyn-rich-text-toolbar__button"
      type="button"
      title="插入图片"
      aria-label="插入图片"
      :disabled="props.disabled || !props.imageEnabled"
      @click="emit('imageSelect')"
    >
      <RiImageAddLine aria-hidden="true" />
    </button>
  </div>
</template>
