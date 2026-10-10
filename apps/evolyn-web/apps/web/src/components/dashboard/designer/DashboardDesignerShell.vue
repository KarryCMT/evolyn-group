<script setup lang="ts">
import type {
  BusinessDashboardDatasetPatch,
  BusinessDashboardDocument,
  BusinessDashboardIssue,
  BusinessDashboardLayout,
  BusinessDashboardWidget,
  BusinessDashboardWidgetDescriptor,
  BusinessDashboardWidgetPatch,
  BusinessDashboardWidgetType,
} from '@evolyn.do/dashboard';
import type { DashboardWorkspaceTab } from '../workspace/dashboardWorkspace.types';
import type { DashboardDesignerSaveStatus } from '~/composables/useDashboardDesigner';
import type { DashboardFormDataSource, DashboardFormFieldCatalog } from '~/types';
import { BusinessDashboardCanvas, businessDashboardWidgetDescriptors } from '@evolyn.do/dashboard';
import { RiCloseLine } from '@remixicon/vue';
import { shallowRef, watch } from 'vue';
import dashboardEmptyState from '~/assets/images/dashboard-empty-state.png';
import { isDark } from '~/composables/dark';
import DashboardComponentPalette from './DashboardComponentPalette.vue';
import DashboardDataPanel from './DashboardDataPanel.vue';
import DashboardDesignerCommandBar from './DashboardDesignerCommandBar.vue';
import DashboardDesignerToolbar from './DashboardDesignerToolbar.vue';
import DashboardPropertiesPanel from './DashboardPropertiesPanel.vue';
import DashboardStylePanel from './DashboardStylePanel.vue';

const props = defineProps<{
  name: string;
  document: BusinessDashboardDocument;
  selectedWidget: BusinessDashboardWidget | null;
  selectedWidgetId: string | null;
  issueWidgetIds: string[];
  issues: BusinessDashboardIssue[];
  focusedIssuePath: string;
  dirty: boolean;
  saveStatus: DashboardDesignerSaveStatus;
  renaming: boolean;
  conflictMessage: string;
  dataSources: DashboardFormDataSource[];
  dataCatalogs: Record<string, DashboardFormFieldCatalog>;
  dataLoading: boolean;
  dataErrorMessage: string;
  selectedDatasetId: string | null;
}>();
const emit = defineEmits<{
  back: [];
  help: [];
  save: [];
  preview: [];
  navigate: [tab: DashboardWorkspaceTab];
  rename: [name: string, onSuccess: () => void];
  reloadConflict: [];
  add: [descriptor: BusinessDashboardWidgetDescriptor];
  drop: [descriptor: BusinessDashboardWidgetDescriptor, layout: BusinessDashboardLayout];
  select: [id: string | null];
  remove: [id: string];
  duplicate: [id: string];
  edit: [id: string];
  updateWidget: [id: string, patch: BusinessDashboardWidgetPatch];
  updateLayouts: [layouts: Array<{ id: string; layout: BusinessDashboardLayout }>];
  updateDesktopSettings: [patch: Partial<BusinessDashboardDocument['settings']['desktop']>];
  focusIssue: [issue: BusinessDashboardIssue];
  addDataset: [source: DashboardFormDataSource];
  selectDataset: [id: string | null];
  updateDataset: [id: string, patch: BusinessDashboardDatasetPatch];
  removeDataset: [id: string];
  requestCatalog: [formCode: string];
}>();

// 面板显隐与画布工具属于当前编辑会话，不写入仪表盘业务协议。
const paletteCollapsed = shallowRef(false);
const dataPanelOpen = shallowRef(false);
const propertiesPanelOpen = shallowRef(false);
const stylePanelOpen = shallowRef(false);
const interactionMode = shallowRef<'move' | 'resize'>('move');
const preview = shallowRef<'desktop' | 'mobile'>('desktop');

// 保存校验定位到具体组件时自动打开设置抽屉，避免只高亮而没有修复入口。
watch(
  () => [props.focusedIssuePath, props.selectedWidgetId] as const,
  ([issuePath, widgetID]) => {
    if (issuePath && widgetID) openWidgetProperties(widgetID);
    if (!widgetID) propertiesPanelOpen.value = false;
  },
);

function openWidgetProperties(id?: string) {
  if (id) emit('select', id);
  if (id || props.selectedWidget) {
    dataPanelOpen.value = false;
    stylePanelOpen.value = false;
    propertiesPanelOpen.value = true;
  }
}

function openDataPanel() {
  propertiesPanelOpen.value = false;
  stylePanelOpen.value = false;
  dataPanelOpen.value = true;
}

function openStylePanel() {
  dataPanelOpen.value = false;
  propertiesPanelOpen.value = false;
  stylePanelOpen.value = true;
}

function dropWidget(type: BusinessDashboardWidgetType, layout: BusinessDashboardLayout) {
  const descriptor = businessDashboardWidgetDescriptors.find((item) => item.type === type);
  if (descriptor) emit('drop', descriptor, layout);
}
</script>

<template>
  <main class="designer-shell">
    <DashboardDesignerToolbar
      :name="name"
      :dirty="dirty"
      :save-status="saveStatus"
      :renaming="renaming"
      @back="emit('back')"
      @help="emit('help')"
      @save="emit('save')"
      @preview="emit('preview')"
      @navigate="emit('navigate', $event)"
      @rename="(name, onSuccess) => emit('rename', name, onSuccess)"
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
    <DashboardDesignerCommandBar
      :selected-widget="selectedWidget"
      :interaction-mode="interactionMode"
      :preview="preview"
      :palette-collapsed="paletteCollapsed"
      @update-interaction-mode="interactionMode = $event"
      @update-preview="preview = $event"
      @toggle-palette="paletteCollapsed = !paletteCollapsed"
      @open-data="openDataPanel"
      @open-style="openStylePanel"
      @remove="emit('remove', $event)"
    />
    <section class="designer-shell__workspace">
      <DashboardComponentPalette
        :descriptors="businessDashboardWidgetDescriptors"
        :collapsed="paletteCollapsed"
        @add="emit('add', $event)"
        @toggle="paletteCollapsed = !paletteCollapsed"
      />
      <BusinessDashboardCanvas
        :document="document"
        :selected-widget-id="selectedWidgetId"
        :issue-widget-ids="issueWidgetIds"
        :preview="preview"
        :interaction-mode="interactionMode"
        :theme="isDark ? 'dark' : 'light'"
        :empty-illustration="dashboardEmptyState"
        @select="emit('select', $event || null)"
        @remove="emit('remove', $event)"
        @edit="emit('edit', $event)"
        @duplicate="emit('duplicate', $event)"
        @learn="emit('help')"
        @drop="dropWidget"
        @update-layouts="emit('updateLayouts', $event)"
      />
    </section>
    <el-drawer
      v-model="dataPanelOpen"
      class="dashboard-designer-drawer"
      direction="rtl"
      size="400px"
      :show-close="false"
      :with-header="false"
    >
      <button
        class="designer-shell__drawer-close"
        type="button"
        aria-label="关闭数据配置"
        @click="dataPanelOpen = false"
      >
        <RiCloseLine aria-hidden="true" />
      </button>
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
    </el-drawer>
    <el-drawer
      v-model="stylePanelOpen"
      class="dashboard-designer-drawer"
      direction="rtl"
      size="360px"
      :show-close="false"
      :with-header="false"
    >
      <button
        class="designer-shell__drawer-close"
        type="button"
        aria-label="关闭仪表盘样式"
        @click="stylePanelOpen = false"
      >
        <RiCloseLine aria-hidden="true" />
      </button>
      <DashboardStylePanel
        :desktop="document.settings.desktop"
        @update="emit('updateDesktopSettings', $event)"
      />
    </el-drawer>
    <el-drawer
      v-model="propertiesPanelOpen"
      class="dashboard-designer-drawer"
      direction="rtl"
      size="380px"
      :show-close="false"
      :with-header="false"
    >
      <button
        class="designer-shell__drawer-close"
        type="button"
        aria-label="关闭组件设置"
        @click="propertiesPanelOpen = false"
      >
        <RiCloseLine aria-hidden="true" />
      </button>
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
    </el-drawer>
  </main>
</template>

<style scoped>
/* Vue 的 :global()/:deep() 用于抽屉 Teleport 与响应式子组件覆盖。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.designer-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--el-bg-color-page);
}

.designer-shell__workspace {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

:global(.dashboard-designer-drawer .el-drawer__body) {
  position: relative;
  padding: 0;
}

.designer-shell__drawer-close {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 4;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: var(--el-fill-color-light);
  border: 0;
  border-radius: 4px;
}

.designer-shell__drawer-close:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.designer-shell__drawer-close svg {
  width: 18px;
  height: 18px;
}

.designer-shell__conflict {
  display: flex;
  gap: 18px;
  align-items: center;
  justify-content: space-between;
  padding: 10px 18px;
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
  border-bottom: 1px solid var(--el-color-danger-light-7);
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

@media (width <= 980px) {
  .designer-shell__workspace :deep(.component-palette) {
    flex-basis: 176px;
  }
}
</style>
