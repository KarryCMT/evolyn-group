<script setup lang="ts">
import type { AppAssetStarter } from './appAssetCatalog';
import { appAssetStarters } from './appAssetCatalog';
import AppAssetStarterCard from './AppAssetStarterCard.vue';

defineOptions({ name: 'AppEmptyState' });

defineProps<{
  creatingAssetType: AppAssetStarter['type'] | null;
}>();

const emit = defineEmits<{
  selectAsset: [starter: AppAssetStarter];
  learnMore: [];
}>();
</script>

<template>
  <main class="app-empty-state">
    <section class="app-empty-state__content" aria-labelledby="app-empty-heading">
      <header class="app-empty-state__heading-row">
        <h1 id="app-empty-heading" class="app-empty-state__heading">创建以下对象，开始构建应用</h1>
        <button class="app-empty-state__learn-more" type="button" @click="emit('learnMore')">
          了解表单和仪表盘
        </button>
      </header>

      <div class="app-empty-state__starter-grid">
        <AppAssetStarterCard
          v-for="starter in appAssetStarters"
          :key="starter.type"
          :starter="starter"
          :disabled="creatingAssetType !== null"
          :loading="creatingAssetType === starter.type"
          @select="emit('selectAsset', $event)"
        />
      </div>
    </section>
  </main>
</template>

<style scoped lang="scss">
.app-empty-state {
  display: grid;
  min-height: 0;
  padding: var(--el-space-6xl) var(--el-space-3xl) 64px;
  flex: 1;
  place-items: center;
  background: var(--el-bg-color-page);

  &__content {
    /* 宽度贴合三张卡片加两列间距：卡片与标题行共享同一版心，缩放卡片时同步收缩。 */
    width: min(100%, calc(230px * 3 + var(--el-space-3xl) * 2));
  }

  &__heading-row {
    display: flex;
    margin-bottom: var(--el-space-2xl);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-3xl);
  }

  &__heading {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    font-weight: 650;
    line-height: 32px;
  }

  &__learn-more {
    padding: 0;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;
    font-size: var(--el-font-size-base);
    line-height: 24px;

    &:hover {
      color: var(--el-color-primary-light-3);
      text-decoration: underline;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 3px;
    }
  }

  &__starter-grid {
    display: grid;
    grid-template-columns: repeat(3, 230px);
    justify-content: space-between;
    gap: var(--el-space-3xl);
  }
}

@media (max-width: 820px) {
  .app-empty-state {
    display: block;
    overflow: auto;
    padding: var(--el-space-4xl) var(--el-space-xl);

    &__heading-row {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--el-space-sm);
    }

    &__starter-grid {
      grid-template-columns: 1fr;
      justify-items: center;
    }
  }
}
</style>
