<script setup lang="ts">
import type { DashboardWorkspaceTab } from '../workspace/dashboardWorkspace.types';
import type { DashboardDesignerSaveStatus } from '~/composables/useDashboardDesigner';
import { RiArrowLeftLine, RiEyeLine, RiQuestionFill, RiSave3Line } from '@remixicon/vue';
import WorkspaceTitleEditor from '~/components/WorkspaceTitleEditor.vue';
import DashboardWorkspaceNavigation from '../workspace/DashboardWorkspaceNavigation.vue';

defineProps<{
  name: string;
  dirty: boolean;
  saveStatus: DashboardDesignerSaveStatus;
  renaming?: boolean;
}>();
const emit = defineEmits<{
  back: [];
  help: [];
  save: [];
  preview: [];
  navigate: [tab: DashboardWorkspaceTab];
  rename: [name: string, onSuccess: () => void];
}>();
</script>

<template>
  <header class="designer-toolbar">
    <button
      class="designer-toolbar__back"
      type="button"
      aria-label="返回应用"
      @click="emit('back')"
    >
      <RiArrowLeftLine />
    </button>
    <WorkspaceTitleEditor
      class="designer-toolbar__identity"
      :name="name"
      resource-label="仪表盘"
      :saving="renaming"
      @submit="(name, onSuccess) => emit('rename', name, onSuccess)"
    />
    <DashboardWorkspaceNavigation
      class="designer-toolbar__tabs"
      active-tab="design"
      publish-disabled
      @navigate="emit('navigate', $event)"
    />
    <div class="designer-toolbar__utility">
      <el-tooltip content="帮助" placement="bottom">
        <button
          class="designer-toolbar__help"
          type="button"
          aria-label="帮助"
          @click="emit('help')"
        >
          <RiQuestionFill aria-hidden="true" />
        </button>
      </el-tooltip>
    </div>
    <div class="designer-toolbar__actions">
      <el-button :disabled="saveStatus === 'saving'" @click="emit('preview')">
        <RiEyeLine />预览
      </el-button>
      <el-button
        type="primary"
        :disabled="!dirty || saveStatus === 'saving'"
        :loading="saveStatus === 'saving'"
        @click="emit('save')"
      >
        <RiSave3Line />保存
      </el-button>
    </div>
  </header>
</template>

<style scoped>
/* Vue 的 :deep() 用于统一 Element Plus 按钮内图标尺寸。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.designer-toolbar {
  display: grid;
  grid-template-columns: 36px minmax(180px, 1fr) auto minmax(180px, 1fr) auto;
  gap: 12px;
  align-items: center;
  min-height: 56px;
  padding: 0 18px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.designer-toolbar__back {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  font-size: 19px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
}

.designer-toolbar__back:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.designer-toolbar__identity {
  min-width: 0;
}

.designer-toolbar__utility {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.designer-toolbar__help {
  display: inline-grid;
  width: 32px;
  height: 32px;
  padding: 0;
  font-size: 20px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
  place-items: center;
}

.designer-toolbar__help:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.designer-toolbar__help:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.designer-toolbar__help svg {
  width: 20px;
  height: 20px;
}

.designer-toolbar__actions {
  display: flex;
  gap: 8px;
}

.designer-toolbar__actions :deep(svg) {
  width: 16px;
  margin-right: 5px;
}

@media (width <= 760px) {
  .designer-toolbar {
    grid-template-columns: 36px 1fr auto;
  }

  .designer-toolbar__tabs,
  .designer-toolbar__utility {
    display: none;
  }
}
</style>
