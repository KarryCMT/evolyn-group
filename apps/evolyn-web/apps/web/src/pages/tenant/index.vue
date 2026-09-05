<script setup lang="ts">
import { RiArrowLeftDoubleFill, RiArrowLeftLine, RiSettings3Fill } from '@remixicon/vue';
import { shallowRef } from 'vue';
import TenantManagementSidebar from '~/components/tenant/TenantManagementSidebar.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';

defineOptions({ name: 'TenantManagementLayout' });

const sidebarCollapsed = shallowRef(false);

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}
</script>

<template>
  <div class="tenant-management-layout">
    <!-- 管理后台沿用统一顶栏，仅关闭成员工作台的快捷导航。 -->
    <TopNavigation title="管理后台" back-to="/dashboard" :show-default-navigation="false">
      <template #leading="{ goBack }">
        <button
          class="tenant-management-layout__back-button"
          type="button"
          aria-label="返回工作台"
          @click="goBack"
        >
          <RiArrowLeftLine />
        </button>
        <RiSettings3Fill class="tenant-management-layout__brand-icon" aria-hidden="true" />
      </template>
    </TopNavigation>

    <main
      class="tenant-management-layout__main"
      :class="{ 'tenant-management-layout__main--sidebar-collapsed': sidebarCollapsed }"
    >
      <div class="tenant-management-layout__sidebar-shell">
        <TenantManagementSidebar :collapsed="sidebarCollapsed" />
      </div>
      <el-tooltip :content="sidebarCollapsed ? '展开' : '收起'" placement="right">
        <button
          class="tenant-management-layout__collapse-button"
          type="button"
          :aria-expanded="!sidebarCollapsed"
          :aria-label="sidebarCollapsed ? '展开管理后台导航' : '收起管理后台导航'"
          @click="toggleSidebar"
        >
          <RiArrowLeftDoubleFill />
        </button>
      </el-tooltip>
      <section class="tenant-management-layout__content">
        <RouterView />
      </section>
    </main>
  </div>
</template>

<style scoped lang="scss">
.tenant-management-layout {
  display: flex;
  height: 100vh;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  /* 继承全局 Element Plus 主题变量，使管理后台随 html.dark 切换。 */
  background: var(--el-bg-color-page);

  &__main {
    position: relative;
    display: flex;
    min-height: 0;
    flex: 1;
  }

  &__sidebar-shell {
    min-width: 0;
    flex: 0 0 234px;
    overflow: hidden;
    transition: flex-basis 0.2s ease;
  }

  &__collapse-button {
    position: absolute;
    z-index: 1;
    /* 贴近侧栏顶部，避开内容区的页签栏。 */
    top: 12px;
    left: 190px;
    display: inline-flex;
    width: 28px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: 0 var(--el-border-radius-medium) var(--el-border-radius-medium) 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      left 0.2s ease;

    svg {
      width: 20px;
      height: 20px;
      transition: transform 0.2s ease;
    }

    &:hover {
      color: var(--el-text-color-regular);
      background: transparent;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__content {
    min-width: 0;
    flex: 1;
    margin: 0 var(--el-space-xl) var(--el-space-xl) 0;
    overflow: hidden;
    border-radius: var(--el-border-radius-large);
    background: var(--el-bg-color);
  }

  &__main--sidebar-collapsed {
    .tenant-management-layout__sidebar-shell {
      flex-basis: 72px;
    }

    .tenant-management-layout__collapse-button {
      left: 22px;

      svg {
        transform: rotate(180deg);
      }
    }
  }
}

.tenant-management-layout__back-button {
  display: inline-flex;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  background: transparent;
  cursor: pointer;

  &:hover {
    background: var(--el-fill-color);
  }
}

.tenant-management-layout__brand-icon {
  width: 31px;
  height: 31px;
  color: var(--el-color-primary);
}

@media (max-width: 720px) {
  .tenant-management-layout {
    &__main {
      overflow-x: auto;
    }
  }
}
</style>
