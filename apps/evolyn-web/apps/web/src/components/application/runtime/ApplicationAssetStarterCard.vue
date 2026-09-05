<script setup lang="ts">
import type { ApplicationAssetStarter } from './applicationAssetCatalog';
import { RiLoader4Fill } from '@remixicon/vue';
import applicationTypeBackground from '~/assets/images/app-type-bg.png';

defineOptions({ name: 'ApplicationAssetStarterCard' });

const props = defineProps<{
  starter: ApplicationAssetStarter;
  disabled?: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  select: [starter: ApplicationAssetStarter];
}>();
</script>

<template>
  <button
    class="application-asset-starter-card"
    type="button"
    :aria-busy="props.loading || undefined"
    :disabled="props.disabled"
    @click="emit('select', props.starter)"
  >
    <span
      class="application-asset-starter-card__visual"
      :class="`application-asset-starter-card__visual--${props.starter.imagePosition}`"
    >
      <span
        class="application-asset-starter-card__illustration"
        :class="`application-asset-starter-card__illustration--${props.starter.imagePosition}`"
        :style="{ backgroundImage: `url(${applicationTypeBackground})` }"
        aria-hidden="true"
      />
      <strong class="application-asset-starter-card__title">{{ props.starter.title }}</strong>
    </span>
    <span v-if="props.loading" class="application-asset-starter-card__loading" role="status">
      <RiLoader4Fill aria-hidden="true" />
      正在创建并打开设计器…
    </span>
    <span v-else class="application-asset-starter-card__description">
      {{ props.starter.description }}
    </span>
  </button>
</template>

<style scoped lang="scss">
.application-asset-starter-card {
  display: flex;
  box-sizing: border-box;
  width: 230px;
  height: 258px;
  padding: var(--el-space-lg) var(--el-space-lg) var(--el-space-xl);
  flex: 0 0 230px;
  flex-direction: column;
  align-items: center;
  color: var(--el-text-color-primary);
  text-align: center;
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;

  &:disabled {
    cursor: not-allowed;
  }

  &:disabled:not([aria-busy='true']) {
    opacity: 0.62;
  }

  &:not(:disabled):hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: var(--el-box-shadow-light);
    transform: translateY(-2px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 3px;
  }

  &__visual {
    display: flex;
    width: 100%;
    height: 162px;
    flex: 0 0 162px;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    border-radius: var(--el-border-radius-large);

    &--workflow-form {
      background: #fff4e5;
    }

    &--form {
      background: #eaf3ff;
    }

    &--dashboard {
      background: #faedff;
    }
  }

  &__illustration {
    /* 精灵图按 7/8 等比缩放展示（112×136 → 98×119），
       background-size 与各帧 background-position 必须同步缩放，否则画面错位。 */
    display: block;
    width: 98px;
    height: 119px;
    margin-top: -6px;
    flex: 0 0 auto;
    background-repeat: no-repeat;
    background-size: 98px 476px;

    &--form {
      background-position: 0 0;
    }

    &--workflow-form {
      background-position: 0 -119px;
    }

    &--dashboard {
      background-position: 0 -357px;
    }
  }

  &__title {
    margin-top: -7px;
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 26px;
  }

  &__description {
    display: -webkit-box;
    max-width: 198px;
    margin-top: var(--el-space-lg);
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  &__loading {
    display: inline-flex;
    gap: var(--el-space-sm);
    align-items: center;
    margin-top: var(--el-space-md);
    font-size: var(--el-font-size-small);
    line-height: 18px;
    color: var(--el-color-primary);

    svg {
      width: 16px;
      height: 16px;
      animation: application-asset-starter-card-spin 0.8s linear infinite;
    }
  }
}

@media (prefers-reduced-motion: reduce) {
  .application-asset-starter-card__loading svg {
    animation: none;
  }
}

@keyframes application-asset-starter-card-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 820px) {
  .application-asset-starter-card {
    width: min(230px, 100%);
    flex-basis: auto;
  }
}
</style>
