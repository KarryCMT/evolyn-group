<script setup lang="ts">
import type { DashboardWorkspaceTab } from '~/components/dashboard/workspace/dashboardWorkspace.types';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getDashboard } from '~/api/dashboard';
import { appWorkspaceRoute } from '~/components/app/workspace/appWorkspaceNavigation';
import DashboardAutoRefreshCard from '~/components/dashboard/extensions/DashboardAutoRefreshCard.vue';
import DashboardExtensionHeader from '~/components/dashboard/extensions/DashboardExtensionHeader.vue';
import DashboardScheduledReminderCard from '~/components/dashboard/extensions/DashboardScheduledReminderCard.vue';

defineOptions({ name: 'DashboardExtensionsPage' });

const route = useRoute();
const router = useRouter();
const dashboardCode = computed(() => String(route.params.dashboardCode ?? ''));
const appCode = computed(() => String(route.params.appCode ?? ''));
const dashboardName = shallowRef('仪表盘');

let requestVersion = 0;
watch(
  dashboardCode,
  async (code) => {
    if (!code) return;
    const version = ++requestVersion;
    try {
      const detail = await getDashboard(code);
      if (version !== requestVersion || dashboardCode.value !== code) return;
      dashboardName.value = detail.name;
      document.title = `${detail.name} - 扩展功能`;
    } catch {
      if (version === requestVersion) ElMessage.error('仪表盘信息加载失败，已显示默认名称');
    }
  },
  { immediate: true },
);

function returnToApp(): void {
  void router.push(appWorkspaceRoute(appCode.value, dashboardCode.value));
}

function navigate(tab: DashboardWorkspaceTab): void {
  if (tab === 'extensions') return;
  if (tab === 'publish') {
    ElMessage.info('仪表盘发布能力正在建设中');
    return;
  }
  void router.push({
    name: 'dashboard-design',
    params: { appCode: appCode.value, dashboardCode: dashboardCode.value },
  });
}

function showHelp(): void {
  ElMessage.info('仪表盘帮助中心正在建设中');
}
</script>

<template>
  <main class="dashboard-extensions-page">
    <DashboardExtensionHeader
      :name="dashboardName"
      @back="returnToApp"
      @help="showHelp"
      @navigate="navigate"
    />
    <div class="dashboard-extensions-page__scroll">
      <div class="dashboard-extensions-page__content">
        <DashboardAutoRefreshCard />
        <DashboardScheduledReminderCard />
      </div>
    </div>
  </main>
</template>

<style scoped>
.dashboard-extensions-page {
  display: flex;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-text-color-primary);
  /* 跟随 Element Plus 页面级主题变量，避免暗黑模式下出现亮色画布。 */
  background: var(--el-bg-color-page);
}

.dashboard-extensions-page__scroll {
  min-height: 0;
  flex: 1;
  overflow: auto;
}

.dashboard-extensions-page__content {
  width: min(1120px, calc(100% - 48px));
  margin: 28px auto 64px;
  display: grid;
  gap: 20px;
}

@media (width <= 640px) {
  .dashboard-extensions-page__content {
    width: calc(100% - 24px);
    margin-top: 12px;
    gap: 12px;
  }
}
</style>
