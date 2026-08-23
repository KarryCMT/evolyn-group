<script setup lang="ts">
import { RiSettings3Fill } from '@remixicon/vue';
import ApplicationAssetStarterCard from './ApplicationAssetStarterCard.vue';
import { applicationAssetStarters, type ApplicationAssetStarter } from './applicationAssetCatalog';

defineOptions({ name: 'ApplicationEmptyState' });

const emit = defineEmits<{
  selectAsset: [starter: ApplicationAssetStarter];
  openManagement: [];
}>();
</script>

<template>
  <main class="application-empty-state">
    <section class="application-empty-state__content" aria-labelledby="application-empty-heading">
      <header class="application-empty-state__heading-row">
        <h1 id="application-empty-heading" class="application-empty-state__heading">
          创建以下对象，开始构建应用
        </h1>
        <button
          class="application-empty-state__learn-more"
          type="button"
          @click="emit('selectAsset', applicationAssetStarters[1])"
        >
          了解表单和仪表盘
        </button>
      </header>

      <div class="application-empty-state__starter-grid">
        <ApplicationAssetStarterCard
          v-for="starter in applicationAssetStarters"
          :key="starter.type"
          :starter="starter"
          @select="emit('selectAsset', $event)"
        />
      </div>

      <button
        class="application-empty-state__management"
        type="button"
        @click="emit('openManagement')"
      >
        <RiSettings3Fill aria-hidden="true" />
        应用后台
      </button>
    </section>
  </main>
</template>

<style scoped lang="scss">
.application-empty-state {
  display: grid;
  min-height: 0;
  padding: 48px 24px 64px;
  flex: 1;
  place-items: center;
  background: #f7f8fa;

  &__content {
    width: min(100%, 1140px);
  }

  &__heading-row {
    display: flex;
    margin-bottom: 20px;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
  }

  &__heading {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 22px;
    font-weight: 650;
    line-height: 32px;
  }

  &__learn-more,
  &__management {
    padding: 0;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;
    font-size: 15px;
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
    grid-template-columns: repeat(3, 270px);
    justify-content: space-between;
    gap: 24px;
  }

  &__management {
    display: flex;
    width: fit-content;
    margin: 42px auto 0;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-regular);
    font-size: 16px;
    font-weight: 600;

    svg {
      font-size: 20px;
    }

    &:hover {
      color: var(--el-color-primary);
    }
  }
}

@media (max-width: 820px) {
  .application-empty-state {
    display: block;
    overflow: auto;
    padding: 32px 16px;

    &__heading-row {
      align-items: flex-start;
      flex-direction: column;
      gap: 6px;
    }

    &__starter-grid {
      grid-template-columns: 1fr;
      justify-items: center;
    }

    &__management {
      margin-top: 28px;
    }
  }
}
</style>
