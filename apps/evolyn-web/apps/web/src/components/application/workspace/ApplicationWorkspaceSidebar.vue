<script setup lang="ts">
import {
  RiAddFill,
  RiArrowLeftDoubleFill,
  RiArrowLeftSLine,
  RiBarChartBoxFill,
  RiBookOpenFill,
  RiFileAddFill,
  RiFolderAddFill,
  RiGitBranchFill,
  RiSearch2Line,
  RiSettings3Fill,
} from '@remixicon/vue';
import { computed } from 'vue';
import type { ApplicationIcon } from '~/types';
import ApplicationWorkspaceAssetItem from './ApplicationWorkspaceAssetItem.vue';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
} from './applicationWorkspace.types';
import { applicationPersonalNavigation } from './applicationWorkspacePreview';

defineOptions({ name: 'ApplicationWorkspaceSidebar' });

const props = defineProps<{
  applicationName: string;
  applicationIcon: ApplicationIcon;
  assets: ApplicationWorkspaceAsset[];
  activeAssetCode: string;
  collapsed: boolean;
  /** 菜单数据源状态：loading/error 由页面层拦截，这里只渲染加载与空态 */
  menuStatus: 'loading' | 'ready';
}>();

const emit = defineEmits<{
  back: [];
  createAsset: [
    payload: { parent?: ApplicationWorkspaceAsset; type: ApplicationWorkspaceCreateAssetType },
  ];
  assetGuide: [];
  selectAsset: [asset: ApplicationWorkspaceAsset];
  assetAction: [
    payload: { asset: ApplicationWorkspaceAsset; action: ApplicationWorkspaceAssetAction },
  ];
  openManagement: [];
  toggleSidebar: [];
}>();

/** 递归收集树内叶子资产（菜单首期树形简单，遍历成本可忽略） */
function flattenLeaves(assets: ApplicationWorkspaceAsset[]): ApplicationWorkspaceAsset[] {
  return assets.flatMap((asset) =>
    asset.children?.length ? flattenLeaves(asset.children) : [asset],
  );
}

const hasAssets = computed(() => flattenLeaves(props.assets).length > 0);

function handleCreateAsset(command: string | number | object) {
  if (typeof command !== 'string') return;
  emit('createAsset', { type: command as ApplicationWorkspaceCreateAssetType });
}
</script>

<template>
  <aside
    class="application-workspace-sidebar"
    :class="{ 'application-workspace-sidebar--collapsed': props.collapsed }"
    aria-label="应用导航"
  >
    <div class="application-workspace-sidebar__content" :aria-hidden="props.collapsed">
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
        <el-dropdown
          placement="right-start"
          trigger="click"
          popper-class="application-workspace-root-create-actions"
          @command="handleCreateAsset"
        >
          <button
            class="application-workspace-sidebar__create"
            type="button"
            aria-label="新建应用资产"
          >
            <RiAddFill aria-hidden="true" />
          </button>
          <template #dropdown>
            <el-dropdown-menu class="application-workspace-root-create-actions__menu">
              <el-dropdown-item command="workflow-form">
                <RiGitBranchFill aria-hidden="true" />
                <span>新建流程表单</span>
              </el-dropdown-item>
              <el-dropdown-item command="form">
                <RiFileAddFill aria-hidden="true" />
                <span>新建表单</span>
              </el-dropdown-item>
              <el-dropdown-item command="dashboard">
                <RiBarChartBoxFill aria-hidden="true" />
                <span>新建仪表盘</span>
              </el-dropdown-item>
              <el-dropdown-item command="folder" divided>
                <RiFolderAddFill aria-hidden="true" />
                <span>新建分组</span>
              </el-dropdown-item>
              <el-dropdown-item
                class="application-workspace-root-create-actions__guide"
                divided
                @click="emit('assetGuide')"
              >
                <RiBookOpenFill aria-hidden="true" />
                <span>了解表单和仪表盘</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <!-- 资产树独立滚动，头部操作与底部应用后台始终保持可见。 -->
      <el-scrollbar class="application-workspace-sidebar__asset-scrollbar">
        <nav class="application-workspace-sidebar__asset-nav" aria-label="应用资产">
          <p v-if="props.menuStatus === 'loading'" class="application-workspace-sidebar__menu-tip">
            菜单加载中…
          </p>
          <p v-else-if="!hasAssets" class="application-workspace-sidebar__menu-tip">
            暂无应用资产，点击 + 创建第一个资产
          </p>
          <template v-else>
            <ApplicationWorkspaceAssetItem
              v-for="asset in props.assets"
              :key="asset.code"
              :asset="asset"
              :active-asset-code="props.activeAssetCode"
              :depth="0"
              @create-asset="emit('createAsset', $event)"
              @select-asset="emit('selectAsset', $event)"
              @asset-action="emit('assetAction', $event)"
            />
          </template>
        </nav>
      </el-scrollbar>

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
    </div>
  </aside>
</template>

<style scoped lang="scss">
.application-workspace-sidebar {
  box-sizing: border-box;
  display: flex;
  width: 280px;
  min-width: 280px;
  padding: var(--el-space-lg) var(--el-space-lg) var(--el-space-xl);
  overflow: hidden;
  flex-direction: column;
  color: var(--el-color-white);
  background: var(--el-color-primary);
  transition:
    width 0.24s ease,
    min-width 0.24s ease,
    padding 0.24s ease;

  &__content {
    display: flex;
    min-height: 0;
    min-width: 252px;
    flex: 1;
    flex-direction: column;
    opacity: 1;
    visibility: visible;
    transform: translateX(0);
    transition:
      opacity 0.16s ease 0.24s,
      visibility 0s linear 0s,
      transform 0.16s ease 0.24s;
  }

  &__header,
  &__nav-item,
  &__asset-tools,
  &__search,
  &__management {
    display: flex;
    min-width: 0;
    align-items: center;
  }

  &__header {
    min-height: 44px;
    gap: var(--el-space-md);
  }

  &__back,
  &__create,
  &__collapse,
  &__nav-item,
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
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-extra-large);

    &:hover {
      background: rgb(255 255 255 / 14%);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-white);
      outline-offset: 2px;
    }
  }

  /* Element Plus 下拉触发器会重置继承色，显式保证顶层创建入口为白色。 */
  &__create {
    color: var(--el-color-white);
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
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-large);
  }

  &__app-name {
    overflow: hidden;
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    line-height: 24px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__collapse {
    margin-left: auto;
    font-size: var(--el-font-size-extra-large);
  }

  &__personal-nav,
  &__asset-nav {
    display: flex;
    flex-direction: column;
  }

  &__personal-nav {
    margin: var(--el-space-2xl) 0 var(--el-space-2xl);
    gap: var(--el-space-xs);
  }

  &__nav-item,
  &__management {
    min-height: 42px;
    padding: 0 var(--el-space-md);
    gap: var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-base);
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
    padding-top: var(--el-space-lg);
    border-top: 1px solid color-mix(in srgb, var(--el-color-white) 30%, var(--el-color-transparent));
    gap: var(--el-space-md);
  }

  &__search {
    min-width: 0;
    flex: 1;
    gap: var(--el-space-md);
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
    gap: var(--el-space-xs);
  }

  &__asset-scrollbar {
    min-height: 0;
    flex: 1;
    margin-top: var(--el-space-lg);

    :deep(.el-scrollbar__wrap) {
      overflow-x: hidden;
    }
  }

  &__menu-tip {
    margin: var(--el-space-sm) var(--el-space-md);
    color: rgb(255 255 255 / 72%);
    font-size: var(--el-font-size-small);
    line-height: 20px;
  }

  &__footer {
    padding-top: var(--el-space-lg);
    flex: 0 0 auto;
    margin-top: auto;
    border-top: 1px solid color-mix(in srgb, var(--el-color-white) 30%, var(--el-color-transparent));
  }

  &__management {
    width: 100%;
    justify-content: flex-start;
  }

  &--collapsed {
    width: 10px;
    min-width: 10px;
    padding: 0;

    .application-workspace-sidebar__content {
      pointer-events: none;
      opacity: 0;
      visibility: hidden;
      transform: translateX(-8px);
      transition:
        opacity 0.12s ease,
        visibility 0s linear 0.12s,
        transform 0.12s ease;
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

<!-- 顶层创建菜单传送至 body，使用唯一类名限定样式。 -->
<style lang="scss">
.application-workspace-root-create-actions.el-popper {
  min-width: 232px;
  border-color: var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}

.application-workspace-root-create-actions__menu.el-dropdown-menu {
  padding: var(--el-space-sm);
  --el-dropdown-menuItem-hover-fill: var(--el-fill-color-light);
  --el-dropdown-menuItem-hover-color: var(--el-text-color-primary);
}

.application-workspace-root-create-actions__menu .el-dropdown-menu__item {
  height: 42px;
  gap: var(--el-space-md);
  padding: 0 var(--el-space-lg);
  border-radius: var(--el-border-radius-medium);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-base);

  svg {
    width: 19px;
    height: 19px;
    color: var(--el-text-color-secondary);
  }
}

.application-workspace-root-create-actions__guide.el-dropdown-menu__item {
  margin-top: var(--el-space-xs);
}
</style>
