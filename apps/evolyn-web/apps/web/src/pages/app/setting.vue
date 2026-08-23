<script setup lang="ts">
import { RiArrowLeftDoubleFill } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import { useRoute } from 'vue-router';
import AppSettingSidebar from '~/components/app/AppSettingSidebar.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';

defineOptions({ name: 'ApplicationSettingLayout' });

const route = useRoute();
const sidebarCollapsed = shallowRef(false);
const appCode = computed(() => String(route.params.appCode ?? ''));
const applicationHomePath = computed(() => `/app/${appCode.value}`);

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}
</script>

<template>
  <div class="application-setting-layout">
    <TopNavigation
      title="应用后台"
      :back-to="applicationHomePath"
      :show-default-navigation="false"
    />

    <main
      class="application-setting-layout__main"
      :class="{ 'application-setting-layout__main--sidebar-collapsed': sidebarCollapsed }"
    >
      <div class="application-setting-layout__sidebar-shell">
        <AppSettingSidebar :collapsed="sidebarCollapsed" />
      </div>
      <el-tooltip :content="sidebarCollapsed ? '展开' : '收起'" placement="right">
        <button
          class="application-setting-layout__collapse-button"
          type="button"
          :aria-expanded="!sidebarCollapsed"
          :aria-label="sidebarCollapsed ? '展开应用后台导航' : '收起应用后台导航'"
          @click="toggleSidebar"
        >
          <RiArrowLeftDoubleFill />
        </button>
      </el-tooltip>
      <section class="application-setting-layout__content">
        <RouterView />
      </section>
    </main>
  </div>
</template>

<style scoped lang="scss">
.application-setting-layout {
  display: flex;
  height: 100vh;
  min-width: 0;
  overflow: hidden;
  flex-direction: column;
  background: #f3f3f8;

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
    cursor: pointer;
    background: transparent;
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
      background: var(--el-fill-color-light);
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
    .application-setting-layout__sidebar-shell {
      flex-basis: 64px;
    }

    .application-setting-layout__collapse-button {
      left: 18px;

      svg {
        transform: rotate(180deg);
      }
    }
  }
}

@media (max-width: 720px) {
  .application-setting-layout__main {
    overflow-x: auto;
  }
}
</style>
