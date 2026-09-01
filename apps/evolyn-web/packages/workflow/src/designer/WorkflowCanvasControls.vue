<script setup lang="ts">
import { RiTargetFill, RiZoomInFill, RiZoomOutFill } from '@remixicon/vue';

defineOptions({ name: 'WorkflowCanvasControls' });

defineProps<{
  zoomPercent: number;
}>();

const emit = defineEmits<{
  fitView: [];
  zoomIn: [];
  zoomOut: [];
}>();
</script>

<template>
  <div class="workflow-canvas-controls" aria-label="流程画布控制">
    <div class="workflow-canvas-controls__zoom">
      <button type="button" aria-label="缩小" @click="emit('zoomOut')">
        <RiZoomOutFill />
      </button>
      <span class="workflow-canvas-controls__zoom-value">{{ zoomPercent }}%</span>
      <button type="button" aria-label="放大" @click="emit('zoomIn')">
        <RiZoomInFill />
      </button>
      <span class="workflow-canvas-controls__divider" aria-hidden="true" />
      <button type="button" aria-label="居中显示流程" @click="emit('fitView')">
        <RiTargetFill />
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.workflow-canvas-controls {
  position: absolute;
  z-index: 3;
  inset: 16px;
  pointer-events: none;

  &__zoom {
    position: absolute;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    padding: 4px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow-lighter);
    pointer-events: auto;
  }

  &__zoom button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border: 0;
    border-radius: calc(var(--el-border-radius-base) - 2px);
    cursor: pointer;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
  }

  &__divider {
    width: 1px;
    height: 24px;
    margin: 0 4px;
    background: var(--el-border-color-lighter);
  }

  &__zoom-value {
    min-width: 52px;
    color: var(--el-text-color-regular);
    font-size: 14px;
    font-variant-numeric: tabular-nums;
    text-align: center;
  }
}

@media (max-width: 700px) {
  .workflow-canvas-controls {
    inset: 8px;
  }
}
</style>
