<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import DashboardDesignerShell from '~/components/dashboard/designer/DashboardDesignerShell.vue';
import { prepareDashboardPreview, useDashboardDesigner } from '~/composables/useDashboardDesigner';
import { useUnsavedChangesGuard } from '~/composables/useUnsavedChangesGuard';

defineOptions({ name: 'DashboardDesignPage' });

const route = useRoute();
const router = useRouter();
const dashboardCode = computed(() => String(route.params.dashboardCode ?? ''));
const appCode = computed(() => String(route.params.appCode ?? ''));
const designer = useDashboardDesigner();

watch(
  dashboardCode,
  async (code) => {
    if (!code) return;
    const loaded = await designer.load(code);
    if (loaded && designer.detail.value)
      document.title = `${designer.detail.value.name} - 仪表盘设计`;
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
  void router.push({ name: 'App', params: { appCode: appCode.value } });
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
    :revision="designer.draftRevision.value"
    :document="designer.editingDocument.value"
    :selected-widget="designer.selectedWidget.value"
    :selected-widget-id="designer.selectedWidgetId.value"
    :issue-widget-ids="designer.issueWidgetIds.value"
    :issues="designer.issues.value"
    :focused-issue-path="designer.focusedIssuePath.value"
    :dirty="designer.isDirty.value"
    :save-status="designer.saveStatus.value"
    :conflict-message="designer.errorMessage.value"
    @back="returnToApp"
    @save="saveDraft"
    @preview="openPreview"
    @reload-conflict="reloadConflict"
    @add="designer.addWidget"
    @select="designer.selectWidget"
    @remove="designer.removeWidget"
    @update-widget="designer.updateWidget"
    @update-layouts="designer.replaceLayouts"
    @focus-issue="designer.focusIssue"
  />
</template>

<style scoped>
.design-page__state {
  min-height: 100vh;
  background: #eef2f5;
}
</style>
