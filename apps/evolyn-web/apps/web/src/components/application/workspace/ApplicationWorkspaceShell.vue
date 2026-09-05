<script setup lang="ts">
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
  ApplicationWorkspaceMode,
} from './applicationWorkspace.types';
import type { ApplicationAssetType } from '../runtime/applicationAssetCatalog';
import type { ApplicationIcon } from '~/types';
import { shallowRef } from 'vue';
import ApplicationWorkspaceFormRuntime from '../runtime/ApplicationWorkspaceFormRuntime.vue';
import ApplicationEmptyState from '../runtime/ApplicationEmptyState.vue';
import ApplicationContentPlaceholder from './ApplicationContentPlaceholder.vue';
import ApplicationWorkspaceHeader from './ApplicationWorkspaceHeader.vue';
import ApplicationWorkspaceSidebar from './ApplicationWorkspaceSidebar.vue';

defineOptions({ name: 'ApplicationWorkspaceShell' });

const props = defineProps<{
  appCode: string;
  applicationName: string;
  applicationIcon: ApplicationIcon;
  assets: ApplicationWorkspaceAsset[];
  /** 当前选中资产；菜单为空（M2-菜单-1 常态）时为 null */
  activeAsset: ApplicationWorkspaceAsset | null;
  mode: ApplicationWorkspaceMode;
  /** 菜单数据源状态：loading 传递给侧栏渲染加载态 */
  menuStatus: 'loading' | 'ready';
  /** 创建请求进行中的资产类型：透传给空态引导页锁定卡片，防止重复创建。 */
  creatingAssetType: ApplicationAssetType | null;
}>();

const emit = defineEmits<{
  back: [];
  createAsset: [
    payload: { parent?: ApplicationWorkspaceAsset; type: ApplicationWorkspaceCreateAssetType },
  ];
  assetGuide: [];
  selectAsset: [asset: ApplicationWorkspaceAsset];
  assetAction: [
    payload: { asset: ApplicationWorkspaceAsset; action: ApplicationWorkspaceAssetAction },
  ];
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
      :active-asset-code="props.activeAsset?.code ?? ''"
      :collapsed="sidebarCollapsed"
      :menu-status="props.menuStatus"
      @back="emit('back')"
      @create-asset="emit('createAsset', $event)"
      @asset-guide="emit('assetGuide')"
      @select-asset="emit('selectAsset', $event)"
      @asset-action="emit('assetAction', $event)"
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
      <ApplicationWorkspaceFormRuntime
        v-if="props.activeAsset?.type === 'form' && props.mode === 'fill'"
        :app-code="props.appCode"
        :asset="props.activeAsset"
      />
      <!--
        菜单加载完成且无任何资产：内容区渲染与应用首页一致的创建引导页，
        复用 ApplicationEmptyState 单一实现；加载中仍走占位，避免引导页闪现。
        卡片选择桥接到既有 createAsset 链路（starter.type 是其类型的子集）。
      -->
      <ApplicationEmptyState
        v-else-if="props.menuStatus === 'ready' && props.assets.length === 0"
        :creating-asset-type="props.creatingAssetType"
        @select-asset="(starter) => emit('createAsset', { type: starter.type })"
        @learn-more="emit('assetGuide')"
      />
      <ApplicationContentPlaceholder v-else :asset="props.activeAsset" :mode="props.mode" />
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
    margin: var(--el-space-md) var(--el-space-md) var(--el-space-md) 0;
    overflow: hidden;
    flex-direction: column;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-large);
    box-shadow: var(--el-box-shadow-light);
  }
}

@media (max-width: 900px) {
  .application-workspace-shell {
    min-width: 860px;
  }
}
</style>
