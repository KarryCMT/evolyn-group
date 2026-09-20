<script setup lang="ts">
import type { AppAssetType } from '../runtime/appAssetCatalog';
import type {
  AppWorkspaceAsset,
  AppWorkspaceAssetAction,
  AppWorkspaceCreateAssetType,
  AppWorkspaceMode,
} from './appWorkspace.types';
import type { WorkflowNavigationForm } from '~/components/workflow-center/WorkflowCenterNavigation.vue';
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import type { AppIcon, WorkflowPendingTaskSummaryDto } from '~/types';
import { computed, shallowRef } from 'vue';
import WorkflowCenter from '~/components/workflow-center/WorkflowCenter.vue';
import AppEmptyState from '../runtime/AppEmptyState.vue';
import AppWorkspaceFormRuntime from '../runtime/AppWorkspaceFormRuntime.vue';
import AppContentPlaceholder from './AppContentPlaceholder.vue';
import AppWorkspaceAppSwitcherDrawer from './AppWorkspaceAppSwitcherDrawer.vue';
import AppWorkspaceHeader from './AppWorkspaceHeader.vue';
import AppWorkspaceSidebar from './AppWorkspaceSidebar.vue';

defineOptions({ name: 'AppWorkspaceShell' });

const props = defineProps<{
  appCode: string;
  appName: string;
  appIcon: AppIcon;
  assets: AppWorkspaceAsset[];
  /** 当前选中资产；菜单为空（M2-菜单-1 常态）时为 null */
  activeAsset: AppWorkspaceAsset | null;
  /** 当前高亮的个人入口；应用资产页传空字符串。 */
  activePersonalCode: string;
  /** 非空时，个人审批视图在应用内容区内渲染，而不是跳出应用壳。 */
  personalScope: WorkflowCenterScope | null;
  /** 个人视图显示在顶栏的标题。 */
  personalTitle: string | null;
  /** 个人待办菜单当前选中的流程表单；空串代表全部待办。 */
  activeWorkflowFormCode: string;
  /** 当前成员有真实待办的流程表单；不能传入完整应用表单目录。 */
  pendingWorkflowForms: readonly WorkflowNavigationForm[];
  pendingWorkflowSummary: WorkflowPendingTaskSummaryDto | null;
  mode: AppWorkspaceMode;
  /** 菜单数据源状态：loading 传递给侧栏渲染加载态 */
  menuStatus: 'loading' | 'ready';
  /** 创建请求进行中的资产类型：透传给空态引导页锁定卡片，防止重复创建。 */
  creatingAssetType: AppAssetType | null;
}>();

const emit = defineEmits<{
  back: [];
  createAsset: [payload: { parent?: AppWorkspaceAsset; type: AppWorkspaceCreateAssetType }];
  assetGuide: [];
  selectAsset: [asset: AppWorkspaceAsset];
  assetAction: [payload: { asset: AppWorkspaceAsset; action: AppWorkspaceAssetAction }];
  selectPersonalNavigation: [code: string];
  updatePersonalScope: [scope: WorkflowCenterScope];
  updatePersonalWorkflowFormCode: [formCode: string];
  updatePendingWorkflowSummary: [summary: WorkflowPendingTaskSummaryDto | null];
  openManagement: [];
  openFavorites: [];
  selectApp: [appCode: string];
  updateMode: [mode: AppWorkspaceMode];
}>();

// 工作区统一持有侧栏展开状态，侧栏与内容头部通过显式 props / emits 保持同步。
const sidebarCollapsed = shallowRef(false);
// 应用切换抽屉属于工作区壳的短生命周期 UI 状态，切换路由后自然重置。
const appSwitcherVisible = shallowRef(false);
// 个人流程视图与资产树是两套独立导航。保留最近资产用于返回后恢复，但在流程
// 视图期间不向资产树投影选中态，避免「我处理的」与某个表单同时高亮。
const visibleActiveAssetCode = computed(() =>
  props.personalScope ? '' : (props.activeAsset?.code ?? ''),
);

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}
</script>

<template>
  <div class="app-workspace-shell">
    <AppWorkspaceSidebar
      :app-name="props.appName"
      :app-icon="props.appIcon"
      :assets="props.assets"
      :active-asset-code="visibleActiveAssetCode"
      :active-personal-code="props.activePersonalCode"
      :collapsed="sidebarCollapsed"
      :menu-status="props.menuStatus"
      :workflow-scope="props.personalScope"
      :active-workflow-form-code="props.activeWorkflowFormCode"
      :pending-workflow-forms="props.pendingWorkflowForms"
      :pending-workflow-summary="props.pendingWorkflowSummary"
      @back="emit('back')"
      @create-asset="emit('createAsset', $event)"
      @asset-guide="emit('assetGuide')"
      @select-asset="emit('selectAsset', $event)"
      @asset-action="emit('assetAction', $event)"
      @select-personal-navigation="emit('selectPersonalNavigation', $event)"
      @update-workflow-scope="emit('updatePersonalScope', $event)"
      @update-workflow-form-code="emit('updatePersonalWorkflowFormCode', $event)"
      @open-management="emit('openManagement')"
      @open-app-switcher="appSwitcherVisible = true"
      @open-favorites="emit('openFavorites')"
      @toggle-sidebar="toggleSidebar"
    />
    <section class="app-workspace-shell__surface">
      <AppWorkspaceHeader
        :mode="props.mode"
        :sidebar-collapsed="sidebarCollapsed"
        :personal-title="props.personalTitle"
        @toggle-sidebar="toggleSidebar"
        @update-mode="emit('updateMode', $event)"
      />
      <WorkflowCenter
        v-if="props.personalScope"
        embedded
        :scope="props.personalScope"
        :form-code="props.activeWorkflowFormCode"
        @update-scope="emit('updatePersonalScope', $event)"
        @pending-summary="emit('updatePendingWorkflowSummary', $event)"
      />
      <AppWorkspaceFormRuntime
        v-else-if="props.activeAsset?.type === 'form' && props.mode === 'fill'"
        :app-code="props.appCode"
        :asset="props.activeAsset"
      />
      <!--
        菜单加载完成且无任何资产：内容区渲染与应用首页一致的创建引导页，
        复用 AppEmptyState 单一实现；加载中仍走占位，避免引导页闪现。
        卡片选择桥接到既有 createAsset 链路（starter.type 是其类型的子集）。
      -->
      <AppEmptyState
        v-else-if="props.menuStatus === 'ready' && props.assets.length === 0"
        :creating-asset-type="props.creatingAssetType"
        @select-asset="(starter) => emit('createAsset', { type: starter.type })"
        @learn-more="emit('assetGuide')"
      />
      <AppContentPlaceholder v-else :asset="props.activeAsset" :mode="props.mode" />
    </section>
    <AppWorkspaceAppSwitcherDrawer
      v-model="appSwitcherVisible"
      :active-app-code="props.appCode"
      @back="emit('back')"
      @select-app="emit('selectApp', $event)"
    />
  </div>
</template>

<style scoped lang="scss">
.app-workspace-shell {
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
  .app-workspace-shell {
    min-width: 860px;
  }
}
</style>
