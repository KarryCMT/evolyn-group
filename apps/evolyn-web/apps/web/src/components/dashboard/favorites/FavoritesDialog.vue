<script setup lang="ts">
import { RiAddFill, RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { useRouter } from 'vue-router';
import type { MenuFavoriteItem } from '~/types';
import { resolveMenuIcon } from '~/components/app/menuIcon';
import { useMenuFavorites } from '~/composables/useMenuFavorites';
import { favoriteTargetRoute } from './favoriteTarget';

defineOptions({ name: 'FavoritesDialog' });

const emit = defineEmits<{
  add: [];
}>();

const visible = defineModel<boolean>({ default: false });
const router = useRouter();
const { items, status, errorMessage, nextCursor, load, loadMore, unfavorite } = useMenuFavorites({
  ensureLoaded: true,
});

function iconOf(item: MenuFavoriteItem) {
  return resolveMenuIcon(item.node.type, item.node.icon);
}

function openFavorite(item: MenuFavoriteItem) {
  const target = favoriteTargetRoute(item);
  if (target) {
    visible.value = false;
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
  <el-drawer
    v-model="visible"
    class="favorites-dialog"
    direction="btt"
    size="90%"
    :show-close="false"
    :close-on-click-modal="false"
    :lock-scroll="true"
    append-to-body
  >
    <template #header>
      <header class="favorites-dialog__header">
        <h1 class="favorites-dialog__heading">我的收藏</h1>
        <el-button
          text
          class="favorites-dialog__close"
          :icon="RiCloseFill"
          aria-label="关闭"
          @click="visible = false"
        />
      </header>
    </template>

    <main class="favorites-dialog__content">
      <section class="favorites-dialog__panel" aria-labelledby="favorites-list-heading">
        <header class="favorites-dialog__panel-header">
          <h2 id="favorites-list-heading" class="favorites-dialog__panel-title">我的收藏</h2>
          <el-button text class="favorites-dialog__add" :icon="RiAddFill" @click="emit('add')">
            添加
          </el-button>
        </header>

        <div v-if="items.length" class="favorites-dialog__grid">
          <div v-for="item in items" :key="item.node.menuId" class="favorites-dialog__app-wrap">
            <button type="button" class="favorites-dialog__app" @click="openFavorite(item)">
              <span class="favorites-dialog__app-icon" aria-hidden="true">
                <el-icon><component :is="iconOf(item)" /></el-icon>
              </span>
              <span class="favorites-dialog__app-text">
                <span class="favorites-dialog__app-name">{{ item.node.name }}</span>
                <span class="favorites-dialog__app-meta">{{ item.app.name }}</span>
              </span>
            </button>
            <el-button
              text
              class="favorites-dialog__remove"
              :icon="RiCloseFill"
              :aria-label="`取消收藏${item.node.name}`"
              @click="removeFavorite(item)"
            />
          </div>
        </div>
        <div v-else-if="status === 'ready'" class="favorites-dialog__empty">
          <span>暂未收藏入口</span>
          <el-button type="primary" @click="emit('add')"> 添加收藏 </el-button>
        </div>
        <div v-else-if="status === 'loading'" class="favorites-dialog__empty">加载中…</div>
        <div v-else class="favorites-dialog__empty">
          <span>{{ errorMessage || '收藏加载失败' }}</span>
          <el-button @click="() => load(true)"> 重试 </el-button>
        </div>

        <div v-if="nextCursor && items.length" class="favorites-dialog__more">
          <el-button text type="primary" @click="loadMore"> 加载更多 </el-button>
        </div>
      </section>
    </main>
  </el-drawer>
</template>

<style lang="scss">
/* 抽屉传送至 body，使用唯一块类将样式限定在收藏面板内。 */
.favorites-dialog.el-drawer {
  display: flex;
  flex-direction: column;
  width: 100vw;
  min-height: 500px;
  overflow: hidden;
  /* 使用页面语义色，抽屉传送至 body 后仍能跟随明暗主题切换。 */
  background: var(--el-bg-color-page);
  box-shadow: none;
}

.favorites-dialog .el-drawer__header {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.favorites-dialog .el-drawer__body {
  flex: 1;
  min-height: 0;
  padding: 0;
}

.favorites-dialog__header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56px;
}

.favorites-dialog__heading,
.favorites-dialog__panel-title {
  margin: 0;
  color: var(--el-text-color-primary);
}

.favorites-dialog__heading {
  font-size: var(--el-font-size-large);
  font-weight: 650;
  line-height: 26px;
}

.favorites-dialog__close.el-button {
  position: absolute;
  top: 12px;
  right: 16px;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
}

.favorites-dialog__close.el-button .el-icon {
  font-size: var(--el-font-size-medium);
}

.favorites-dialog__content {
  box-sizing: border-box;
  height: 100%;
  padding: var(--el-space-3xl) var(--el-space-4xl);
}

.favorites-dialog__panel {
  min-height: 248px;
  padding: var(--el-space-3xl) var(--el-space-4xl);
  background: var(--el-bg-color);
  border-radius: var(--el-border-radius-large);
}

.favorites-dialog__panel-header {
  display: flex;
  align-items: center;
  gap: var(--el-space-lg);
}

.favorites-dialog__panel-title {
  font-size: var(--el-font-size-medium);
  font-weight: 650;
  line-height: 1.2;
}

.favorites-dialog__add.el-button {
  height: 30px;
  padding: 0;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-medium);
}

.favorites-dialog__add .el-icon {
  font-size: var(--el-font-size-medium);
}

.favorites-dialog__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--el-space-3xl) var(--el-space-2xl);
  padding: var(--el-space-4xl) var(--el-space-2xl) var(--el-space-md);
}

.favorites-dialog__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  gap: var(--el-space-xl);
  color: var(--el-text-color-secondary);
}

.favorites-dialog__more {
  display: flex;
  justify-content: center;
}

.favorites-dialog__app-wrap {
  position: relative;
  min-width: 0;
}

.favorites-dialog__app {
  display: flex;
  align-items: center;
  min-width: 0;
  padding: 0;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
}

.favorites-dialog__app:hover .favorites-dialog__app-name {
  color: var(--el-color-primary);
}

.favorites-dialog__app-icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin-right: var(--el-space-xl);
  color: var(--el-color-white);
  background: var(--el-color-primary);
  border-radius: var(--el-border-radius-large);
}

.favorites-dialog__app-icon .el-icon {
  font-size: var(--el-font-size-medium);
}

.favorites-dialog__app-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.favorites-dialog__app-name {
  overflow: hidden;
  font-size: var(--el-font-size-medium);
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.favorites-dialog__app-meta {
  overflow: hidden;
  font-size: var(--el-font-size-small);
  line-height: 1.4;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.favorites-dialog__remove.el-button {
  position: absolute;
  top: -6px;
  right: -6px;
  display: none;
  width: 24px;
  height: 24px;
  padding: 0;
  color: var(--el-text-color-secondary);
  border-radius: var(--el-border-radius-circle);

  &:hover {
    color: var(--el-color-danger);
    background: var(--el-fill-color-light);
  }
}

.favorites-dialog__app-wrap:hover .favorites-dialog__remove {
  display: inline-flex;
}

@media (max-width: 960px) {
  .favorites-dialog__content {
    padding: var(--el-space-3xl) var(--el-space-xl);
  }
  .favorites-dialog__panel {
    padding: var(--el-space-3xl) var(--el-space-3xl);
  }
  .favorites-dialog__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding-inline: 0;
  }
  .favorites-dialog__app-icon {
    margin-right: var(--el-space-lg);
  }
  .favorites-dialog__app-name,
  .favorites-dialog__app-meta {
    font-size: var(--el-font-size-large);
  }
}
</style>
