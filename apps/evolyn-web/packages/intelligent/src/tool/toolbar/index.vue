<script setup lang="ts">
import {
  RiAddLine,
  RiArrowGoBackLine,
  RiArrowGoForwardLine,
  RiDeleteBinLine,
  RiLayoutGridLine,
  RiQuestionLine,
  RiTargetLine,
  RiZoomInLine,
  RiZoomOutLine,
} from '@remixicon/vue';
import { ElTooltip } from 'element-plus';

defineOptions({ name: 'IntelligentToolbar' });

defineProps<{
  canUndo: boolean;
  canRedo: boolean;
  canDelete: boolean;
  zoomPercent: number;
  triggerLabel: string;
}>();

const emit = defineEmits<{
  addNode: [];
  beautify: [];
  deleteNode: [];
  fitView: [];
  help: [];
  redo: [];
  undo: [];
  zoomIn: [];
  zoomOut: [];
}>();
</script>

<template>
  <nav class="intelligent-toolbar" aria-label="智能助手设计工具栏">
    <div class="intelligent-toolbar__group">
      <ElTooltip content="撤销（⌘ / Ctrl + Z）" placement="bottom">
        <button type="button" aria-label="撤销" :disabled="!canUndo" @click="emit('undo')">
          <RiArrowGoBackLine />
        </button>
      </ElTooltip>
      <ElTooltip content="重做（⌘ + Shift + Z / Ctrl + Y）" placement="bottom">
        <button type="button" aria-label="重做" :disabled="!canRedo" @click="emit('redo')">
          <RiArrowGoForwardLine />
        </button>
      </ElTooltip>
    </div>
    <span class="intelligent-toolbar__divider" />
    <div class="intelligent-toolbar__group">
      <button type="button" aria-label="缩小" @click="emit('zoomOut')"><RiZoomOutLine /></button>
      <span class="intelligent-toolbar__zoom">{{ zoomPercent }}%</span>
      <button type="button" aria-label="放大" @click="emit('zoomIn')"><RiZoomInLine /></button>
      <button type="button" aria-label="居中画布" @click="emit('fitView')"><RiTargetLine /></button>
      <button type="button" aria-label="美化布局" @click="emit('beautify')"><RiLayoutGridLine /></button>
    </div>
    <span class="intelligent-toolbar__divider" />
    <div class="intelligent-toolbar__group">
      <button type="button" class="intelligent-toolbar__text" @click="emit('addNode')">
        <RiAddLine />添加执行节点
      </button>
      <button
        type="button"
        aria-label="删除选中节点"
        :disabled="!canDelete"
        @click="emit('deleteNode')"
      >
        <RiDeleteBinLine />
      </button>
    </div>
    <span class="intelligent-toolbar__divider" />
    <button type="button" class="intelligent-toolbar__text" @click="emit('help')">
      <RiQuestionLine />帮助文档
    </button>
    <span class="intelligent-toolbar__trigger">{{ triggerLabel }}</span>
  </nav>
</template>

<style scoped lang="scss">
.intelligent-toolbar {
  display: flex;
  min-width: 0;
  padding: 0 28px;
  align-items: center;
  gap: 14px;
  background: #fff;
  border-bottom: 1px solid #e5e8ee;
  box-shadow: 0 3px 10px rgb(31 43 61 / 5%);

  &__group { display: flex; align-items: center; gap: 4px; }

  button {
    display: inline-flex;
    height: 34px;
    min-width: 34px;
    padding: 0 8px;
    align-items: center;
    justify-content: center;
    gap: 6px;
    color: #485467;
    background: transparent;
    border: 0;
    border-radius: 6px;
    cursor: pointer;
    font: inherit;

    &:hover:not(:disabled) { color: #00a99d; background: #edf9f7; }
    &:disabled { color: #b4bcc8; cursor: not-allowed; }
    svg { width: 18px; height: 18px; }
  }

  &__text { width: auto; }
  &__divider { width: 1px; height: 24px; background: #e1e5eb; }
  &__zoom { min-width: 48px; color: #3f4a5c; font-size: 14px; text-align: center; }
  &__trigger {
    overflow: hidden;
    margin-left: auto;
    color: #7a8596;
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

@media (max-width: 760px) {
  .intelligent-toolbar {
    padding: 8px 12px;
    flex-wrap: wrap;

    &__trigger { width: 100%; margin-left: 0; }
  }
}
</style>
