<script setup lang="ts">
import EnterpriseSettingsPanel from '~/components/tenant/enterpriseSettings/EnterpriseSettingsPanel.vue';

defineOptions({ name: 'TenantEnterpriseSettingsPage' });
</script>

<template>
  <section class="tenant-enterprise-settings-page">
    <EnterpriseSettingsPanel />
  </section>
</template>

<style scoped lang="scss">
.tenant-enterprise-settings-page {
  height: 100%;
  min-height: 0;
  background: var(--el-bg-color);
}
</style>
