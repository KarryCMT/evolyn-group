<script setup lang="ts">
import type {
  BusinessDashboardLayout,
  BusinessDashboardWidgetDescriptor,
} from '@evolyn.do/dashboard';
import type { DashboardFormDataSource } from '~/types';
import { findAvailableWidgetPosition } from '@evolyn.do/dashboard';
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import DashboardDataSourceDialog from '~/components/dashboard/designer/DashboardDataSourceDialog.vue';
import DashboardDesignerShell from '~/components/dashboard/designer/DashboardDesignerShell.vue';
import { useDashboardDesignWorkspaceContext } from '~/components/dashboard/designer/dashboardDesignWorkspace';

defineOptions({ name: 'DashboardDesignCanvasPage' });

const router = useRouter();
const workspace = useDashboardDesignWorkspaceContext();
const sourceDialogOpen = shallowRef(false);
const pendingCreation = shallowRef<{
  descriptor: BusinessDashboardWidgetDescriptor;
  layout?: BusinessDashboardLayout;
} | null>(null);

function requestCreation(
  descriptor: BusinessDashboardWidgetDescriptor,
  layout?: BusinessDashboardLayout,
) {
  if (descriptor.type !== 'chart') {
    workspace.designer.addWidgetAtLayout(
      descriptor,
      layout ??
        findAvailableWidgetPosition(
          workspace.designer.editingDocument.value.widgets,
          descriptor.defaultLayout,
        ),
    );
    return;
  }
  pendingCreation.value = { descriptor, ...(layout ? { layout } : {}) };
  sourceDialogOpen.value = true;
}

async function confirmSource(source: DashboardFormDataSource) {
  const pending = pendingCreation.value;
  if (!pending) return;
  const catalog = await workspace.dataCatalog.ensureCatalog(source.code);
  if (!catalog) {
    ElMessage.error(workspace.dataCatalog.errorMessage.value || '字段目录加载失败');
    return;
  }
  const layout =
    pending.layout ??
    findAvailableWidgetPosition(
      workspace.designer.editingDocument.value.widgets,
      pending.descriptor.defaultLayout,
    );
  const draft = workspace.chartEditor.beginCreate({
    descriptor: pending.descriptor,
    layout,
    source,
  });
  pendingCreation.value = null;
  sourceDialogOpen.value = false;
  await router.push({
    name: 'dashboard-widget-edit',
    params: {
      appCode: workspace.appCode.value,
      dashboardCode: workspace.dashboardCode.value,
      widgetId: draft.widget.id,
    },
  });
}

async function editWidget(id: string) {
  const widget = workspace.designer.editingDocument.value.widgets.find((item) => item.id === id);
  if (!widget || widget.type !== 'chart' || !widget.datasetId) {
    ElMessage.info('该组件暂不支持独立配置页');
    return;
  }
  const dataset = workspace.designer.editingDocument.value.datasets.find(
    (item) => item.id === widget.datasetId,
  );
  const source = workspace.dataCatalog.sources.value.find(
    (item) => item.code === dataset?.source.formCode,
  );
  if (!dataset || !source) {
    ElMessage.error('组件数据源已不可用，请重新选择');
    return;
  }
  const catalog = await workspace.dataCatalog.ensureCatalog(source.code);
  if (!catalog) {
    ElMessage.error(workspace.dataCatalog.errorMessage.value || '字段目录加载失败');
    return;
  }
  workspace.chartEditor.beginEdit({ widget, dataset, source });
  await router.push({
    name: 'dashboard-widget-edit',
    params: {
      appCode: workspace.appCode.value,
      dashboardCode: workspace.dashboardCode.value,
      widgetId: widget.id,
    },
  });
}

function removeDataset(id: string) {
  workspace.designer.removeDataset(id);
  workspace.selectedDatasetId.value =
    workspace.designer.editingDocument.value.datasets[0]?.id ?? null;
}

function addDataset(source: DashboardFormDataSource) {
  const id = workspace.designer.addFormDataset(source.code, source.name);
  workspace.selectedDatasetId.value = id;
  void workspace.dataCatalog.ensureCatalog(source.code);
}

function showNewFormHint() {
  ElMessage.info('可从应用工作区新建表单');
}
</script>

<template>
  <DashboardDesignerShell
    :name="workspace.designer.detail.value!.name"
    :document="workspace.designer.editingDocument.value"
    :selected-widget="workspace.designer.selectedWidget.value"
    :selected-widget-id="workspace.designer.selectedWidgetId.value"
    :issue-widget-ids="workspace.designer.issueWidgetIds.value"
    :issues="workspace.designer.issues.value"
    :focused-issue-path="workspace.designer.focusedIssuePath.value"
    :dirty="workspace.designer.isDirty.value"
    :save-status="workspace.designer.saveStatus.value"
    :renaming="workspace.renaming.value"
    :conflict-message="workspace.designer.errorMessage.value"
    :data-sources="workspace.dataCatalog.sources.value"
    :data-catalogs="workspace.dataCatalog.catalogs.value"
    :data-loading="workspace.dataCatalog.loading.value"
    :data-error-message="workspace.dataCatalog.errorMessage.value"
    :selected-dataset-id="workspace.selectedDatasetId.value"
    @back="workspace.returnToApp"
    @help="workspace.showHelp"
    @save="workspace.saveDraft"
    @preview="workspace.openPreview"
    @navigate="workspace.navigateWorkspace"
    @rename="workspace.renameDashboard"
    @reload-conflict="workspace.reloadConflict"
    @add="requestCreation"
    @drop="requestCreation"
    @edit="editWidget"
    @select="workspace.designer.selectWidget"
    @remove="workspace.designer.removeWidget"
    @duplicate="workspace.designer.duplicateWidget"
    @update-widget="workspace.designer.updateWidget"
    @update-layouts="workspace.designer.replaceLayouts"
    @update-desktop-settings="workspace.designer.updateDesktopSettings"
    @focus-issue="workspace.designer.focusIssue"
    @add-dataset="addDataset"
    @select-dataset="workspace.selectedDatasetId.value = $event"
    @update-dataset="workspace.designer.updateDataset"
    @remove-dataset="removeDataset"
    @request-catalog="workspace.dataCatalog.ensureCatalog"
  />
  <DashboardDataSourceDialog
    v-model="sourceDialogOpen"
    :sources="workspace.dataCatalog.sources.value"
    :loading="workspace.dataCatalog.loading.value"
    :error-message="workspace.dataCatalog.errorMessage.value"
    @confirm="confirmSource"
    @retry="workspace.dataCatalog.load(workspace.dashboardCode.value)"
    @new-form="showNewFormHint"
  />
</template>
