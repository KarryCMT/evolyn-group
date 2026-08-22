<script setup lang="ts">
import { shallowRef } from 'vue';
import WorkbenchEditorShell from '~/components/dashboard/editor/WorkbenchEditorShell.vue';
import WorkbenchEditorToolbar from '~/components/dashboard/editor/WorkbenchEditorToolbar.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';

const device = shallowRef<'desktop' | 'mobile'>('desktop');
function notify() {}
</script>

<template>
  <div class="custom-workbench-page">
    <TopNavigation title="自定义工作台" back-to="/dashboard" />
    <div class="custom-dashboard-setting">
      <WorkbenchEditorToolbar
        v-model:device="device"
        @page-style="notify"
        @preview="notify"
        @save="notify"
      />
      <WorkbenchEditorShell :device="device" />
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
  background: #f3f3f8;
  .custom-dashboard-setting {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    margin: 0 8px 8px;
    overflow: hidden;
    background: var(--el-bg-color);
    border-radius: 10px;
  }
}

/* 设计器遵循工作台配置稿的浅色画布，避免继承成员端的深色偏好。 */
.custom-workbench-page {
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
  /* 与成员端工作台保持一致：卡片边界轻、圆角和投影可感知。 */
  --el-border-radius-base: 10px;
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
