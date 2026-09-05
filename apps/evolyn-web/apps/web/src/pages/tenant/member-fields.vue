<script setup lang="ts">
import type { TenantRouteTab } from '~/types/tenant';
import TenantRouteTabs from '~/components/tenant/TenantRouteTabs.vue';

defineOptions({ name: 'TenantMemberFieldsPage' });

const tabs: TenantRouteTab[] = [
  { label: '字段设置', path: '/tenant/member-fields/fields' },
  { label: '卡片展示', path: '/tenant/member-fields/cards' },
];
</script>

<template>
  <section class="tenant-member-fields-page">
    <TenantRouteTabs :tabs="tabs" />
    <section class="tenant-member-fields-page__content">
      <RouterView />
    </section>
  </section>
</template>

<style scoped lang="scss">
.tenant-member-fields-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: var(--el-bg-color);

  // 与管理员页共用沉浸式页签：内容画布连续，状态由文字和底部指示线表达。
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

  &__content {
    min-height: 0;
    flex: 1;
  }
}
</style>
