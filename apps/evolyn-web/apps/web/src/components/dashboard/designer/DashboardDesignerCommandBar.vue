<script setup lang="ts">
import type { BusinessDashboardWidget } from '@evolyn.do/dashboard';
import {
  RiComputerFill,
  RiDatabase2Line,
  RiDeleteBin6Line,
  RiDragMove2Fill,
  RiLayoutLeftLine,
  RiPaintBrushLine,
  RiSmartphoneFill,
} from '@remixicon/vue';

defineProps<{
  selectedWidget: BusinessDashboardWidget | null;
  interactionMode: 'move' | 'resize';
  preview: 'desktop' | 'mobile';
  paletteCollapsed: boolean;
}>();

const emit = defineEmits<{
  updateInteractionMode: [mode: 'move' | 'resize'];
  updatePreview: [preview: 'desktop' | 'mobile'];
  togglePalette: [];
  openData: [];
  openStyle: [];
  remove: [id: string];
}>();
</script>

<template>
  <section class="designer-command-bar" aria-label="画布操作工具栏">
    <div class="designer-command-bar__group">
      <button
        class="designer-command-bar__button"
        :class="{ 'is-active': interactionMode === 'move' }"
        type="button"
        @click="emit('updateInteractionMode', 'move')"
      >
        <RiDragMove2Fill aria-hidden="true" />移动
      </button>
      <button
        class="designer-command-bar__button"
        :class="{ 'is-active': interactionMode === 'resize' }"
        type="button"
        @click="emit('updateInteractionMode', 'resize')"
      >
        <RiLayoutLeftLine aria-hidden="true" />调整尺寸
      </button>
      <button
        class="designer-command-bar__button"
        type="button"
        :disabled="!selectedWidget"
        @click="selectedWidget && emit('remove', selectedWidget.id)"
      >
        <RiDeleteBin6Line aria-hidden="true" />删除
      </button>
      <button class="designer-command-bar__button" type="button" @click="emit('togglePalette')">
        <RiLayoutLeftLine aria-hidden="true" />{{ paletteCollapsed ? '展开组件库' : '收起组件库' }}
      </button>
      <button class="designer-command-bar__button" type="button" @click="emit('openData')">
        <RiDatabase2Line aria-hidden="true" />数据配置
      </button>
    </div>

    <div class="designer-command-bar__group designer-command-bar__group--end">
      <div class="designer-command-bar__devices" aria-label="预览设备">
        <button
          type="button"
          aria-label="桌面画布"
          :class="{ 'is-active': preview === 'desktop' }"
          @click="emit('updatePreview', 'desktop')"
        >
          <RiComputerFill aria-hidden="true" />
        </button>
        <button
          type="button"
          aria-label="移动画布"
          :class="{ 'is-active': preview === 'mobile' }"
          @click="emit('updatePreview', 'mobile')"
        >
          <RiSmartphoneFill aria-hidden="true" />
        </button>
      </div>
      <span class="designer-command-bar__divider" />
      <button class="designer-command-bar__button" type="button" @click="emit('openStyle')">
        <RiPaintBrushLine aria-hidden="true" />仪表盘样式
      </button>
    </div>
  </section>
</template>

<style scoped>
.designer-command-bar {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  min-height: 48px;
  padding: 0 18px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-lighter);
}

.designer-command-bar__group {
  display: flex;
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.designer-command-bar__group--end {
  flex: none;
}

.designer-command-bar__button {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  height: 34px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
}

.designer-command-bar__button svg {
  width: 17px;
  height: 17px;
}

.designer-command-bar__button:hover:not(:disabled),
.designer-command-bar__button.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.designer-command-bar__button:disabled {
  color: var(--el-text-color-disabled);
  cursor: not-allowed;
}

.designer-command-bar__devices {
  display: flex;
  gap: 4px;
  align-items: center;
}

.designer-command-bar__devices button {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  padding: 0;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 5px;
}

.designer-command-bar__devices button.is-active {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}

.designer-command-bar__devices svg {
  width: 19px;
  height: 19px;
}

.designer-command-bar__divider {
  width: 1px;
  height: 24px;
  margin: 0 4px;
  background: var(--el-border-color);
}

@media (width <= 900px) {
  .designer-command-bar__button {
    justify-content: center;
    width: 34px;
    padding: 0;
    overflow: hidden;
    font-size: 0;
  }
}
</style>
