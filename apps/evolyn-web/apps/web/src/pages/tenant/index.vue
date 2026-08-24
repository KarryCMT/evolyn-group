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
  background: linear-gradient(142deg, rgb(255 255 255 / 0%) 40%, rgb(225 232 248 / 64%)), #f3f4f9;

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
  --el-text-color-placeholder: #a8b0bd;
  --el-border-color: #dfe3eb;
  --el-border-color-light: #e5e8ee;
  --el-border-color-lighter: #ebedf2;
  --el-border-color-extra-light: #f2f4f7;
  --el-fill-color-extra-light: #fafbfc;

  &__main {
    position: relative;
    display: flex;
    min-height: 0;
    flex: 1;
  }

  &__sidebar-shell {
    min-width: 0;
    flex: 0 0 320px;
    overflow: hidden;
    transition: flex-basis 0.2s ease;
  }

  &__collapse-button {
    position: absolute;
    z-index: 1;
    /* 贴近侧栏顶部，避开内容区的页签栏。 */
    top: 12px;
    left: 276px;
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
    margin: 0 16px 16px 0;
    overflow: hidden;
    border-radius: 16px;
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
  border-radius: 8px;
  align-items: center;
  justify-content: center;
  color: #596271;
  background: transparent;
  cursor: pointer;

  svg {
    width: 26px;
    height: 26px;
  }

  &:hover {
    background: rgb(54 65 82 / 8%);
  }
}

.tenant-management-layout__brand-icon {
  width: 31px;
  height: 31px;
  color: var(--el-color-primary);
}

/* 管理后台的顶栏比成员工作台更舒展，和下方宽菜单形成同一套管理界面比例。 */
:deep(.top-navigation) {
  height: 72px;
  min-height: 72px;
  padding: 0 28px;
  background: transparent;
}

:deep(.top-navigation__brand) {
  gap: 10px;
}

:deep(.top-navigation__title) {
  font-size: 21px;
}

@media (max-width: 720px) {
  .tenant-management-layout {
    &__main {
      overflow-x: auto;
    }
  }
}
</style>
