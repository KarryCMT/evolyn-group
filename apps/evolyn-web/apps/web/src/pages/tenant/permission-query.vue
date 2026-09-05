<script setup lang="ts">
import PermissionQueryPanel from '~/components/tenant/permissionQuery/PermissionQueryPanel.vue';

defineOptions({ name: 'TenantPermissionQueryPage' });
</script>

<template>
  <section class="tenant-permission-query-page">
    <PermissionQueryPanel />
  </section>
</template>

<style scoped lang="scss">
.tenant-permission-query-page {
  // 权限查询与其他二级页共用连续的沉浸式内容画布。
  height: 100%;
  min-height: 0;
  background: var(--el-bg-color);
}
</style>
