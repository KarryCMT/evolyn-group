<script setup lang="ts">
import {
  RiAddFill,
  RiArrowDownSFill,
  RiArrowRightSFill,
  RiBarChartBoxFill,
  RiDeleteBin6Fill,
  RiDragMove2Fill,
  RiEditFill,
  RiFileAddFill,
  RiFolderAddFill,
  RiGitBranchFill,
  RiMoreFill,
  RiStarFill,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
} from './applicationWorkspace.types';

defineOptions({ name: 'ApplicationWorkspaceAssetItem' });

const props = defineProps<{
  asset: ApplicationWorkspaceAsset;
  activeAssetCode: string;
  /** 菜单首期最多三层，缩进由树深度决定。 */
  depth: number;
}>();

const emit = defineEmits<{
  createAsset: [
    payload: { parent?: ApplicationWorkspaceAsset; type: ApplicationWorkspaceCreateAssetType },
  ];
  selectAsset: [asset: ApplicationWorkspaceAsset];
  assetAction: [
    payload: { asset: ApplicationWorkspaceAsset; action: ApplicationWorkspaceAssetAction },
  ];
}>();

const expanded = shallowRef(true);
const isFolder = computed(() => props.asset.type === 'folder');
// 分组仅承担展开、管理与容纳子节点职责，不能成为内容区的选中资产。
const isActive = computed(() => !isFolder.value && props.asset.code === props.activeAssetCode);
const hasActions = computed(
  () =>
    props.asset.capabilities.manage ||
    props.asset.capabilities.move ||
    props.asset.capabilities.favorite ||
    props.asset.capabilities.delete,
);
const canCreateChild = computed(() => isFolder.value && props.asset.capabilities.manage);

const actionItems = computed(() => {
  const items: Array<{
    action: ApplicationWorkspaceAssetAction;
    label: string;
    icon: typeof RiEditFill;
    danger?: boolean;
  }> = [];

  if (props.asset.capabilities.manage) {
    items.push({
      action: 'edit',
      label: isFolder.value ? '修改名称和图标' : '编辑',
      icon: RiEditFill,
    });
  }
  if (props.asset.capabilities.move) {
    items.push({ action: 'move', label: '移动', icon: RiDragMove2Fill });
  }
  if (props.asset.capabilities.favorite) {
    items.push({ action: 'favorite', label: '收藏', icon: RiStarFill });
  }
  if (props.asset.capabilities.delete) {
    items.push({ action: 'delete', label: '删除', icon: RiDeleteBin6Fill, danger: true });
  }
  return items;
});

function activateAsset() {
  if (isFolder.value) {
    expanded.value = !expanded.value;
    return;
  }
  emit('selectAsset', props.asset);
}

function handleCreateAsset(command: string | number | object) {
  if (typeof command !== 'string') return;
  emit('createAsset', {
    parent: props.asset,
    type: command as ApplicationWorkspaceCreateAssetType,
  });
}

function handleAction(command: string | number | object) {
  if (typeof command !== 'string') return;
  emit('assetAction', {
    asset: props.asset,
    action: command as ApplicationWorkspaceAssetAction,
  });
}
</script>

<template>
  <div
    class="application-workspace-asset-item"
    :class="{
      'application-workspace-asset-item--active': isActive,
      'application-workspace-asset-item--folder': isFolder,
      [`application-workspace-asset-item--depth-${props.depth}`]: true,
    }"
  >
    <div class="application-workspace-asset-item__row">
      <button
        class="application-workspace-asset-item__main"
        type="button"
        :aria-label="props.asset.label"
        :aria-expanded="isFolder ? expanded : undefined"
        @click="activateAsset"
      >
        <span
          v-if="isFolder"
          class="application-workspace-asset-item__icon-slot"
          aria-hidden="true"
        >
          <component :is="props.asset.icon" class="application-workspace-asset-item__folder-icon" />
          <component
            :is="expanded ? RiArrowDownSFill : RiArrowRightSFill"
            class="application-workspace-asset-item__group-toggle"
          />
        </span>
        <component v-else :is="props.asset.icon" aria-hidden="true" />
        <span>{{ props.asset.label }}</span>
      </button>

      <div v-if="hasActions || canCreateChild" class="application-workspace-asset-item__actions">
        <el-dropdown
          v-if="hasActions"
          placement="right-start"
          trigger="click"
          popper-class="application-workspace-asset-actions"
          @command="handleAction"
        >
          <button
            class="application-workspace-asset-item__action"
            type="button"
            :aria-label="`${props.asset.label}更多操作`"
          >
            <RiMoreFill aria-hidden="true" />
          </button>
          <template #dropdown>
            <el-dropdown-menu class="application-workspace-asset-actions__menu">
              <el-dropdown-item
                v-for="item in actionItems"
                :key="item.action"
                :command="item.action"
                :class="{ 'application-workspace-asset-actions__item--danger': item.danger }"
              >
                <component :is="item.icon" aria-hidden="true" />
                <span>{{ item.label }}</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-dropdown
          v-if="canCreateChild"
          placement="right-start"
          trigger="click"
          popper-class="application-workspace-asset-create-actions"
          @command="handleCreateAsset"
        >
          <button
            class="application-workspace-asset-item__action"
            type="button"
            :aria-label="`在${props.asset.label}中创建应用资产`"
          >
            <RiAddFill aria-hidden="true" />
          </button>
          <template #dropdown>
            <el-dropdown-menu class="application-workspace-asset-create-actions__menu">
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
                <span>新建子分组</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <div
      v-if="isFolder && expanded && props.asset.children?.length"
      class="application-workspace-asset-item__children"
    >
      <ApplicationWorkspaceAssetItem
        v-for="child in props.asset.children"
        :key="child.code"
        :asset="child"
        :active-asset-code="props.activeAssetCode"
        :depth="props.depth + 1"
        @create-asset="emit('createAsset', $event)"
        @select-asset="emit('selectAsset', $event)"
        @asset-action="emit('assetAction', $event)"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.application-workspace-asset-item {
  min-width: 0;

  &__row {
    display: flex;
    min-width: 0;
    min-height: 42px;
    align-items: center;
    border-radius: 7px;

    /* 仅鼠标移入显示操作态；点击分组后的 DOM 焦点不能留下类似选中态的背景。 */
    &:hover {
      background: rgb(0 0 0 / 12%);

      .application-workspace-asset-item__action {
        opacity: 1;
      }

      .application-workspace-asset-item__folder-icon {
        opacity: 0;
      }

      .application-workspace-asset-item__group-toggle {
        opacity: 1;
      }
    }
  }

  &__main,
  &__action {
    border: 0;
    color: inherit;
    cursor: pointer;
    background: transparent;
  }

  &__main {
    display: flex;
    min-width: 0;
    min-height: 42px;
    padding: 0 4px 0 10px;
    flex: 1;
    align-items: center;
    gap: 10px;
    font-size: 15px;
    text-align: left;

    svg {
      width: 19px;
      height: 19px;
      flex: 0 0 auto;
    }

    span {
      overflow: hidden;
      min-width: 0;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-white);
      outline-offset: -2px;
    }
  }

  &__icon-slot {
    position: relative;
    display: inline-flex;
    width: 19px;
    height: 19px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
  }

  &__folder-icon,
  &__group-toggle {
    position: absolute;
    transition: opacity 0.16s ease;
  }

  &__folder-icon {
    opacity: 1;
  }

  &__group-toggle {
    opacity: 0;
  }

  &__actions {
    display: inline-flex;
    margin-right: 6px;
    flex: 0 0 auto;
    align-items: center;
    gap: 2px;
  }

  &__action {
    display: inline-flex;
    width: 30px;
    height: 30px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    color: var(--el-color-white);
    font-size: 19px;
    opacity: 0;
    transition:
      opacity 0.16s ease,
      background-color 0.16s ease;

    &:hover {
      background: rgb(255 255 255 / 18%);
    }

    &:focus-visible {
      opacity: 1;
      outline: 2px solid var(--el-color-white);
      outline-offset: -2px;
    }
  }

  &--active > &__row {
    background: rgb(0 0 0 / 14%);
    font-weight: 650;
  }

  &--depth-1 > &__row &__main {
    padding-left: 22px;
  }

  &--depth-2 > &__row &__main {
    padding-left: 34px;
  }
}
</style>

<!-- 下拉浮窗传送至 body，使用唯一类名限定样式，避免影响其他下拉菜单。 -->
<style lang="scss">
.application-workspace-asset-actions.el-popper {
  min-width: 168px;
  border-color: var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}

.application-workspace-asset-actions__menu.el-dropdown-menu {
  padding: 6px;
  --el-dropdown-menuItem-hover-fill: var(--el-fill-color-light);
  --el-dropdown-menuItem-hover-color: var(--el-text-color-primary);
}

.application-workspace-asset-actions__menu .el-dropdown-menu__item {
  height: 38px;
  gap: 9px;
  padding: 0 10px;
  border-radius: 6px;
  color: var(--el-text-color-primary);

  svg {
    width: 17px;
    height: 17px;
    color: var(--el-text-color-secondary);
  }
}

.application-workspace-asset-actions__item--danger.el-dropdown-menu__item {
  color: var(--el-color-danger);
  --el-dropdown-menuItem-hover-fill: var(--el-color-danger-light-9);
  --el-dropdown-menuItem-hover-color: var(--el-color-danger);

  svg {
    color: inherit;
  }
}

.application-workspace-asset-create-actions.el-popper {
  min-width: 208px;
  border-color: var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}

.application-workspace-asset-create-actions__menu.el-dropdown-menu {
  padding: 6px;
  --el-dropdown-menuItem-hover-fill: var(--el-fill-color-light);
  --el-dropdown-menuItem-hover-color: var(--el-text-color-primary);
}

.application-workspace-asset-create-actions__menu .el-dropdown-menu__item {
  height: 42px;
  gap: 10px;
  padding: 0 12px;
  border-radius: 6px;
  color: var(--el-text-color-primary);
  font-size: 15px;

  svg {
    width: 19px;
    height: 19px;
    color: var(--el-text-color-secondary);
  }
}
</style>
