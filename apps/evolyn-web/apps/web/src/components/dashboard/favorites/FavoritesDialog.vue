<script setup lang="ts">
import type { FavoriteApplication } from './favoriteCatalog';
import { Close, Plus } from '@element-plus/icons-vue';

defineOptions({ name: 'FavoritesDialog' });

defineProps<{
  applications: FavoriteApplication[];
}>();

const emit = defineEmits<{
  add: [];
}>();

const visible = defineModel<boolean>({ default: false });
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
          :icon="Close"
          aria-label="关闭"
          @click="visible = false"
        />
      </header>
    </template>

    <main class="favorites-dialog__content">
      <section class="favorites-dialog__panel" aria-labelledby="favorites-list-heading">
        <header class="favorites-dialog__panel-header">
          <h2 id="favorites-list-heading" class="favorites-dialog__panel-title">我的收藏</h2>
          <el-button text class="favorites-dialog__add" :icon="Plus" @click="emit('add')">
            添加
          </el-button>
        </header>

        <div v-if="applications.length" class="favorites-dialog__grid">
          <button
            v-for="application in applications"
            :key="application.id"
            type="button"
            class="favorites-dialog__application"
          >
            <span
              class="favorites-dialog__application-icon"
              :class="`favorites-dialog__application-icon--${application.tone}`"
              aria-hidden="true"
            >
              <el-icon><component :is="application.icon" /></el-icon>
            </span>
            <span class="favorites-dialog__application-name">{{ application.label }}</span>
          </button>
        </div>
        <div v-else class="favorites-dialog__empty">
          <span>暂未收藏应用</span>
          <el-button type="primary" @click="emit('add')"> 添加应用 </el-button>
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
  background: #f6f7f9;
  box-shadow: none;

  /* 弹层已传送至 body，显式继承项目品牌蓝。 */
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-9: #e8f1ff;
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
  font-size: var(--el-font-size-extra-large);
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
}

.favorites-dialog__close.el-button .el-icon {
  font-size: var(--el-font-size-extra-large);
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
  font-size: var(--el-font-size-extra-large);
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
  font-size: var(--el-font-size-extra-large);
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

.favorites-dialog__application {
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

.favorites-dialog__application:hover .favorites-dialog__application-name {
  color: var(--el-color-primary);
}

.favorites-dialog__application-icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin-right: var(--el-space-xl);
  color: var(--el-color-white);
  border-radius: var(--el-border-radius-large);
}

.favorites-dialog__application-icon .el-icon {
  font-size: var(--el-font-size-extra-large);
}

.favorites-dialog__application-icon--blue {
  background: #4b8cf7;
}
.favorites-dialog__application-icon--cyan {
  background: #1aaee2;
}
.favorites-dialog__application-icon--green {
  background: #48b860;
}
.favorites-dialog__application-icon--orange {
  background: #ff9d32;
}
.favorites-dialog__application-icon--purple {
  background: #8367ee;
}
.favorites-dialog__application-icon--red {
  background: #f36061;
}

.favorites-dialog__application-name {
  overflow: hidden;
  font-size: var(--el-font-size-extra-large);
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  .favorites-dialog__application-icon {
    margin-right: var(--el-space-lg);
  }
  .favorites-dialog__application-name {
    font-size: var(--el-font-size-large);
  }
}
</style>
