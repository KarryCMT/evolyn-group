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

  :deep(.tenant-route-tabs) {
    min-height: 48px;
    flex: 0 0 48px;
    padding: 0 0 0 8px;
    border-bottom: 0;
    align-items: flex-start;
    gap: 4px;
    background: var(--el-fill-color-light);
  }

  :deep(.tenant-route-tabs__item) {
    min-height: 48px;
    padding: 0 24px;
    border-bottom: 0;
    border-radius: 16px 16px 0 0;
    font-size: 16px;
  }

  :deep(.tenant-route-tabs__item:hover) {
    background: rgb(255 255 255 / 64%);
  }

  :deep(.tenant-route-tabs__item--active) {
    background: var(--el-bg-color);
  }

  &__content {
    min-height: 0;
    flex: 1;
  }
}
</style>
