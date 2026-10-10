<script setup lang="ts">
import type { DashboardWorkspaceTab } from '../workspace/dashboardWorkspace.types';
import { RiArrowLeftLine, RiQuestionFill } from '@remixicon/vue';
import DashboardWorkspaceNavigation from '../workspace/DashboardWorkspaceNavigation.vue';

defineOptions({ name: 'DashboardExtensionHeader' });

defineProps<{
  name: string;
}>();

const emit = defineEmits<{
  back: [];
  help: [];
  navigate: [tab: DashboardWorkspaceTab];
}>();
</script>

<template>
  <header class="dashboard-extension-header">
    <div class="dashboard-extension-header__identity">
      <button
        class="dashboard-extension-header__icon-button"
        type="button"
        aria-label="返回应用"
        @click="emit('back')"
      >
        <RiArrowLeftLine aria-hidden="true" />
      </button>
      <strong class="dashboard-extension-header__name" :title="name">{{ name }}</strong>
    </div>

    <DashboardWorkspaceNavigation
      class="dashboard-extension-header__navigation"
      active-tab="extensions"
      publish-disabled
      @navigate="emit('navigate', $event)"
    />

    <div class="dashboard-extension-header__utility">
      <el-tooltip content="帮助" placement="bottom">
        <button
          class="dashboard-extension-header__icon-button"
          type="button"
          aria-label="帮助"
          @click="emit('help')"
        >
          <RiQuestionFill aria-hidden="true" />
        </button>
      </el-tooltip>
    </div>
  </header>
</template>

<style scoped>
.dashboard-extension-header {
  position: relative;
  display: grid;
  grid-template-columns: minmax(220px, 1fr) auto minmax(220px, 1fr);
  align-items: center;
  min-height: 56px;
  padding: 0 18px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 2px 10px rgb(31 45 61 / 7%);
  z-index: 2;
}

.dashboard-extension-header__identity,
.dashboard-extension-header__utility {
  display: flex;
  align-items: center;
}

.dashboard-extension-header__identity {
  min-width: 0;
  gap: 10px;
}

.dashboard-extension-header__utility {
  justify-content: flex-end;
}

.dashboard-extension-header__name {
  overflow: hidden;
  font-size: 15px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-extension-header__icon-button {
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

.dashboard-extension-header__icon-button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.dashboard-extension-header__icon-button:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.dashboard-extension-header__icon-button svg {
  width: 20px;
  height: 20px;
}

@media (width <= 760px) {
  .dashboard-extension-header {
    grid-template-columns: 1fr auto;
  }

  .dashboard-extension-header__navigation {
    display: none;
  }
}
</style>
