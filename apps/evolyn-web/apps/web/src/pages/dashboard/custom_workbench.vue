<script setup lang="ts">
import { ref } from 'vue';
import WorkbenchEditorShell from '~/components/dashboard/editor/WorkbenchEditorShell.vue';
import WorkbenchEditorToolbar from '~/components/dashboard/editor/WorkbenchEditorToolbar.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';

const device = ref<'desktop' | 'mobile'>('desktop');
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
    width: 100%;
    height: 100%;
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
  --el-border-color-lighter: #ebeef5;
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-9: #e8f1ff;
}

.custom-workbench-page :deep(.grid-stack > .grid-stack-item > .grid-stack-item-content) {
  overflow: hidden !important;
}
</style>
