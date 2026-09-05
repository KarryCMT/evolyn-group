<script setup lang="ts">
import type { TenantRouteTab } from '~/types/tenant';
import TenantRouteTabs from '~/components/tenant/TenantRouteTabs.vue';

defineOptions({ name: 'TenantAdministratorsPage' });

const tabs: TenantRouteTab[] = [
  { label: '系统', path: '/tenant/administrators/system' },
  { label: '灵衍云', path: '/tenant/administrators/apps' },
];
</script>

<template>
  <section class="tenant-administrators-page">
    <TenantRouteTabs :tabs="tabs" />
    <RouterView />
  </section>
</template>

<style scoped lang="scss">
.tenant-administrators-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color);

  // 管理员页采用沉浸式页签：页签与左右工作区共用一张内容画布。
  :deep(.tenant-route-tabs) {
    min-height: 72px;
    flex: 0 0 72px;
    padding: 0 var(--el-space-4xl);
    gap: var(--el-space-xl);
    background: transparent;
  }

  :deep(.tenant-route-tabs__item) {
    min-height: 72px;
    padding: 0 var(--el-space-lg);
    border-bottom-width: 3px;
    font-size: var(--el-font-size-medium);
  }

  :deep(.tenant-route-tabs__item:hover) {
    background: transparent;
  }

  :deep(.tenant-route-tabs__item--active) {
    background: transparent;
  }

  :deep(.administrator-workspace) {
    flex: 1;
  }
}
</style>
