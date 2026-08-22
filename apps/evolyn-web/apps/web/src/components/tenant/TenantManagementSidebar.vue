<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { tenantNavigationGroups } from './tenantNavigation';

defineOptions({ name: 'TenantManagementSidebar' });

const route = useRoute();

const props = defineProps<{
  collapsed: boolean;
}>();

/** 父级路由下的 Tab 仍归属同一个侧栏功能项。 */
const activePath = computed(() => route.path);

function isActive(path: string, nestedPath?: string) {
  return (
    activePath.value === path ||
    Boolean(nestedPath && activePath.value.startsWith(`${nestedPath}/`))
  );
}
</script>

<template>
  <aside
    class="tenant-management-sidebar"
    :class="{ 'tenant-management-sidebar--collapsed': props.collapsed }"
    aria-label="管理后台导航"
  >
    <nav class="tenant-management-sidebar__nav">
      <section
        v-for="group in tenantNavigationGroups"
        :key="group.label"
        class="tenant-management-sidebar__group"
      >
        <h2 class="tenant-management-sidebar__group-title">{{ group.label }}</h2>
        <RouterLink
          v-for="item in group.items"
          :key="item.key"
          class="tenant-management-sidebar__item"
          :class="{
            'tenant-management-sidebar__item--active': isActive(item.path, item.activePath),
          }"
          :to="item.path"
        >
          <component :is="item.icon" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </section>
    </nav>
  </aside>
</template>

<style scoped lang="scss">
.tenant-management-sidebar {
  /* 固定宽度包含内边距，避免选中卡片被外层容器裁切并贴住内容区。 */
  box-sizing: border-box;
  width: 236px;
  flex: 0 0 236px;
  padding: 12px;
  overflow-y: auto;

  &__nav,
  &__group {
    display: flex;
    flex-direction: column;
  }

  &__group {
    gap: 2px;

    &:not(:first-child) {
      /* 分组只保留识别间距，避免菜单总高度在常规桌面视口产生滚动。 */
      margin-top: 10px;
    }
  }

  &__group-title {
    margin: 0;
    padding: 0 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 500;
    line-height: 20px;
  }

  &__item {
    display: flex;
    min-height: 36px;
    align-items: center;
    gap: 8px;
    padding: 0 10px;
    border-radius: 6px;
    color: var(--el-text-color-regular);
    font-size: 14px;
    text-decoration: none;
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    svg {
      width: 16px;
      height: 16px;
      color: #98a2b3;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      box-shadow: 0 2px 8px rgb(15 23 42 / 7%);

      svg {
        color: var(--el-color-primary);
      }
    }
  }

  &--collapsed {
    width: 64px;
    flex-basis: 64px;
    /* 收起后控制按钮位于图标栏顶部，菜单需避开这一独立控制区。 */
    padding: 52px 8px 12px;

    .tenant-management-sidebar__group {
      gap: 4px;

      &:not(:first-child) {
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid var(--el-border-color-lighter);
      }
    }

    .tenant-management-sidebar__group-title,
    .tenant-management-sidebar__item span {
      display: none;
    }

    .tenant-management-sidebar__item {
      width: 48px;
      min-height: 42px;
      justify-content: center;
      padding: 0;
      border-radius: 8px;
    }
  }
}

@media (max-width: 900px) {
  .tenant-management-sidebar:not(.tenant-management-sidebar--collapsed) {
    width: 200px;
    flex-basis: 200px;
  }
}
</style>
