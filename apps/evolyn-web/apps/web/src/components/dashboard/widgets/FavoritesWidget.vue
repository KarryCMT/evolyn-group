<script setup lang="ts">
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import type { DashboardWidgetContent } from '~/types/dashboard';
import type { MenuFavoriteItem } from '~/types';
import { resolveMenuIcon } from '~/components/app/menuIcon';
import FavoritesWorkspaceDialog from '../favorites/FavoritesWorkspaceDialog.vue';
import { favoriteTargetRoute } from '../favorites/favoriteTarget';
import { useMenuFavorites } from '~/composables/useMenuFavorites';

defineOptions({ name: 'FavoritesWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
const router = useRouter();
const favoritesVisible = shallowRef(false);
const { items, status, load, unfavorite } = useMenuFavorites({ ensureLoaded: true });
const isRecent = computed(
  () => props.widget.config?.variant === 'recent' || props.widget.title === '最近使用',
);

// 卡片空间有限，优先展示前四个收藏，完整列表在「我的收藏」面板内查看。
const visibleItems = computed(() => items.value.slice(0, 4));

function iconOf(item: MenuFavoriteItem) {
  return resolveMenuIcon(item.node.type, item.node.icon);
}

function openFavorite(item: MenuFavoriteItem) {
  const target = favoriteTargetRoute(item);
  if (target) {
    void router.push(target);
    return;
  }
  ElMessage.info('该资产类型的入口暂未开放，敬请期待');
}

async function removeFavorite(item: MenuFavoriteItem) {
  try {
    await unfavorite(item.node.menuId);
    ElMessage.success(`已取消收藏「${item.node.name}」`);
  } catch {
    ElMessage.error('取消收藏失败，请稍后重试');
  }
}
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template #actions>
      <el-button
        v-if="!props.editorMode && !isRecent"
        text
        type="primary"
        @click="favoritesVisible = true"
      >
        管理
      </el-button>
    </template>
    <div v-if="isRecent" class="favorites-widget favorites-widget--recent">
      <!-- 最近使用来自访问事件（与显式收藏不同源），接入前保持占位 -->
      <span class="favorites-widget__empty">最近使用即将上线</span>
    </div>
    <div v-else class="favorites-widget">
      <el-button
        v-for="item in visibleItems"
        :key="item.node.menuId"
        text
        class="favorites-widget__item"
        @click="openFavorite(item)"
      >
        <span class="favorites-widget__icon" aria-hidden="true">
          <el-icon><component :is="iconOf(item)" /></el-icon>
        </span>
        <span class="favorites-widget__label">{{ item.node.name }}</span>
        <span
          v-if="!props.editorMode"
          role="button"
          aria-label="取消收藏"
          class="favorites-widget__remove"
          @click.stop="removeFavorite(item)"
        >
          <el-icon><RiCloseFill /></el-icon>
        </span>
      </el-button>
      <span v-if="status === 'ready' && !visibleItems.length" class="favorites-widget__empty">
        暂无收藏
      </span>
      <el-button
        v-else-if="status === 'error'"
        text
        type="primary"
        class="favorites-widget__empty"
        @click="() => load(true)"
      >
        收藏加载失败，点击重试
      </el-button>
    </div>
  </DashboardWidgetFrame>
  <FavoritesWorkspaceDialog v-model="favoritesVisible" />
</template>

<style scoped lang="scss">
.favorites-widget {
  display: flex;
  align-items: center;
  height: 100%;
  gap: var(--el-space-2xl);

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  &--recent {
    padding-left: var(--el-space-xs);
  }
  &__item {
    display: inline-flex;
    max-width: 220px;
    margin: 0;
    color: var(--el-text-color-primary);
  }
  &__label {
    overflow: hidden;
    min-width: 0;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__remove {
    display: none;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    margin-left: var(--el-space-xs);
    color: var(--el-text-color-secondary);
    border-radius: var(--el-border-radius-small);
    cursor: pointer;

    &:hover {
      color: var(--el-color-danger);
      background: var(--el-fill-color-light);
    }
  }
  &__item:hover &__remove {
    display: inline-flex;
  }
  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    margin-right: var(--el-space-md);
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-small);
  }
  &__empty {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}
</style>
