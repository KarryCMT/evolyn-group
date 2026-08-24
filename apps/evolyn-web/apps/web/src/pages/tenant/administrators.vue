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
  height: 100%;
  min-height: 0;
}
</style>
