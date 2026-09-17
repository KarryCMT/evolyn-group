<script setup lang="ts">
import type { AppMenu, AppMenuNode, AppMenuType } from '~/types';
import {
  RiArrowDownSFill,
  RiArrowRightSFill,
  RiCheckFill,
  RiCloseFill,
  RiSearchFill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { getAppMenuByCode, listApps } from '~/api/apps';
import { resolveMenuIcon } from '~/components/app/menuIcon';
import { useMenuFavorites } from '~/composables/useMenuFavorites';

defineOptions({ name: 'FavoritePickerDialog' });

/** 目录树节点：应用行与分组行仅承载导航（menuId 为空），资产叶子行可勾选。 */
interface CatalogNode {
  key: string;
  /** 可收藏资产节点的 menuId；应用/分组行为 null。 */
  menuId: string | null;
  appCode: string;
  label: string;
  iconKey: string | null;
  type: 'app' | AppMenuType;
  children?: CatalogNode[];
}

/** 渲染行：树按展开态拍平，缩进由 depth 决定。 */
interface CatalogRow extends CatalogNode {
  depth: number;
  expandable: boolean;
  expanded: boolean;
  loading: boolean;
}

const visible = defineModel<boolean>({ default: false });

const emit = defineEmits<{
  /** 差量应用完成（无论全部或部分成功）；列表状态由共享 composable 维护。 */
  applied: [];
}>();

const { items, favorite, unfavorite } = useMenuFavorites();

const searchText = shallowRef('');
const apps = shallowRef<CatalogNode[]>([]);
const appsStatus = shallowRef<'loading' | 'ready' | 'error'>('loading');
/** 应用行展开后的菜单子树（appCode → children），加载一次后复用。 */
const childrenByApp = shallowRef<Record<string, CatalogNode[]>>({});
const loadingApps = shallowRef<ReadonlySet<string>>(new Set());
const expandedKeys = shallowRef<ReadonlySet<string>>(new Set());
/** 草稿勾选：menuId → appCode（收藏写入需要归属应用编码）。 */
const draftChecked = shallowRef<Map<string, string>>(new Map());
const applying = shallowRef(false);

// 每次打开均以当前收藏为基准创建草稿并加载目录，取消不会产生任何写入。
watch(
  () => visible.value,
  (isVisible) => {
    if (!isVisible) return;
    searchText.value = '';
    draftChecked.value = new Map(items.value.map((item) => [item.node.menuId, item.app.code]));
    if (appsStatus.value !== 'ready') {
      void loadApps();
    }
  },
);

async function loadApps() {
  appsStatus.value = 'loading';
  try {
    // 目录只列可用应用；游标翻页拉全（应用数量受配额限制，量级有限）。
    const collected: CatalogNode[] = [];
    let cursor = '';
    do {
      const page = await listApps({ status: 'active', limit: 100, cursor: cursor || undefined });
      for (const app of page.items) {
        if (!app.capabilities.view) continue;
        collected.push({
          key: `app:${app.code}`,
          menuId: null,
          appCode: app.code,
          label: app.name,
          iconKey: app.icon?.type === 'remix' ? app.icon.name : 'bookmark',
          type: 'app',
        });
      }
      cursor = page.nextCursor;
    } while (cursor);
    apps.value = collected;
    appsStatus.value = 'ready';
  } catch (error) {
    console.warn('[favorite-picker] load apps failed', error);
    appsStatus.value = 'error';
  }
}

/** 应用菜单 → 可收藏目录子树：仅保留 capabilities.view 节点，分组无可见
 * 可收藏后代时整枝裁剪；收藏资格以 capabilities.favorite 投影为准（P1 与
 * 服务端裁决同源，前端不二次推断）。 */
function buildCatalogTree(menu: AppMenu, appCode: string): CatalogNode[] {
  const childrenByParent = new Map<string | null, AppMenuNode[]>();
  for (const node of Object.values(menu.nodeMap)) {
    if (!node.capabilities.view) continue;
    const siblings = childrenByParent.get(node.parentMenuId) ?? [];
    siblings.push(node);
    childrenByParent.set(node.parentMenuId, siblings);
  }
  for (const siblings of childrenByParent.values()) {
    siblings.sort((a, b) =>
      a.sortOrder === b.sortOrder ? a.menuId.localeCompare(b.menuId) : a.sortOrder - b.sortOrder,
    );
  }
  const build = (parent: string | null): CatalogNode[] => {
    const rows: CatalogNode[] = [];
    for (const node of childrenByParent.get(parent) ?? []) {
      if (node.type === 'group') {
        const children = build(node.menuId);
        if (children.length > 0) {
          rows.push({
            key: `menu:${node.menuId}`,
            menuId: null,
            appCode,
            label: node.name,
            iconKey: node.icon,
            type: 'group',
            children,
          });
        }
        continue;
      }
      if (node.capabilities.favorite) {
        rows.push({
          key: `menu:${node.menuId}`,
          menuId: node.menuId,
          appCode,
          label: node.name,
          iconKey: node.icon,
          type: node.type,
        });
      }
    }
    return rows;
  };
  return build(null);
}

async function ensureAppChildren(app: CatalogNode) {
  if (childrenByApp.value[app.appCode]) return;
  const nextLoading = new Set(loadingApps.value);
  nextLoading.add(app.appCode);
  loadingApps.value = nextLoading;
  try {
    const menu = await getAppMenuByCode(app.appCode);
    childrenByApp.value = {
      ...childrenByApp.value,
      [app.appCode]: buildCatalogTree(menu, app.appCode),
    };
  } catch (error) {
    console.warn('[favorite-picker] load menu failed', error);
    ElMessage.error(`「${app.label}」菜单加载失败，请稍后重试`);
  } finally {
    const done = new Set(loadingApps.value);
    done.delete(app.appCode);
    loadingApps.value = done;
  }
}

async function toggleExpanded(node: CatalogNode) {
  const next = new Set(expandedKeys.value);
  const opening = !next.has(node.key);
  if (opening) {
    next.add(node.key);
  } else {
    next.delete(node.key);
  }
  expandedKeys.value = next;
  if (opening && node.type === 'app') {
    await ensureAppChildren(node);
  }
}

function isChecked(menuId: string) {
  return draftChecked.value.has(menuId);
}

function toggleChecked(node: CatalogNode) {
  if (!node.menuId) return;
  const next = new Map(draftChecked.value);
  if (next.has(node.menuId)) {
    next.delete(node.menuId);
  } else {
    next.set(node.menuId, node.appCode);
  }
  draftChecked.value = next;
}

const searching = computed(() => searchText.value.trim().length > 0);

function filterTree(nodes: CatalogNode[], keyword: string): CatalogNode[] {
  return nodes.flatMap((node) => {
    const matchingChildren = node.children ? filterTree(node.children, keyword) : [];
    const isMatched = node.label.toLocaleLowerCase().includes(keyword);
    if (!isMatched && matchingChildren.length === 0) return [];
    return [{ ...node, children: matchingChildren.length ? matchingChildren : node.children }];
  });
}

/** 拍平渲染行：先装配（应用行挂接已加载子树）再按关键词裁剪，搜索时
 * 全展开；应用子树仅在展开且已加载后出现。 */
const catalogRows = computed<CatalogRow[]>(() => {
  const keyword = searchText.value.trim().toLocaleLowerCase();
  const merged = apps.value.map((app) => ({
    ...app,
    children: childrenByApp.value[app.appCode],
  }));
  const tree = keyword ? filterTree(merged, keyword) : merged;
  const rows: CatalogRow[] = [];
  const walk = (nodes: CatalogNode[], depth: number) => {
    for (const node of nodes) {
      const expanded = searching.value || expandedKeys.value.has(node.key);
      const loading = node.type === 'app' && loadingApps.value.has(node.appCode);
      rows.push({
        ...node,
        depth,
        expandable: Boolean(node.children?.length) || loading,
        expanded,
        loading,
      });
      if (node.children?.length && expanded) {
        walk(node.children, depth + 1);
      }
    }
  };
  walk(tree, 0);
  return rows;
});

/** 差量应用：新增勾选逐个收藏、取消勾选逐个取消；部分失败仅提示数量。 */
async function confirm() {
  if (applying.value) return;
  applying.value = true;
  const initial = new Map(items.value.map((item) => [item.node.menuId, item.app.code]));
  const additions = [...draftChecked.value.entries()].filter(
    ([menuId]) => !initial.has(menuId),
  ) as [menuId: string, appCode: string][];
  const removals = [...initial.keys()].filter((menuId) => !draftChecked.value.has(menuId));
  try {
    const results = await Promise.allSettled([
      ...additions.map(([menuId, appCode]) => favorite(appCode, menuId)),
      ...removals.map((menuId) => unfavorite(menuId)),
    ]);
    const failed = results.filter((result) => result.status === 'rejected').length;
    if (failed > 0) {
      ElMessage.warning(`有 ${failed} 项收藏未保存成功，请稍后重试`);
    } else {
      ElMessage.success('收藏已更新');
    }
    emit('applied');
    visible.value = false;
  } finally {
    applying.value = false;
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="favorite-picker-dialog"
    width="630px"
    top="8vh"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
  >
    <template #header>
      <header class="favorite-picker-dialog__header">
        <h2 class="favorite-picker-dialog__heading">添加收藏</h2>
        <el-button
          text
          class="favorite-picker-dialog__close"
          :icon="RiCloseFill"
          aria-label="关闭"
          @click="visible = false"
        />
      </header>
    </template>

    <div class="favorite-picker-dialog__body">
      <el-input
        v-model="searchText"
        class="favorite-picker-dialog__search"
        :prefix-icon="RiSearchFill"
        placeholder="搜索应用或入口名称"
        clearable
      />

      <div class="favorite-picker-dialog__list" role="tree" aria-label="应用与菜单入口列表">
        <div v-if="appsStatus === 'loading'" class="favorite-picker-dialog__hint">加载中…</div>
        <div v-else-if="appsStatus === 'error'" class="favorite-picker-dialog__hint">
          应用目录加载失败
          <el-button text type="primary" @click="loadApps"> 重试 </el-button>
        </div>
        <div v-else-if="!catalogRows.length" class="favorite-picker-dialog__hint">
          {{ searching ? '没有匹配的入口' : '暂无可用应用' }}
        </div>
        <template v-else>
          <div
            v-for="row in catalogRows"
            :key="row.key"
            class="favorite-picker-dialog__row"
            role="treeitem"
            :style="{ paddingLeft: `${row.depth * 32}px` }"
          >
            <button
              v-if="row.expandable"
              type="button"
              class="favorite-picker-dialog__expander"
              :aria-label="row.expanded ? `收起${row.label}` : `展开${row.label}`"
              @click="toggleExpanded(row)"
            >
              <el-icon>
                <component :is="row.expanded ? RiArrowDownSFill : RiArrowRightSFill" />
              </el-icon>
            </button>
            <span v-else class="favorite-picker-dialog__indent" aria-hidden="true" />
            <span class="favorite-picker-dialog__app-icon" aria-hidden="true">
              <el-icon><component :is="resolveMenuIcon(row.type, row.iconKey)" /></el-icon>
            </span>
            <span class="favorite-picker-dialog__name">{{ row.label }}</span>
            <button
              v-if="row.menuId"
              type="button"
              class="favorite-picker-dialog__checkbox"
              :class="{ 'favorite-picker-dialog__checkbox--checked': isChecked(row.menuId) }"
              role="checkbox"
              :aria-checked="isChecked(row.menuId)"
              :aria-label="`收藏${row.label}`"
              @click="toggleChecked(row)"
            >
              <el-icon v-if="isChecked(row.menuId)">
                <RiCheckFill />
              </el-icon>
            </button>
          </div>
        </template>
      </div>
    </div>

    <template #footer>
      <div class="favorite-picker-dialog__footer">
        <el-button size="large" @click="visible = false"> 取消 </el-button>
        <el-button type="primary" size="large" :loading="applying" @click="confirm">
          确定
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.favorite-picker-dialog.el-dialog {
  display: flex;
  flex-direction: column;
  height: 600px;
  margin-bottom: 0;
  overflow: hidden;
  border-radius: var(--el-border-radius-round);
}

.favorite-picker-dialog .el-dialog__header {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.favorite-picker-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  padding: var(--el-space-3xl) var(--el-space-3xl);
  overflow: hidden;
}

.favorite-picker-dialog .el-dialog__footer {
  flex: 0 0 auto;
  padding: var(--el-space-xl) var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color-lighter);
}

.favorite-picker-dialog__header {
  position: relative;
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 var(--el-space-3xl);
}

.favorite-picker-dialog__heading {
  margin: 0;
  font-size: var(--el-font-size-medium);
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.favorite-picker-dialog__close.el-button {
  position: absolute;
  top: 10px;
  right: 14px;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-medium);
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
}

.favorite-picker-dialog__body {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.favorite-picker-dialog__search.el-input {
  flex: 0 0 auto;
}

.favorite-picker-dialog__search .el-input__wrapper {
  min-height: 48px;
  padding: 0 var(--el-space-lg);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-medium);
  box-shadow: none;
}

.favorite-picker-dialog__search .el-input__inner {
  font-size: var(--el-font-size-medium);
}

.favorite-picker-dialog__search .el-input__prefix-inner {
  font-size: var(--el-font-size-medium);
}

.favorite-picker-dialog__list {
  flex: 1;
  min-height: 0;
  padding: var(--el-space-lg) 0;
  overflow: auto;
  scrollbar-color: var(--el-border-color) transparent;
}

.favorite-picker-dialog__hint {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  gap: var(--el-space-md);
  color: var(--el-text-color-secondary);
}

.favorite-picker-dialog__row {
  display: flex;
  align-items: center;
  min-height: 48px;
  padding-right: var(--el-space-lg);
}

.favorite-picker-dialog__expander,
.favorite-picker-dialog__indent {
  display: inline-flex;
  flex: 0 0 26px;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
}

.favorite-picker-dialog__expander {
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;

  &:hover {
    color: var(--el-color-primary);
  }
}

.favorite-picker-dialog__app-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-right: var(--el-space-md);
  color: var(--el-color-white);
  background: var(--el-color-primary);
  border-radius: var(--el-border-radius-medium);
}

.favorite-picker-dialog__name {
  min-width: 0;
  overflow: hidden;
  font-size: var(--el-font-size-medium);
  line-height: 1.4;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.favorite-picker-dialog__checkbox {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  margin-left: auto;
  color: var(--el-color-white);
  cursor: pointer;
  background: var(--el-bg-color);
  border: 2px solid var(--el-border-color-darker);
  border-radius: var(--el-border-radius-medium);
}

.favorite-picker-dialog__checkbox--checked {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
}

.favorite-picker-dialog__checkbox .el-icon {
  font-size: var(--el-font-size-large);
}

.favorite-picker-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--el-space-lg);
}

.favorite-picker-dialog__footer .el-button {
  min-width: 88px;
  height: 44px;
  margin: 0;
  font-size: var(--el-font-size-medium);
  border-radius: var(--el-border-radius-medium);
}

@media (max-width: 720px) {
  .favorite-picker-dialog.el-dialog {
    width: 100vw !important;
    height: 100vh;
    top: 0;
    border-radius: 0;
  }
  .favorite-picker-dialog .el-dialog__body {
    padding: var(--el-space-3xl) var(--el-space-2xl);
  }
  .favorite-picker-dialog .el-dialog__footer {
    padding: var(--el-space-xl) var(--el-space-2xl);
  }
  .favorite-picker-dialog__header {
    height: 76px;
    padding: 0 var(--el-space-2xl);
  }
  .favorite-picker-dialog__name {
    font-size: var(--el-font-size-large);
  }
  .favorite-picker-dialog__footer .el-button {
    height: 46px;
    font-size: var(--el-font-size-medium);
  }
}
</style>
