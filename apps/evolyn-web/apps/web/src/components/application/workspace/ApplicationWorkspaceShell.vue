<script setup lang="ts">
import { shallowRef } from 'vue';
import type { ApplicationIcon } from '~/types';
import ApplicationContentPlaceholder from './ApplicationContentPlaceholder.vue';
import ApplicationWorkspaceHeader from './ApplicationWorkspaceHeader.vue';
import ApplicationWorkspaceSidebar from './ApplicationWorkspaceSidebar.vue';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceMode,
} from './applicationWorkspace.types';

defineOptions({ name: 'ApplicationWorkspaceShell' });

const props = defineProps<{
  applicationName: string;
  applicationIcon: ApplicationIcon;
  assets: ApplicationWorkspaceAsset[];
  activeAsset: ApplicationWorkspaceAsset;
  mode: ApplicationWorkspaceMode;
}>();

const emit = defineEmits<{
  back: [];
  createAsset: [];
  selectAsset: [asset: ApplicationWorkspaceAsset];
  openManagement: [];
  updateMode: [mode: ApplicationWorkspaceMode];
}>();

// 工作区统一持有侧栏展开状态，侧栏与内容头部通过显式 props / emits 保持同步。
const sidebarCollapsed = shallowRef(false);

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}
</script>

<template>
  <div class="application-workspace-shell">
    <ApplicationWorkspaceSidebar
      :application-name="props.applicationName"
      :application-icon="props.applicationIcon"
      :assets="props.assets"
      :active-asset-code="props.activeAsset.code"
      :collapsed="sidebarCollapsed"
      @back="emit('back')"
      @create-asset="emit('createAsset')"
      @select-asset="emit('selectAsset', $event)"
      @open-management="emit('openManagement')"
      @toggle-sidebar="toggleSidebar"
    />
    <section class="application-workspace-shell__surface">
      <ApplicationWorkspaceHeader
        :mode="props.mode"
        :sidebar-collapsed="sidebarCollapsed"
        @toggle-sidebar="toggleSidebar"
        @update-mode="emit('updateMode', $event)"
      />
      <ApplicationContentPlaceholder :asset="props.activeAsset" :mode="props.mode" />
    </section>
  </div>
</template>

<style scoped lang="scss">
.application-workspace-shell {
  display: flex;
  height: 100vh;
  min-width: 0;
  overflow: hidden;
  background: var(--el-color-primary);

  &__surface {
    display: flex;
    min-width: 0;
    min-height: 0;
    flex: 1;
    margin: 10px 10px 10px 0;
    overflow: hidden;
    flex-direction: column;
    background: var(--el-bg-color);
    border-radius: 14px;
    box-shadow: var(--el-box-shadow-light);
  }
}

@media (max-width: 900px) {
  .application-workspace-shell {
    min-width: 860px;
  }
}
</style>
