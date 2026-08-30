<script setup lang="ts">
import {
  RiArrowDownSFill,
  RiArrowRightSFill,
  RiBarChartBoxFill,
  RiFileList3Fill,
  RiFolderFill,
  RiGitBranchFill,
  RiSearchFill,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type { PermissionAsset, PermissionAssetType } from './permission.types';
defineOptions({ name: 'PermissionAssetList' });

const props = defineProps<{
  assets: PermissionAsset[];
  keyword: string;
  selectedAssetId: string;
}>();

const emit = defineEmits<{
  batchSelect: [];
  updateKeyword: [keyword: string];
  select: [assetId: string];
}>();

// 资产列表按层级平铺展示（group 为可展开/收起的容器行，不可选中），图标按资产类型区分。
const assetTypeIcons: Record<Exclude<PermissionAssetType, 'group'>, typeof RiFileList3Fill> = {
  'workflow-form': RiGitBranchFill,
  form: RiFileList3Fill,
  dashboard: RiBarChartBoxFill,
};

const GROUP_INDENT_STEP = 20;

interface VisibleAssetRow {
  asset: PermissionAsset;
  depth: number;
  /** 分组行是否处于展开态；无 children 的分组不参与展开。 */
  expanded?: boolean;
}

const collapsedGroupIds = shallowRef<ReadonlySet<string>>(new Set());

function toggleGroup(groupId: string) {
  const next = new Set(collapsedGroupIds.value);
  if (next.has(groupId)) {
    next.delete(groupId);
  } else {
    next.add(groupId);
  }
  collapsedGroupIds.value = next;
}

const normalizedKeyword = computed(() => props.keyword.trim().toLocaleLowerCase());

// 名称溢出省略后，仅对真正发生截断的行在悬停时弹出完整文案提示。
const hoveredRowId = shallowRef<string>();
const hoveredRowOverflowing = shallowRef(false);

function handleRowHover(row: VisibleAssetRow, event: MouseEvent) {
  hoveredRowId.value = row.asset.id;
  const label = (event.currentTarget as HTMLElement).querySelector('[data-asset-name]');
  hoveredRowOverflowing.value = !!label && label.scrollWidth > label.clientWidth;
}

/** 关键字过滤保留层级：名称未命中的节点仅当其子树内存在命中时保留。 */
function filterAssetTree(list: PermissionAsset[], keyword: string): PermissionAsset[] {
  const result: PermissionAsset[] = [];
  for (const asset of list) {
    if (asset.name.toLocaleLowerCase().includes(keyword)) {
      result.push(asset);
      continue;
    }
    if (asset.children?.length) {
      const children = filterAssetTree(asset.children, keyword);
      if (children.length) result.push({ ...asset, children });
    }
  }
  return result;
}

const visibleRows = computed<VisibleAssetRow[]>(() => {
  const source = normalizedKeyword.value
    ? filterAssetTree(props.assets, normalizedKeyword.value)
    : props.assets;

  const rows: VisibleAssetRow[] = [];
  // 搜索时忽略收起状态全量展开，保证命中结果不因分组收起而被隐藏。
  const searching = Boolean(normalizedKeyword.value);
  const walk = (list: PermissionAsset[], depth: number) => {
    for (const asset of list) {
      const expanded = Boolean(asset.children?.length) && (searching || !collapsedGroupIds.value.has(asset.id));
      rows.push({ asset, depth, expanded });
      if (expanded) walk(asset.children!, depth + 1);
    }
  };
  walk(source, 0);
  return rows;
});

function rowIndentStyle(depth: number) {
  return { paddingLeft: `calc(var(--el-space-md) + ${depth * GROUP_INDENT_STEP}px)` };
}

function updateKeyword(event: Event) {
  emit('updateKeyword', (event.target as HTMLInputElement).value);
}
</script>

<template>
  <aside class="permission-asset-list" aria-label="选择表单或仪表盘">
    <header class="permission-asset-list__header">
      <div>
        <p class="permission-asset-list__step">01</p>
        <h2 class="permission-asset-list__title">选择资产</h2>
      </div>
      <button class="permission-asset-list__batch" type="button" @click="emit('batchSelect')">
        批量选择
      </button>
    </header>

    <label class="permission-asset-list__search">
      <RiSearchFill aria-hidden="true" />
      <input
        :value="props.keyword"
        type="search"
        placeholder="搜索表单或仪表盘"
        aria-label="搜索表单或仪表盘"
        @input="updateKeyword"
      />
    </label>

    <!-- 资产列表独立滚动，搜索框与批量操作始终保留在可见区域。 -->
    <el-scrollbar class="permission-asset-list__scrollbar">
      <div v-if="visibleRows.length" class="permission-asset-list__list">
        <template v-for="row in visibleRows" :key="row.asset.id">
          <!-- 分组行可点击展开/收起，仅作层级容器展示，不参与选中。 -->
          <button
            v-if="row.asset.type === 'group'"
            class="permission-asset-list__group-row"
            :class="{ 'permission-asset-list__group-row--leaf': !row.asset.children?.length }"
            :style="rowIndentStyle(row.depth)"
            type="button"
            :aria-expanded="row.expanded ? 'true' : 'false'"
            @click="toggleGroup(row.asset.id)"
            @mouseenter="handleRowHover(row, $event)"
          >
            <RiArrowDownSFill
              v-if="row.expanded"
              class="permission-asset-list__chevron"
              aria-hidden="true"
            />
            <RiArrowRightSFill v-else class="permission-asset-list__chevron" aria-hidden="true" />
            <RiFolderFill aria-hidden="true" />
            <el-tooltip
              :content="row.asset.name"
              placement="right"
              :show-after="300"
              :disabled="!(hoveredRowId === row.asset.id && hoveredRowOverflowing)"
            >
              <span data-asset-name class="permission-asset-list__name">{{
                row.asset.name
              }}</span>
            </el-tooltip>
          </button>
          <button
            v-else
            class="permission-asset-list__asset"
            :class="{ 'permission-asset-list__asset--active': props.selectedAssetId === row.asset.id }"
            :style="rowIndentStyle(row.depth)"
            type="button"
            @click="emit('select', row.asset.id)"
            @mouseenter="handleRowHover(row, $event)"
          >
            <component :is="assetTypeIcons[row.asset.type]" aria-hidden="true" />
            <el-tooltip
              :content="row.asset.name"
              placement="right"
              :show-after="300"
              :disabled="!(hoveredRowId === row.asset.id && hoveredRowOverflowing)"
            >
              <span data-asset-name class="permission-asset-list__name">{{
                row.asset.name
              }}</span>
            </el-tooltip>
          </button>
        </template>
      </div>
      <div v-else class="permission-asset-list__empty">未找到匹配的应用资产</div>
    </el-scrollbar>
  </aside>
</template>

<style scoped lang="scss">
.permission-asset-list {
  box-sizing: border-box;
  display: flex;
  min-height: 0;
  width: 292px;
  min-width: 292px;
  padding: var(--el-space-2xl) var(--el-space-xl);
  overflow: hidden;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);

  &__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--el-space-lg);
  }

  &__step,
  &__title {
    margin: 0;
  }

  &__step {
    margin-bottom: var(--el-space-xs);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-extra-small);
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 16px;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 26px;
  }

  &__batch {
    min-height: 30px;
    padding: 0 var(--el-space-sm);
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font-size: var(--el-font-size-small);

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__search {
    display: flex;
    height: 38px;
    margin-top: var(--el-space-2xl);
    padding: 0 var(--el-space-lg);
    align-items: center;
    gap: var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-light);

    &:focus-within {
      outline: 2px solid var(--el-color-primary-light-5);
      background: var(--el-fill-color-blank);
    }

    svg {
      width: 17px;
      height: 17px;
      flex: 0 0 auto;
    }

    input {
      width: 100%;
      min-width: 0;
      border: 0;
      outline: 0;
      color: var(--el-text-color-primary);
      background: transparent;
      font: inherit;
      font-size: var(--el-font-size-small);

      &::placeholder {
        color: var(--el-text-color-placeholder);
      }
    }
  }

  &__scrollbar {
    height: 0;
    margin-top: var(--el-space-2xl);
    min-height: 0;
    flex: 1;
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-xs);
  }

  &__group-row {
    display: flex;
    min-height: 34px;
    padding-right: var(--el-space-md);
    align-items: center;
    gap: var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-small);
    font-weight: 600;
    text-align: left;

    svg {
      width: 18px;
      height: 18px;
      flex: 0 0 auto;
    }

    &:hover {
      color: var(--el-text-color-primary);
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    // 空分组没有子级可展开，呈现为普通展示行。
    &--leaf {
      cursor: default;

      &:hover {
        color: var(--el-text-color-secondary);
        background: transparent;
      }
    }
  }

  &__chevron {
    width: 18px !important;
    height: 18px !important;
    color: var(--el-text-color-placeholder);
  }

  // 名称单行截断省略；溢出时由 el-tooltip 悬浮展示完整文案。
  &__name {
    min-width: 0;
    overflow: hidden;
    flex: 1;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__asset {
    display: flex;
    min-height: 38px;
    padding: 0 var(--el-space-md);
    align-items: center;
    gap: var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-base);
    text-align: left;

    svg {
      width: 18px;
      height: 18px;
      flex: 0 0 auto;
      color: var(--el-text-color-secondary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);

      svg {
        color: var(--el-color-primary);
      }
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      color: var(--el-color-primary);
      font-weight: 600;
      background: var(--el-color-primary-light-9);

      svg {
        color: var(--el-color-primary);
      }
    }
  }

  &__empty {
    display: grid;
    min-height: 180px;
    place-items: center;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}

@media (max-width: 900px) {
  .permission-asset-list {
    width: 240px;
    min-width: 240px;
  }
}
</style>
