<script setup lang="ts">
import {
  RiAddFill,
  RiArrowLeftDoubleFill,
  RiArrowLeftSLine,
  RiSearch2Line,
  RiSettings3Fill,
} from '@remixicon/vue';
import type { ApplicationIcon } from '~/types';
import type { ApplicationWorkspaceAsset } from './applicationWorkspace.types';
import { applicationPersonalNavigation } from './applicationWorkspacePreview';

defineOptions({ name: 'ApplicationWorkspaceSidebar' });

const props = defineProps<{
  applicationName: string;
  applicationIcon: ApplicationIcon;
  assets: ApplicationWorkspaceAsset[];
  activeAssetCode: string;
  collapsed: boolean;
}>();

const emit = defineEmits<{
  back: [];
  createAsset: [];
  selectAsset: [asset: ApplicationWorkspaceAsset];
  openManagement: [];
  toggleSidebar: [];
}>();
</script>

<template>
  <aside
    class="application-workspace-sidebar"
    :class="{ 'application-workspace-sidebar--collapsed': props.collapsed }"
    aria-label="应用导航"
  >
    <header class="application-workspace-sidebar__header">
      <button
        class="application-workspace-sidebar__back"
        type="button"
        aria-label="返回工作台"
        @click="emit('back')"
      >
        <RiArrowLeftSLine />
      </button>
      <span class="application-workspace-sidebar__app-icon" aria-hidden="true">
        <component :is="props.applicationIcon" />
      </span>
      <strong class="application-workspace-sidebar__app-name">{{ props.applicationName }}</strong>
      <button
        v-if="!props.collapsed"
        class="application-workspace-sidebar__collapse"
        type="button"
        aria-label="收起侧边栏"
        aria-expanded="true"
        title="收起侧边栏"
        @click="emit('toggleSidebar')"
      >
        <RiArrowLeftDoubleFill aria-hidden="true" />
      </button>
    </header>

    <nav class="application-workspace-sidebar__personal-nav" aria-label="个人应用入口">
      <button
        v-for="item in applicationPersonalNavigation"
        :key="item.code"
        class="application-workspace-sidebar__nav-item"
        type="button"
        :aria-label="item.label"
      >
        <component :is="item.icon" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="application-workspace-sidebar__asset-tools">
      <label class="application-workspace-sidebar__search">
        <RiSearch2Line aria-hidden="true" />
        <input placeholder="输入名称来搜索" aria-label="搜索应用资产" />
      </label>
      <button
        class="application-workspace-sidebar__create"
        type="button"
        aria-label="新建应用资产"
        @click="emit('createAsset')"
      >
        <RiAddFill />
      </button>
    </div>

    <nav class="application-workspace-sidebar__asset-nav" aria-label="应用资产">
      <button
        v-for="asset in props.assets"
        :key="asset.code"
        class="application-workspace-sidebar__asset-item"
        :class="{
          'application-workspace-sidebar__asset-item--active': asset.code === props.activeAssetCode,
        }"
        type="button"
        :aria-label="asset.label"
        @click="emit('selectAsset', asset)"
      >
        <component :is="asset.icon" aria-hidden="true" />
        <span>{{ asset.label }}</span>
      </button>
    </nav>

    <footer class="application-workspace-sidebar__footer">
      <button
        class="application-workspace-sidebar__management"
        type="button"
        aria-label="应用后台"
        @click="emit('openManagement')"
      >
        <RiSettings3Fill aria-hidden="true" />
        <span>应用后台</span>
      </button>
    </footer>
  </aside>
</template>

<style scoped lang="scss">
.application-workspace-sidebar {
  box-sizing: border-box;
  display: flex;
  width: 280px;
  min-width: 280px;
  padding: 14px 14px 18px;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-color-white);
  background: var(--el-color-primary);
  transition:
    width 0.2s ease,
    min-width 0.2s ease,
    padding 0.2s ease;

  &__header,
  &__nav-item,
  &__asset-tools,
  &__search,
  &__asset-item,
  &__management {
    display: flex;
    min-width: 0;
    align-items: center;
  }

  &__header {
    min-height: 44px;
    gap: 10px;
  }

  &__back,
  &__create,
  &__collapse,
  &__nav-item,
  &__asset-item,
  &__management {
    border: 0;
    color: inherit;
    cursor: pointer;
    background: transparent;
  }

  &__back,
  &__create,
  &__collapse {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    border-radius: 7px;
    font-size: 23px;

    &:hover {
      background: rgb(255 255 255 / 14%);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-white);
      outline-offset: 2px;
    }
  }

  &__app-icon {
    display: inline-flex;
    width: 30px;
    height: 30px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary);
    background: var(--el-color-white);
    border-radius: 7px;
    font-size: 19px;
  }

  &__app-name {
    overflow: hidden;
    font-size: 17px;
    font-weight: 650;
    line-height: 24px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__collapse {
    margin-left: auto;
    font-size: 20px;
  }

  &__personal-nav,
  &__asset-nav {
    display: flex;
    flex-direction: column;
  }

  &__personal-nav {
    margin: 20px 0 22px;
    gap: 3px;
  }

  &__nav-item,
  &__asset-item,
  &__management {
    min-height: 42px;
    padding: 0 10px;
    gap: 10px;
    border-radius: 7px;
    font-size: 15px;
    text-align: left;

    svg {
      width: 19px;
      height: 19px;
      flex: 0 0 auto;
    }

    &:hover {
      background: rgb(255 255 255 / 14%);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-white);
      outline-offset: -2px;
    }
  }

  &__asset-tools {
    padding-top: 14px;
    border-top: 1px solid rgb(255 255 255 / 30%);
    gap: 8px;
  }

  &__search {
    min-width: 0;
    flex: 1;
    gap: 8px;
    color: rgb(255 255 255 / 78%);

    input {
      width: 100%;
      min-width: 0;
      padding: 0;
      color: inherit;
      outline: 0;
      background: transparent;
      border: 0;
      font: inherit;

      &::placeholder {
        color: rgb(255 255 255 / 68%);
      }
    }
  }

  &__asset-nav {
    margin-top: 12px;
    gap: 3px;
  }

  &__asset-item {
    &--active {
      background: rgb(0 0 0 / 14%);
      font-weight: 650;
    }
  }

  &__footer {
    padding-top: 14px;
    margin-top: auto;
    border-top: 1px solid rgb(255 255 255 / 30%);
  }

  &__management {
    width: 100%;
    justify-content: flex-start;
  }

  &--collapsed {
    width: 10px;
    min-width: 10px;
    padding: 0;

    .application-workspace-sidebar__header,
    .application-workspace-sidebar__personal-nav,
    .application-workspace-sidebar__asset-tools,
    .application-workspace-sidebar__asset-nav,
    .application-workspace-sidebar__footer {
      display: none;
    }
  }
}

@media (max-width: 900px) {
  .application-workspace-sidebar {
    width: 280px;
    min-width: 280px;

    &--collapsed {
      width: 10px;
      min-width: 10px;
    }
  }
}
</style>
