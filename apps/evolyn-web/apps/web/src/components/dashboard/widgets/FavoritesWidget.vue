<script setup lang="ts">
import type { DashboardWidgetContent } from '~/types/dashboard';
import { DataAnalysis } from '@element-plus/icons-vue';
import { computed, shallowRef } from 'vue';
import DashboardWidgetFrame from '../DashboardWidgetFrame.vue';
import FavoritesWorkspaceDialog from '../favorites/FavoritesWorkspaceDialog.vue';
import { useFavoriteApplications } from '../favorites/useFavoriteApplications';

defineOptions({ name: 'FavoritesWidget' });
const props = defineProps<{ widget: DashboardWidgetContent }>();
const favoritesVisible = shallowRef(false);
const { favoriteApplications } = useFavoriteApplications();

// 卡片空间有限，优先展示前四个收藏，完整列表在「我的收藏」面板内查看。
const visibleApplications = computed(() => favoriteApplications.value.slice(0, 4));
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template #actions>
      <el-button
        v-if="props.widget.title !== '最近使用'"
        text
        type="primary"
        @click="favoritesVisible = true"
      >
        添加
      </el-button>
    </template>
    <div v-if="widget.title === '最近使用'" class="favorites-widget favorites-widget--recent">
      <el-button text class="favorites-widget__recent-item" :icon="DataAnalysis">
        合同统计看板
      </el-button>
    </div>
    <div v-else class="favorites-widget">
      <el-button
        v-for="app in visibleApplications"
        :key="app.id"
        text
        class="favorites-widget__item"
        @click="favoritesVisible = true"
      >
        <span class="favorites-widget__icon" :class="`favorites-widget__icon--${app.tone}`">
          <el-icon><component :is="app.icon" /></el-icon>
        </span>
        {{ app.label }}
      </el-button>
      <span v-if="!visibleApplications.length" class="favorites-widget__empty">暂无收藏</span>
    </div>
  </DashboardWidgetFrame>
  <FavoritesWorkspaceDialog v-model="favoritesVisible" />
</template>

<style scoped lang="scss">
.favorites-widget {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 20px;

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  &--recent {
    padding-left: 4px;
  }
  &__recent-item {
    margin: 0;
    color: var(--el-text-color-primary);
  }
  &__item {
    display: inline-flex;
    margin: 0;
    color: var(--el-text-color-primary);
    max-width: 220px;
  }
  &__empty {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    margin-right: 8px;
    color: var(--el-color-white);
    border-radius: var(--el-border-radius-small);

    &--blue {
      background: #4b8cf7;
    }
    &--cyan {
      background: #1aaee2;
    }
    &--green {
      background: #48b860;
    }
    &--orange {
      background: #ff9d32;
    }
    &--purple {
      background: #8367ee;
    }
    &--red {
      background: #f36061;
    }
  }
}
</style>
