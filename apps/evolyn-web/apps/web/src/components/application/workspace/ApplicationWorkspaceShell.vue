<script setup lang="ts">
import type { ApplicationAssetType } from '../runtime/applicationAssetCatalog';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
  ApplicationWorkspaceMode,
} from './applicationWorkspace.types';
import type { WorkflowNavigationForm } from '~/components/workflow-center/WorkflowCenterNavigation.vue';
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import type { ApplicationIcon, WorkflowPendingTaskSummaryDto } from '~/types';
import { computed, shallowRef } from 'vue';
import WorkflowCenter from '~/components/workflow-center/WorkflowCenter.vue';
import ApplicationEmptyState from '../runtime/ApplicationEmptyState.vue';
import ApplicationWorkspaceFormRuntime from '../runtime/ApplicationWorkspaceFormRuntime.vue';
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
  selectPersonalNavigation: [code: string];
  updatePersonalScope: [scope: WorkflowCenterScope];
  updatePersonalWorkflowFormCode: [formCode: string];
  updatePendingWorkflowSummary: [summary: WorkflowPendingTaskSummaryDto | null];
  openManagement: [];
  updateMode: [mode: ApplicationWorkspaceMode];
}>();

// 工作区统一持有侧栏展开状态，侧栏与内容头部通过显式 props / emits 保持同步。
const sidebarCollapsed = shallowRef(false);
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
  <div class="application-workspace-shell">
    <ApplicationWorkspaceSidebar
      :application-name="props.applicationName"
      :application-icon="props.applicationIcon"
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
      @toggle-sidebar="toggleSidebar"
    />
    <section class="application-workspace-shell__surface">
      <ApplicationWorkspaceHeader
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
      <ApplicationWorkspaceFormRuntime
        v-else-if="props.activeAsset?.type === 'form' && props.mode === 'fill'"
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
