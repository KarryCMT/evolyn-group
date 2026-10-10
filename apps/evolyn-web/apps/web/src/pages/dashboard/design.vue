<script setup lang="ts">
import type { DashboardWorkspaceTab } from '~/components/dashboard/workspace/dashboardWorkspace.types';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, provide, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { updateDashboard } from '~/api/dashboard';
import { appWorkspaceRoute } from '~/components/app/workspace/appWorkspaceNavigation';
import { dashboardDesignWorkspaceKey } from '~/components/dashboard/designer/dashboardDesignWorkspace';
import { useDashboardChartEditorSession } from '~/composables/useDashboardChartEditorSession';
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
const chartEditor = useDashboardChartEditorSession({
  commit(widget, dataset) {
    designer.upsertWidgetWithDataset(widget, dataset);
    selectedDatasetId.value = dataset.id;
  },
});
const workspaceDirty = computed(() => designer.isDirty.value || chartEditor.isDirty.value);

watch(
  dashboardCode,
  async (code) => {
    if (!code) return;
    chartEditor.discard();
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
  dirty: workspaceDirty,
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
    // 请求返回时路由可能已经切换，避免旧请求覆盖新仪表盘的标题与详情。
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

provide(dashboardDesignWorkspaceKey, {
  appCode,
  dashboardCode,
  designer,
  dataCatalog,
  chartEditor,
  selectedDatasetId,
  renaming,
  saveDraft,
  openPreview,
  returnToApp,
  showHelp,
  navigateWorkspace,
  renameDashboard,
  reloadConflict,
});
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
  <RouterView v-else-if="designer.loadStatus.value === 'ready' && designer.detail.value" />
</template>

<style scoped>
.design-page__state {
  min-height: 100vh;
  background: var(--el-bg-color-page);
}
</style>
