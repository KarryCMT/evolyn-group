<script setup lang="ts">
import { RiArrowLeftSLine } from '@remixicon/vue';
import { ElButton } from 'element-plus';
import { computed, useTemplateRef, watch } from 'vue';
import type { IntelligentDesignerResources, IntelligentDocument } from '../schema';
import LogicPanel from '../components/LogicPanel.vue';
import IntelligentToolbar from '../tool/toolbar/index.vue';
import { useIntelligentDesigner } from './useIntelligentDesigner';

defineOptions({ name: 'IntelligentDesigner' });

const props = defineProps<{
  document: IntelligentDocument;
  resources?: IntelligentDesignerResources;
}>();

const emit = defineEmits<{
  updateDocument: [document: IntelligentDocument];
  back: [];
  save: [document: IntelligentDocument];
  enable: [document: IntelligentDocument];
  help: [];
}>();

interface LogicPanelExpose {
  fitView: () => void;
  openInsertMenu: () => void;
  zoomIn: () => void;
  zoomOut: () => void;
  zoomPercent: number;
}

const logicPanelRef = useTemplateRef<LogicPanelExpose>('logicPanelRef');
const {
  addNode,
  beautify,
  canRedo,
  canUndo,
  deleteSelected,
  handleKeyboard,
  moveNode,
  redo,
  removeNode,
  resetHistory,
  selectedAction,
  selectedNodeId,
  undo,
  updateNode,
  updateTrigger,
} = useIntelligentDesigner({
  getDocument: () => props.document,
  updateDocument: (document) => emit('updateDocument', document),
});

const triggerLabel = computed(() => {
  if (props.document.trigger.type === 'schedule') return '定时触发';
  if (props.document.trigger.type === 'http') return 'HTTP 触发';
  return `表单触发 · ${props.document.trigger.formName || '当前表单'}`;
});

// 同一路由切换助手时清空旧助手的撤销记录与选择状态。
watch(
  () => props.document.id,
  () => resetHistory(props.document),
);
</script>

<template>
  <section class="intelligent-designer" tabindex="-1" @keydown="handleKeyboard">
    <header class="intelligent-designer__header">
      <div class="intelligent-designer__identity">
        <button type="button" aria-label="返回智能助手列表" @click="emit('back')">
          <RiArrowLeftSLine />
        </button>
        <h1>{{ document.name }}</h1>
        <span>有未发布变更</span>
      </div>
      <div class="intelligent-designer__actions">
        <button type="button" class="intelligent-designer__link">联动触发</button>
        <ElButton @click="emit('save', document)">仅保存</ElButton>
        <ElButton type="primary" @click="emit('enable', document)">保存并启用</ElButton>
      </div>
    </header>

    <IntelligentToolbar
      :can-undo="canUndo"
      :can-redo="canRedo"
      :can-delete="Boolean(selectedAction)"
      :zoom-percent="logicPanelRef?.zoomPercent ?? 100"
      :trigger-label="triggerLabel"
      @undo="undo"
      @redo="redo"
      @zoom-in="logicPanelRef?.zoomIn()"
      @zoom-out="logicPanelRef?.zoomOut()"
      @fit-view="logicPanelRef?.fitView()"
      @beautify="beautify"
      @add-node="logicPanelRef?.openInsertMenu()"
      @delete-node="deleteSelected"
      @help="emit('help')"
    />

    <LogicPanel
      ref="logicPanelRef"
      class="intelligent-designer__canvas"
      :document="document"
      :selected-node-id="selectedNodeId"
      :resources="resources"
      @select-node="selectedNodeId = $event"
      @move-node="moveNode"
      @add-node="addNode"
      @update-node="updateNode"
      @update-trigger="updateTrigger"
      @remove-node="removeNode"
    />
  </section>
</template>

<style scoped lang="scss">
.intelligent-designer {
  display: grid;
  width: 100%;
  min-width: 0;
  min-height: 100vh;
  color: #172033;
  background: #f7f8fa;
  outline: none;
  grid-template-rows: 64px 64px minmax(0, 1fr);

  &__header {
    display: flex;
    padding: 0 28px;
    align-items: center;
    justify-content: space-between;
    background: #fff;
    border-bottom: 1px solid #e5e8ee;
  }

  &__identity,
  &__actions { display: flex; align-items: center; }

  &__identity {
    min-width: 0;
    gap: 10px;

    > button {
      display: inline-flex;
      width: 34px;
      height: 34px;
      padding: 0;
      align-items: center;
      justify-content: center;
      color: #354154;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;

      &:hover { color: #00a99d; background: #edf9f7; }
      svg { width: 26px; height: 26px; }
    }

    h1 {
      overflow: hidden;
      margin: 0;
      font-size: 18px;
      font-weight: 600;
      line-height: 28px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      padding: 3px 8px;
      flex: 0 0 auto;
      color: #9b721b;
      background: #fff4d9;
      border-radius: 4px;
      font-size: 12px;
    }
  }

  &__actions {
    gap: 10px;

    :deep(.el-button--primary) {
      --el-button-bg-color: #00afa2;
      --el-button-border-color: #00afa2;
      --el-button-hover-bg-color: #08baad;
      --el-button-hover-border-color: #08baad;
    }
  }

  &__link {
    margin-right: 4px;
    color: #00a99d;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
  }

  &__canvas { min-height: 0; }
}

@media (max-width: 760px) {
  .intelligent-designer {
    grid-template-rows: auto auto minmax(520px, 1fr);

    &__header {
      padding: 12px 16px;
      align-items: flex-start;
      flex-direction: column;
      gap: 12px;
    }

    &__actions { align-self: flex-end; }
  }
}
</style>
