<script setup lang="ts">
import {
  RiAlignLeft,
  RiAlignVertically,
  RiArrowGoBackFill,
  RiArrowGoForwardFill,
  RiDeleteBinLine,
  RiGitBranchFill,
  RiKeyboardBoxLine,
  RiPuzzle2Fill,
  RiSendPlaneFill,
  RiUser3Fill,
} from '@remixicon/vue';
import { ElPopover, ElTooltip } from 'element-plus';
import type { Component } from 'vue';
import type { WorkflowNodeType } from '../schema';

/** 横向工具栏：节点既可点击添加，也可拖入画布成为未连接节点。 */
defineOptions({ name: 'WorkflowPalette' });

defineProps<{
  canUndo?: boolean;
  canRedo?: boolean;
  canDelete?: boolean;
}>();

const emit = defineEmits<{
  addNode: [type: WorkflowNodeType];
  undo: [];
  redo: [];
  align: [];
  distribute: [];
  deleteSelected: [];
}>();

const PALETTE_ITEMS: Array<{ type: WorkflowNodeType; label: string; icon: Component }> = [
  { type: 'approval', label: '流程节点', icon: RiUser3Fill },
  { type: 'cc', label: '抄送节点', icon: RiSendPlaneFill },
  { type: 'subflow', label: '子流程', icon: RiGitBranchFill },
  { type: 'plugin', label: '插件节点', icon: RiPuzzle2Fill },
];

function startDrag(event: DragEvent, type: WorkflowNodeType) {
  event.dataTransfer?.setData('application/x-workflow-node-type', type);
  event.dataTransfer?.setData('text/plain', type);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
}
</script>

<template>
  <nav class="workflow-palette" aria-label="流程设计工具栏">
    <div class="workflow-palette__group">
      <ElTooltip content="撤销（⌘ + Z）" placement="bottom">
        <button
          type="button"
          class="workflow-palette__icon-button"
          aria-label="撤销"
          :disabled="!canUndo"
          @click="emit('undo')"
        >
          <RiArrowGoBackFill />
        </button>
      </ElTooltip>
      <ElTooltip content="重做（⌘ + Y）" placement="bottom">
        <button
          type="button"
          class="workflow-palette__icon-button"
          aria-label="重做"
          :disabled="!canRedo"
          @click="emit('redo')"
        >
          <RiArrowGoForwardFill />
        </button>
      </ElTooltip>
    </div>

    <span class="workflow-palette__divider" aria-hidden="true" />

    <div class="workflow-palette__group workflow-palette__nodes">
      <button
        v-for="item in PALETTE_ITEMS"
        :key="item.type"
        type="button"
        class="workflow-palette__node-button"
        draggable="true"
        :aria-label="`添加${item.label}`"
        @click="emit('addNode', item.type)"
        @dragstart="startDrag($event, item.type)"
      >
        <component :is="item.icon" />
        <span>{{ item.label }}</span>
      </button>
    </div>

    <span class="workflow-palette__divider" aria-hidden="true" />

    <div class="workflow-palette__group">
      <ElTooltip content="对齐" placement="bottom">
        <button type="button" class="workflow-palette__icon-button" aria-label="对齐" @click="emit('align')">
          <RiAlignLeft />
        </button>
      </ElTooltip>
      <ElTooltip content="排列" placement="bottom">
        <button type="button" class="workflow-palette__icon-button" aria-label="排列" @click="emit('distribute')">
          <RiAlignVertically />
        </button>
      </ElTooltip>
    </div>

    <span class="workflow-palette__divider" aria-hidden="true" />

    <div class="workflow-palette__group">
      <ElTooltip content="删除选中节点" placement="bottom">
        <button
          type="button"
          class="workflow-palette__icon-button"
          aria-label="删除选中节点"
          :disabled="!canDelete"
          @click="emit('deleteSelected')"
        >
          <RiDeleteBinLine />
        </button>
      </ElTooltip>
      <ElPopover placement="bottom-end" :width="250" trigger="click">
        <template #reference>
          <button type="button" class="workflow-palette__icon-button" aria-label="快捷键说明">
            <RiKeyboardBoxLine />
          </button>
        </template>
        <div class="workflow-palette__shortcuts">
          <span>撤销</span><kbd>⌘ / Ctrl + Z</kbd>
          <span>重做</span><kbd>⌘ + Shift + Z / Ctrl + Y</kbd>
          <span>删除选中</span><kbd>Delete / Backspace</kbd>
        </div>
      </ElPopover>
    </div>
  </nav>
</template>

<style scoped lang="scss">
.workflow-palette {
  display: flex;
  min-height: 56px;
  padding: 0 18px;
  align-items: center;
  gap: 10px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__group {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  &__nodes {
    gap: 2px;
  }

  &__divider {
    width: 1px;
    height: 24px;
    margin: 0 4px;
    background: var(--el-border-color-lighter);
  }

  &__icon-button,
  &__node-button {
    display: inline-flex;
    height: 38px;
    padding: 0 10px;
    align-items: center;
    justify-content: center;
    gap: 7px;
    appearance: none;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-regular);
    cursor: pointer;
    font: inherit;

    svg {
      width: 19px;
      height: 19px;
    }

    &:hover:not(:disabled) {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);
    }

    &:disabled {
      color: var(--el-text-color-disabled);
      cursor: not-allowed;
    }
  }

  &__icon-button {
    width: 38px;
    padding: 0;
  }

  &__node-button {
    color: var(--el-text-color-primary);
    font-weight: 600;

    &:nth-child(1) svg { color: var(--el-color-primary); }
    &:nth-child(2) svg { color: var(--el-color-success); }
    &:nth-child(3) svg { color: var(--el-color-warning); }
    &:nth-child(4) svg { color: var(--el-color-purple, #7c5ce7); }
  }

  &__shortcuts {
    display: grid;
    align-items: center;
    gap: 10px 12px;
    grid-template-columns: 1fr auto;

    kbd {
      padding: 3px 6px;
      color: var(--el-text-color-regular);
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color);
      border-radius: 4px;
      font-family: inherit;
      font-size: 12px;
    }
  }
}

@media (max-width: 900px) {
  .workflow-palette__node-button span {
    display: none;
  }
}
</style>
