<script setup lang="ts">
import type { DashboardWorkspaceTab } from './dashboardWorkspace.types';

defineOptions({ name: 'DashboardWorkspaceNavigation' });

defineProps<{
  activeTab: DashboardWorkspaceTab;
  publishDisabled?: boolean;
}>();

const emit = defineEmits<{
  navigate: [tab: DashboardWorkspaceTab];
}>();

const items: Array<{ tab: DashboardWorkspaceTab; label: string }> = [
  { tab: 'design', label: '仪表盘设计' },
  { tab: 'extensions', label: '扩展功能' },
  { tab: 'publish', label: '仪表盘发布' },
];
</script>

<template>
  <nav class="dashboard-workspace-navigation" aria-label="仪表盘管理导航">
    <button
      v-for="item in items"
      :key="item.tab"
      class="dashboard-workspace-navigation__item"
      :class="{ 'is-active': activeTab === item.tab }"
      type="button"
      :disabled="item.tab === 'publish' && publishDisabled"
      :aria-current="activeTab === item.tab ? 'page' : undefined"
      :title="item.tab === 'publish' && publishDisabled ? '发布能力即将开放' : undefined"
      @click="emit('navigate', item.tab)"
    >
      {{ item.label }}
    </button>
  </nav>
</template>

<style scoped>
.dashboard-workspace-navigation {
  display: flex;
  gap: 38px;
  align-items: stretch;
  justify-content: center;
  height: 56px;
}

.dashboard-workspace-navigation__item {
  position: relative;
  padding: 0;
  font-size: 15px;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.dashboard-workspace-navigation__item:hover:not(:disabled),
.dashboard-workspace-navigation__item.is-active {
  color: var(--el-color-primary);
}

.dashboard-workspace-navigation__item.is-active {
  font-weight: 600;
}

.dashboard-workspace-navigation__item.is-active::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 3px;
  content: '';
  background: var(--el-color-primary);
}

.dashboard-workspace-navigation__item:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -5px;
}

.dashboard-workspace-navigation__item:disabled {
  color: var(--el-text-color-disabled);
  cursor: not-allowed;
}
</style>
