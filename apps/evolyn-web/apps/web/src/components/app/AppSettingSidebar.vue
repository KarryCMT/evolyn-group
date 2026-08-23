<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { appSettingNavigationGroups } from './appSettingNavigation';

defineOptions({ name: 'AppSettingSidebar' });

const route = useRoute();

const props = defineProps<{
  collapsed: boolean;
}>();

/** appCode 是所有应用设置子路由的必填参数，由当前父路由统一提供。 */
const appCode = computed(() => String(route.params.appCode ?? ''));

function isActive(routeName: string) {
  return route.name === routeName;
}
</script>

<template>
  <aside
    class="app-setting-sidebar"
    :class="{ 'app-setting-sidebar--collapsed': props.collapsed }"
    aria-label="应用后台导航"
  >
    <!-- 侧栏菜单项较多时仅菜单区域滚动，避免使用浏览器原生滚动条。 -->
    <el-scrollbar class="app-setting-sidebar__scrollbar">
      <nav class="app-setting-sidebar__nav">
        <section
          v-for="group in appSettingNavigationGroups"
          :key="group.label"
          class="app-setting-sidebar__group"
        >
          <h2 class="app-setting-sidebar__group-title">{{ group.label }}</h2>
          <RouterLink
            v-for="item in group.items"
            :key="item.key"
            class="app-setting-sidebar__item"
            :class="{ 'app-setting-sidebar__item--active': isActive(item.routeName) }"
            :to="{ name: item.routeName, params: { appCode } }"
          >
            <component :is="item.icon" aria-hidden="true" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </section>
      </nav>
    </el-scrollbar>
  </aside>
</template>

<style scoped lang="scss">
.app-setting-sidebar {
  box-sizing: border-box;
  display: flex;
  min-height: 0;
  width: 236px;
  flex: 0 0 236px;
  padding: 12px;
  overflow: hidden;
  flex-direction: column;

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__nav,
  &__group {
    display: flex;
    flex-direction: column;
  }

  &__group {
    gap: 2px;

    &:not(:first-child) {
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
    cursor: pointer;
    font-size: 14px;
    text-decoration: none;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    svg {
      width: 16px;
      height: 16px;
      color: var(--el-text-color-secondary);
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
    padding: 52px 8px 12px;

    .app-setting-sidebar__group {
      gap: 4px;

      &:not(:first-child) {
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid var(--el-border-color-lighter);
      }
    }

    .app-setting-sidebar__group-title,
    .app-setting-sidebar__item span {
      display: none;
    }

    .app-setting-sidebar__item {
      width: 48px;
      min-height: 42px;
      justify-content: center;
      padding: 0;
      border-radius: 8px;
    }
  }
}

@media (max-width: 900px) {
  .app-setting-sidebar:not(.app-setting-sidebar--collapsed) {
    width: 200px;
    flex-basis: 200px;
  }
}
</style>
