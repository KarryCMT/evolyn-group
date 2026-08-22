<script setup lang="ts">
import { RiArrowLeftDoubleFill } from '@remixicon/vue';
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
    <TopNavigation title="管理后台" back-to="/dashboard" :show-default-navigation="false" />

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
  background: #f3f3f8;

  /* 管理后台遵循浅色界面，避免受到成员端深色偏好的影响。 */
  --el-bg-color: #ffffff;
  --el-bg-color-overlay: #ffffff;
  --el-fill-color: #f4f6fa;
  --el-fill-color-light: #f7f8fc;
  --el-fill-color-lighter: #fafbfc;
  --el-fill-color-blank: #ffffff;
  --el-text-color-primary: #202938;
  --el-text-color-regular: #515968;
  --el-text-color-secondary: #8a94a6;
  --el-border-color-lighter: #ebedf2;

  &__main {
    position: relative;
    display: flex;
    min-height: 0;
    flex: 1;
  }

  &__sidebar-shell {
    min-width: 0;
    flex: 0 0 236px;
    overflow: hidden;
    transition: flex-basis 0.2s ease;
  }

  &__collapse-button {
    position: absolute;
    z-index: 1;
    /* 贴近侧栏顶部，避开内容区的页签栏。 */
    top: 4px;
    left: 204px;
    display: inline-flex;
    width: 28px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: 0 8px 8px 0;
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
    margin: 0 8px 8px 0;
    overflow: auto;
    border-radius: 10px;
    background: var(--el-bg-color);
  }

  &__main--sidebar-collapsed {
    .tenant-management-layout__sidebar-shell {
      flex-basis: 64px;
    }

    .tenant-management-layout__collapse-button {
      left: 18px;

      svg {
        transform: rotate(180deg);
      }
    }
  }
}

@media (max-width: 720px) {
  .tenant-management-layout {
    &__main {
      overflow-x: auto;
    }
  }
}
</style>
