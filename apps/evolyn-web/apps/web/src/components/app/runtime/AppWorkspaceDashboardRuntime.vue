<script setup lang="ts">
import type {
  BusinessDashboardDocument,
  BusinessDashboardWidgetRuntime,
} from '@evolyn.do/dashboard';
import type { AppWorkspaceAsset } from '../workspace/appWorkspace.types';
import {
  BusinessDashboardRenderer,
  normalizeBusinessDashboardDocument,
} from '@evolyn.do/dashboard';
import { shallowRef, watch } from 'vue';
import { getDashboardRuntime, queryDashboardWidget } from '~/api/dashboard';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'AppWorkspaceDashboardRuntime' });

const props = defineProps<{
  asset: AppWorkspaceAsset;
}>();

type RuntimeStatus = 'loading' | 'ready' | 'error';

const status = shallowRef<RuntimeStatus>('loading');
const document = shallowRef<BusinessDashboardDocument | null>(null);
const runtimes = shallowRef<Record<string, BusinessDashboardWidgetRuntime>>({});
const errorMessage = shallowRef('仪表盘加载失败，请稍后重试');
const runtimeVersion = shallowRef(0);
const reloadRevision = shallowRef(0);

// 一个加载会话覆盖定义和全部组件查询；切换资产或手动重载时统一取消，
// 防止旧仪表盘的迟到响应覆盖当前画布。
let sessionController: AbortController | null = null;
let sessionId = 0;
const widgetRequestVersions = new Map<string, number>();

watch(
  [() => props.asset.targetCode, reloadRevision],
  async ([dashboardCode], _previous, onCleanup) => {
    sessionController?.abort();
    const controller = new AbortController();
    const currentSession = ++sessionId;
    sessionController = controller;
    onCleanup(() => controller.abort());

    status.value = 'loading';
    document.value = null;
    runtimes.value = {};
    widgetRequestVersions.clear();

    if (!dashboardCode) {
      errorMessage.value = '当前菜单未关联有效仪表盘';
      status.value = 'error';
      return;
    }

    try {
      const bootstrap = await getDashboardRuntime(dashboardCode, controller.signal);
      if (controller.signal.aborted || currentSession !== sessionId) return;

      const normalized = normalizeBusinessDashboardDocument(bootstrap.document);
      if (!normalized.document) {
        throw new Error(normalized.issues[0]?.message ?? '仪表盘配置无效');
      }

      runtimeVersion.value = bootstrap.version;
      document.value = normalized.document;
      runtimes.value = Object.fromEntries(
        normalized.document.widgets.map((widget) => [
          widget.id,
          { status: widget.datasetId ? 'loading' : 'idle' },
        ]),
      );
      status.value = 'ready';

      // 画布先进入可见状态，再并行加载各组件，避免慢查询阻塞整体布局。
      await Promise.all(
        normalized.document.widgets
          .filter((widget) => widget.datasetId)
          .map((widget) => loadWidget(widget.id, 1, currentSession, controller.signal)),
      );
    } catch (error) {
      if (controller.signal.aborted || currentSession !== sessionId) return;
      errorMessage.value = error instanceof Error ? error.message : '仪表盘加载失败，请稍后重试';
      status.value = 'error';
    }
  },
  { immediate: true },
);

async function loadWidget(
  widgetId: string,
  page = 1,
  expectedSession = sessionId,
  signal = sessionController?.signal,
) {
  const dashboardCode = props.asset.targetCode;
  if (!dashboardCode || !signal || signal.aborted || expectedSession !== sessionId) return;
  const requestVersion = (widgetRequestVersions.get(widgetId) ?? 0) + 1;
  widgetRequestVersions.set(widgetId, requestVersion);

  runtimes.value = { ...runtimes.value, [widgetId]: { status: 'loading' } };
  try {
    const result = await queryDashboardWidget(
      dashboardCode,
      widgetId,
      runtimeVersion.value,
      page,
      signal,
    );
    if (
      signal.aborted ||
      expectedSession !== sessionId ||
      widgetRequestVersions.get(widgetId) !== requestVersion
    )
      return;
    runtimes.value = { ...runtimes.value, [widgetId]: { status: 'success', result } };
  } catch (error) {
    if (
      signal.aborted ||
      expectedSession !== sessionId ||
      widgetRequestVersions.get(widgetId) !== requestVersion
    )
      return;
    runtimes.value = {
      ...runtimes.value,
      [widgetId]: {
        status: 'error',
        message: error instanceof Error ? error.message : '数据加载失败，请稍后重试',
      },
    };
  }
}

function retryWidget(widgetId: string): void {
  if (!document.value?.widgets.some((widget) => widget.id === widgetId)) return;
  void loadWidget(widgetId);
}

function reload(): void {
  reloadRevision.value += 1;
}
</script>

<template>
  <main class="app-workspace-dashboard-runtime" :aria-label="`${props.asset.label}仪表盘`">
    <section
      v-if="status === 'loading'"
      v-loading="true"
      class="app-workspace-dashboard-runtime__state"
      aria-label="正在加载仪表盘"
    />

    <el-result
      v-else-if="status === 'error'"
      class="app-workspace-dashboard-runtime__state"
      icon="error"
      title="加载仪表盘失败"
      :sub-title="errorMessage"
    >
      <template #extra>
        <el-button type="primary" @click="reload">
          重新加载
        </el-button>
      </template>
    </el-result>

    <BusinessDashboardRenderer
      v-else-if="document"
      class="app-workspace-dashboard-runtime__surface"
      :document="document"
      :runtimes="runtimes"
      :theme="isDark ? 'dark' : 'light'"
      @retry="retryWidget"
      @page-change="loadWidget"
    />
  </main>
</template>

<style scoped lang="scss">
.app-workspace-dashboard-runtime {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--el-bg-color-page);

  &__state,
  &__surface {
    flex: 1;
    min-width: 0;
    min-height: 0;
  }
}
</style>
