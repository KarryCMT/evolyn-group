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
    <RouterView />
  </section>
</template>

<style scoped lang="scss">
.tenant-member-fields-page {
  min-height: 100%;
}
</style>
