<script setup lang="ts">
import type { TenantRouteTab } from '~/types/tenant';

defineOptions({ name: 'TenantRouteTabs' });

defineProps<{
  tabs: TenantRouteTab[];
}>();
</script>

<template>
  <nav class="tenant-route-tabs" aria-label="页面导航">
    <RouterLink
      v-for="tab in tabs"
      :key="tab.path"
      v-slot="{ href, isExactActive, navigate }"
      custom
      :to="tab.path"
    >
      <a
        class="tenant-route-tabs__item"
        :class="{ 'tenant-route-tabs__item--active': isExactActive }"
        :href="href"
        @click="navigate"
      >
        {{ tab.label }}
      </a>
    </RouterLink>
  </nav>
</template>

<style scoped lang="scss">
.tenant-route-tabs {
  display: flex;
  min-height: 52px;
  align-items: flex-end;
  padding: 0 22px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  gap: 8px;
  background: var(--el-bg-color);

  &__item {
    display: inline-flex;
    min-height: 52px;
    align-items: center;
    padding: 0 16px;
    border-bottom: 2px solid transparent;
    color: var(--el-text-color-regular);
    font-size: 14px;
    text-decoration: none;
    cursor: pointer;
    transition:
      color 0.18s ease,
      border-color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      border-bottom-color: var(--el-color-primary);
      color: var(--el-color-primary);
      font-weight: 600;
    }
  }
}
</style>
