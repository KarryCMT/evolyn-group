<script setup lang="ts">
import {
  RiArrowRightDoubleFill,
  RiDatabase2Fill,
  RiEditBoxFill,
  RiNotification3Fill,
  RiQuestionFill,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import MessageCenterDrawer from '~/components/dashboard/messageCenter/MessageCenterDrawer.vue';
import UserMenu from '~/components/navigation/UserMenu.vue';
import { useNotificationStore } from '~/stores/notification';
import type { ApplicationWorkspaceMode } from './applicationWorkspace.types';

defineOptions({ name: 'ApplicationWorkspaceHeader' });

const props = defineProps<{
  mode: ApplicationWorkspaceMode;
  sidebarCollapsed: boolean;
}>();

const emit = defineEmits<{
  updateMode: [mode: ApplicationWorkspaceMode];
  toggleSidebar: [];
}>();

const messageCenterVisible = shallowRef(false);
// 未读摘要读 Pinia notification store（与主导航顶栏共读同一事实源）
const notificationStore = useNotificationStore();
const unreadMessageCount = computed(() => notificationStore.unreadTotal);

const modeItems: { mode: ApplicationWorkspaceMode; label: string; icon: typeof RiEditBoxFill }[] = [
  { mode: 'fill', label: '仅添加数据', icon: RiEditBoxFill },
  { mode: 'design', label: '编辑', icon: RiEditBoxFill },
  { mode: 'data', label: '数据管理', icon: RiDatabase2Fill },
];
</script>

<template>
  <header class="application-workspace-header">
    <nav class="application-workspace-header__modes" aria-label="当前资产操作模式">
      <button
        v-if="props.sidebarCollapsed"
        class="application-workspace-header__sidebar-toggle"
        type="button"
        aria-label="展开侧边栏"
        aria-expanded="false"
        title="展开侧边栏"
        @click="emit('toggleSidebar')"
      >
        <RiArrowRightDoubleFill aria-hidden="true" />
      </button>
      <button
        v-for="item in modeItems"
        :key="item.mode"
        class="application-workspace-header__mode"
        :class="{ 'application-workspace-header__mode--active': props.mode === item.mode }"
        type="button"
        :aria-current="props.mode === item.mode ? 'page' : undefined"
        @click="emit('updateMode', item.mode)"
      >
        <component :is="item.icon" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="application-workspace-header__actions">
      <button
        class="application-workspace-header__icon-button"
        type="button"
        aria-label="通知"
        @click="messageCenterVisible = true"
      >
        <RiNotification3Fill />
        <span v-if="unreadMessageCount" class="application-workspace-header__notice-dot" />
      </button>
      <button class="application-workspace-header__icon-button" type="button" aria-label="帮助">
        <RiQuestionFill />
      </button>
      <UserMenu />
    </div>
  </header>
  <MessageCenterDrawer v-model="messageCenterVisible" />
  />
</template>

<style scoped lang="scss">
.application-workspace-header {
  display: flex;
  height: 64px;
  min-height: 64px;
  padding: 0 var(--el-space-xl);
  align-items: center;
  justify-content: space-between;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__modes,
  &__actions {
    display: flex;
    align-items: center;
  }

  &__modes {
    height: 100%;
    gap: var(--el-space-xs);
  }

  &__mode,
  &__icon-button,
  &__sidebar-toggle {
    display: inline-flex;
    border: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border-radius: var(--el-border-radius-base);
  }

  &__mode {
    height: 36px;
    padding: 0 var(--el-space-lg);
    gap: var(--el-space-sm);
    font-size: var(--el-font-size-medium);

    svg {
      width: 19px;
      height: 19px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &--active {
      color: var(--el-text-color-primary);
      font-weight: 650;
    }
  }

  &__sidebar-toggle {
    width: 36px;
    height: 36px;
    padding: 0;
    font-size: var(--el-font-size-extra-large);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__actions {
    gap: var(--el-space-md);
  }

  &__icon-button {
    position: relative;
    width: 32px;
    height: 32px;
    padding: 0;
    font-size: var(--el-font-size-extra-large);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__notice-dot {
    position: absolute;
    top: 5px;
    right: 5px;
    width: 7px;
    height: 7px;
    background: var(--el-color-danger);
    border: 1px solid var(--el-bg-color);
    border-radius: var(--el-border-radius-half);
  }
}
</style>
