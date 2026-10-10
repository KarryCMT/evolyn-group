<script setup lang="ts">
import type { DashboardWorkspaceTab } from '~/components/dashboard/workspace/dashboardWorkspace.types';
import type { DashboardFormDataSource } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { updateDashboard } from '~/api/dashboard';
import { appWorkspaceRoute } from '~/components/app/workspace/appWorkspaceNavigation';
import DashboardDesignerShell from '~/components/dashboard/designer/DashboardDesignerShell.vue';
import { useDashboardDataCatalog } from '~/composables/useDashboardDataCatalog';
import { prepareDashboardPreview, useDashboardDesigner } from '~/composables/useDashboardDesigner';
import { useUnsavedChangesGuard } from '~/composables/useUnsavedChangesGuard';

defineOptions({ name: 'DashboardDesignPage' });

const route = useRoute();
const router = useRouter();
const dashboardCode = computed(() => String(route.params.dashboardCode ?? ''));
const appCode = computed(() => String(route.params.appCode ?? ''));
const designer = useDashboardDesigner();
const dataCatalog = useDashboardDataCatalog();
const selectedDatasetId = shallowRef<string | null>(null);
const renaming = shallowRef(false);

watch(
  dashboardCode,
  async (code) => {
    if (!code) return;
    const loaded = await designer.load(code);
    if (loaded && designer.detail.value) {
      document.title = `${designer.detail.value.name} - 仪表盘设计`;
      await dataCatalog.load(code);
      selectedDatasetId.value = designer.editingDocument.value.datasets[0]?.id ?? null;
      const selected = designer.editingDocument.value.datasets[0];
      if (selected) await dataCatalog.ensureCatalog(selected.source.formCode);
    }
  },
  { immediate: true },
);

useUnsavedChangesGuard({
  dirty: designer.isDirty,
  async confirmLeave() {
    try {
      await ElMessageBox.confirm('当前修改尚未保存，离开后将丢失本次编辑。', '未保存的修改', {
        confirmButtonText: '放弃修改并离开',
        cancelButtonText: '继续编辑',
        type: 'warning',
      });
      return true;
    } catch {
      return false;
    }
  },
});

async function saveDraft() {
  const result = await designer.save(dashboardCode.value);
  if (result) ElMessage.success('仪表盘草稿已保存');
  else if (designer.saveStatus.value === 'error' && designer.errorMessage.value) {
    ElMessage.error(designer.errorMessage.value);
  }
}

async function openPreview() {
  const revision = await prepareDashboardPreview({
    dirty: designer.isDirty.value,
    revision: designer.draftRevision.value,
    save: () => designer.save(dashboardCode.value),
  });
  if (revision === null) {
    if (designer.saveStatus.value !== 'conflict') {
      ElMessage.error(designer.errorMessage.value || '草稿保存失败，无法预览');
    }
    return;
  }
  await router.push({
    name: 'dashboard-preview',
    params: { appCode: appCode.value, dashboardCode: dashboardCode.value },
    query: { revision: String(revision) },
  });
}

function returnToApp() {
  void router.push(appWorkspaceRoute(appCode.value, dashboardCode.value));
}

/** 帮助中心尚未接入时提供明确反馈，避免图标按钮成为无响应入口。 */
function showHelp() {
  ElMessage.info('仪表盘帮助中心正在建设中');
}

function navigateWorkspace(tab: DashboardWorkspaceTab) {
  if (tab === 'design') return;
  if (tab === 'publish') {
    ElMessage.info('仪表盘发布能力正在建设中');
    return;
  }
  void router.push({
    name: 'dashboard-extensions',
    params: { appCode: appCode.value, dashboardCode: dashboardCode.value },
  });
}

/** 名称属于仪表盘资产，改名成功后只更新详情，不覆盖画布中的未保存草稿。 */
async function renameDashboard(name: string, onSuccess: () => void): Promise<void> {
  const code = dashboardCode.value;
  const normalizedName = name.trim();
  if (!code || renaming.value || !normalizedName) return;

  renaming.value = true;
  try {
    const detail = await updateDashboard(code, { name: normalizedName });
    // 请求期间可能切换到另一张仪表盘，旧响应不能覆盖当前工作区标题。
    if (dashboardCode.value !== code) return;
    designer.patchDetail({
      name: detail.name,
      icon: detail.icon,
      color: detail.color,
      updatedAt: detail.updatedAt,
    });
    document.title = `${detail.name} - 仪表盘设计`;
    onSuccess();
    ElMessage.success('仪表盘名称已修改');
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'DASHBOARD_NAME_INVALID') {
      ElMessage.error('仪表盘名称不能为空，且不能超过 128 个字符');
    } else if (error instanceof ApiError && error.errCode === 'FORBIDDEN') {
      ElMessage.error('没有修改仪表盘名称的权限');
    } else if (error instanceof ApiError && error.errCode === 'DASHBOARD_NOT_FOUND') {
      ElMessage.error('仪表盘已不存在，请返回应用后刷新');
    } else {
      ElMessage.error('仪表盘名称修改失败，请稍后重试');
    }
  } finally {
    renaming.value = false;
  }
}

async function reloadConflict() {
  try {
    await ElMessageBox.confirm(
      '重新加载会丢弃当前页面保留的本地修改，且不会自动合并。',
      '重新加载服务端版本',
      { confirmButtonText: '重新加载', cancelButtonText: '保留本地修改', type: 'warning' },
    );
  } catch {
    return;
  }
  await designer.reloadServerVersion(dashboardCode.value);
}

async function addDataset(source: DashboardFormDataSource) {
  const existing = designer.editingDocument.value.datasets.find(
    (item) => item.source.formCode === source.code,
  );
  const id = existing?.id ?? designer.addFormDataset(source.code, source.name);
  selectedDatasetId.value = id;
  await dataCatalog.ensureCatalog(source.code);
}

function removeDataset(id: string) {
  designer.removeDataset(id);
  selectedDatasetId.value = designer.editingDocument.value.datasets[0]?.id ?? null;
}
</script>

<template>
  <section
    v-if="designer.loadStatus.value === 'loading'"
    v-loading="true"
    class="design-page__state"
  />
  <el-result
    v-else-if="designer.loadStatus.value === 'error'"
    class="design-page__state"
    icon="error"
    title="无法加载仪表盘"
    :sub-title="designer.errorMessage.value"
  >
    <template #extra>
      <el-button @click="returnToApp">
        返回应用
      </el-button>
      <el-button type="primary" @click="designer.load(dashboardCode)">
        重新加载
      </el-button>
    </template>
  </el-result>
  <DashboardDesignerShell
    v-else-if="designer.loadStatus.value === 'ready' && designer.detail.value"
    :name="designer.detail.value.name"
    :document="designer.editingDocument.value"
    :selected-widget="designer.selectedWidget.value"
    :selected-widget-id="designer.selectedWidgetId.value"
    :issue-widget-ids="designer.issueWidgetIds.value"
    :issues="designer.issues.value"
    :focused-issue-path="designer.focusedIssuePath.value"
    :dirty="designer.isDirty.value"
    :save-status="designer.saveStatus.value"
    :renaming="renaming"
    :conflict-message="designer.errorMessage.value"
    :data-sources="dataCatalog.sources.value"
    :data-catalogs="dataCatalog.catalogs.value"
    :data-loading="dataCatalog.loading.value"
    :data-error-message="dataCatalog.errorMessage.value"
    :selected-dataset-id="selectedDatasetId"
    @back="returnToApp"
    @help="showHelp"
    @save="saveDraft"
    @preview="openPreview"
    @navigate="navigateWorkspace"
    @rename="renameDashboard"
    @reload-conflict="reloadConflict"
    @add="designer.addWidget"
    @drop="designer.addWidgetAtLayout"
    @select="designer.selectWidget"
    @remove="designer.removeWidget"
    @duplicate="designer.duplicateWidget"
    @update-widget="designer.updateWidget"
    @update-layouts="designer.replaceLayouts"
    @update-desktop-settings="designer.updateDesktopSettings"
    @focus-issue="designer.focusIssue"
    @add-dataset="addDataset"
    @select-dataset="selectedDatasetId = $event"
    @update-dataset="designer.updateDataset"
    @remove-dataset="removeDataset"
    @request-catalog="dataCatalog.ensureCatalog"
  />
</template>

<style scoped>
.design-page__state {
  min-height: 100vh;
  background: #eef2f5;
}
</style>
