<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { onMounted, onUnmounted, shallowRef, watch } from 'vue';
import { onBeforeRouteLeave, useRouter } from 'vue-router';
import { useDashboardPersistence } from '@evolyn.do/dashboard';
import { ApiError } from '@evolyn.do/utils';
import WorkbenchEditorShell from '~/components/dashboard/editor/WorkbenchEditorShell.vue';
import WorkbenchEditorToolbar from '~/components/dashboard/editor/WorkbenchEditorToolbar.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import { useAdminScope } from '~/composables/useAdminScope';
import { dashboardWorkspaceAdapter } from '~/composables/useDashboardWorkspace';
import { isDashboardWidgetType } from '~/types/dashboard';

const router = useRouter();
// 工作台是企业级配置（000078）：仅企业管理员可进设计页；服务端
// workbench:update 权限仍是最终边界，此处只做入口与直访的前端拦截
const { isSystemAdmin } = useAdminScope();

const device = shallowRef<'desktop' | 'mobile'>('desktop');
const { document, isDirty, isLoading, isSaving, issues, load, save } = useDashboardPersistence({
  initialDocument: createDefaultWorkbenchSchema(),
  adapter: dashboardWorkspaceAdapter,
  isWidgetType: isDashboardWidgetType,
});

// 非管理员自查返回后直接送回工作台；null（自查中）保持加载占位
watch(
  isSystemAdmin,
  (value) => {
    if (value === false) {
      ElMessage.warning('工作台设置仅企业管理员可访问。');
      void router.replace('/dashboard');
    }
  },
  { immediate: true },
);

onMounted(async () => {
  try {
    await load();
  } catch {
    // 接口不可达等加载失败：回退默认布局继续编辑，保存时会再次尝试
    ElMessage.error('工作台配置加载失败，已展示默认布局。');
  }
  if (issues.value.length) ElMessage.warning(issues.value[0].message);
  window.addEventListener('beforeunload', confirmBrowserLeave);
});

onUnmounted(() => {
  window.removeEventListener('beforeunload', confirmBrowserLeave);
});

/** 保存只提交 schema 文档，成员/租户/请求细节由 Web 侧适配器承担。 */
async function saveWorkspace() {
  try {
    await save();
    ElMessage.success('工作台已保存');
  } catch (error) {
    // 优先透出服务端稳定文案（如 409 乐观锁冲突的刷新重试提示），
    // 本地结构问题回落校验 issue 提示
    if (error instanceof ApiError) {
      ElMessage.error(error.message);
      return;
    }
    ElMessage.error(issues.value[0]?.message ?? '工作台保存失败，请稍后重试。');
  }
}

function confirmBrowserLeave(event: BeforeUnloadEvent) {
  if (!isDirty.value) return;

  event.preventDefault();
  event.returnValue = '';
}

onBeforeRouteLeave(async () => {
  if (!isDirty.value) return true;

  try {
    await ElMessageBox.confirm('当前修改尚未保存，确认离开后将丢失本次修改。', '未保存的修改', {
      confirmButtonText: '放弃修改并离开',
      cancelButtonText: '继续编辑',
      type: 'warning',
    });
    return true;
  } catch {
    return false;
  }
});
</script>

<template>
  <div class="custom-workbench-page">
    <TopNavigation title="自定义工作台" back-to="/dashboard" />
    <div class="custom-dashboard-setting">
      <!-- 自查未返回前保持加载占位；非管理员由守卫送回工作台 -->
      <div v-if="isSystemAdmin !== true" class="custom-dashboard-setting__loading">
        正在校验访问权限…
      </div>
      <template v-else>
        <WorkbenchEditorToolbar
          v-model:device="device"
          :is-dirty="isDirty"
          :is-saving="isSaving"
          @save="saveWorkspace"
        />
        <div v-if="isLoading" class="custom-dashboard-setting__loading">正在加载工作台配置…</div>
        <WorkbenchEditorShell v-else v-model="document" :device="device" />
      </template>
    </div>
  </div>
</template>

<style scoped lang="scss">
.custom-workbench-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;

  /* 页面整体背景：与工作台页面统一的设计浅灰底 */
  background: var(--el-bg-color-page);
  .custom-dashboard-setting {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    margin: 0 var(--el-space-md) var(--el-space-md);
    overflow: hidden;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-large);

    &__loading {
      display: flex;
      flex: 1;
      align-items: center;
      justify-content: center;
      color: var(--el-text-color-secondary);
      font-size: var(--el-font-size-small);
    }
  }
}

/* 浅色模式保留设计器专属的卡片规格；暗黑模式直接继承全局主题变量。 */
.custom-workbench-page {
  /* 与成员端工作台保持一致：卡片边界轻、圆角和投影可感知。 */
  --el-border-radius-base: 10px;
}

:global(html:not(.dark) .custom-workbench-page) {
  --el-bg-color: #ffffff;
  --el-bg-color-overlay: #ffffff;
  --el-fill-color: #f0f2f5;
  --el-fill-color-light: #f5f7fa;
  --el-fill-color-lighter: #fafafa;
  --el-fill-color-blank: #ffffff;
  --el-text-color-primary: #1f2937;
  --el-text-color-regular: #4b5563;
  --el-text-color-secondary: #909399;
  --el-border-color: #dcdfe6;
  --el-border-color-light: #e4e7ed;
  --el-border-color-lighter: rgba(31, 35, 41, 0.06);
  --el-box-shadow-lighter: 0 0 2px 0 rgba(19, 29, 46, 0.02), 0 1px 4px 0 rgba(19, 29, 46, 0.06);
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-9: #e8f1ff;
}

.custom-workbench-page :deep(.evolyn-grid .grid-stack-item-content) {
  /* 设计器预览使用和成员端相同的投影策略，防止 GridStack 裁剪阴影。 */
  overflow: visible !important;
}
</style>
