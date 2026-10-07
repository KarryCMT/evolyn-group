<script setup lang="ts">
import type {
  BusinessDashboardDatasetPatch,
  BusinessDashboardDocument,
  BusinessDashboardIssue,
  BusinessDashboardLayout,
  BusinessDashboardWidget,
  BusinessDashboardWidgetDescriptor,
  BusinessDashboardWidgetPatch,
} from '@evolyn.do/dashboard';
import type { DashboardDesignerSaveStatus } from '~/composables/useDashboardDesigner';
import type { DashboardFormDataSource, DashboardFormFieldCatalog } from '~/types';
import { BusinessDashboardCanvas, businessDashboardWidgetDescriptors } from '@evolyn.do/dashboard';
import DashboardComponentPalette from './DashboardComponentPalette.vue';
import DashboardDataPanel from './DashboardDataPanel.vue';
import DashboardDesignerToolbar from './DashboardDesignerToolbar.vue';
import DashboardPropertiesPanel from './DashboardPropertiesPanel.vue';

defineProps<{
  name: string;
  revision: number;
  document: BusinessDashboardDocument;
  selectedWidget: BusinessDashboardWidget | null;
  selectedWidgetId: string | null;
  issueWidgetIds: string[];
  issues: BusinessDashboardIssue[];
  focusedIssuePath: string;
  dirty: boolean;
  saveStatus: DashboardDesignerSaveStatus;
  conflictMessage: string;
  dataSources: DashboardFormDataSource[];
  dataCatalogs: Record<string, DashboardFormFieldCatalog>;
  dataLoading: boolean;
  dataErrorMessage: string;
  selectedDatasetId: string | null;
}>();
const emit = defineEmits<{
  back: [];
  save: [];
  preview: [];
  reloadConflict: [];
  add: [descriptor: BusinessDashboardWidgetDescriptor];
  select: [id: string | null];
  remove: [id: string];
  updateWidget: [id: string, patch: BusinessDashboardWidgetPatch];
  updateLayouts: [layouts: Array<{ id: string; layout: BusinessDashboardLayout }>];
  focusIssue: [issue: BusinessDashboardIssue];
  addDataset: [source: DashboardFormDataSource];
  selectDataset: [id: string | null];
  updateDataset: [id: string, patch: BusinessDashboardDatasetPatch];
  removeDataset: [id: string];
  requestCatalog: [formCode: string];
}>();
</script>

<template>
  <main class="designer-shell">
    <DashboardDesignerToolbar
      :name="name"
      :revision="revision"
      :dirty="dirty"
      :save-status="saveStatus"
      @back="emit('back')"
      @save="emit('save')"
      @preview="emit('preview')"
    />
    <section v-if="saveStatus === 'conflict'" class="designer-shell__conflict" role="alert">
      <div>
        <strong>服务端草稿已经更新</strong>
        <span>{{ conflictMessage }} 当前本地修改尚未被覆盖。</span>
      </div>
      <el-button type="danger" plain @click="emit('reloadConflict')">
        重新加载服务端版本
      </el-button>
    </section>
    <section class="designer-shell__workspace">
      <DashboardComponentPalette
        :descriptors="businessDashboardWidgetDescriptors"
        @add="emit('add', $event)"
      />
      <DashboardDataPanel
        :sources="dataSources"
        :datasets="document.datasets"
        :catalogs="dataCatalogs"
        :loading="dataLoading"
        :error-message="dataErrorMessage"
        :selected-dataset-id="selectedDatasetId"
        @add-dataset="emit('addDataset', $event)"
        @select-dataset="emit('selectDataset', $event)"
        @update-dataset="(id, patch) => emit('updateDataset', id, patch)"
        @remove-dataset="emit('removeDataset', $event)"
        @request-catalog="emit('requestCatalog', $event)"
      />
      <BusinessDashboardCanvas
        :document="document"
        :selected-widget-id="selectedWidgetId"
        :issue-widget-ids="issueWidgetIds"
        @select="emit('select', $event || null)"
        @remove="emit('remove', $event)"
        @update-layouts="emit('updateLayouts', $event)"
      />
      <DashboardPropertiesPanel
        :widget="selectedWidget"
        :issues="issues"
        :focused-issue-path="focusedIssuePath"
        :datasets="document.datasets"
        :catalogs="dataCatalogs"
        @update="(id, patch) => emit('updateWidget', id, patch)"
        @remove="emit('remove', $event)"
        @focus-issue="emit('focusIssue', $event)"
        @request-catalog="emit('requestCatalog', $event)"
      />
    </section>
  </main>
</template>

<style scoped>
.designer-shell {
  display: flex;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  background: #eef2f5;
}
.designer-shell__workspace {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.designer-shell__conflict {
  display: flex;
  padding: 10px 18px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  color: #7d3828;
  background: #fff2ed;
  border-bottom: 1px solid #edc9bd;
}
.designer-shell__conflict div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.designer-shell__conflict strong {
  font-size: 13px;
}
.designer-shell__conflict span {
  font-size: 11px;
}
@media (max-width: 980px) {
  .designer-shell__workspace :deep(.component-palette) {
    flex-basis: 190px;
  }
  .designer-shell__workspace :deep(.properties-panel) {
    flex-basis: 240px;
  }
}
</style>
