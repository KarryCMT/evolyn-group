<script setup lang="ts">
import { RiAddLargeFill, RiArrowRightSLine } from '@remixicon/vue';
import type { ApplicationStarter } from './applicationTemplateCatalog';

defineOptions({ name: 'ApplicationStarterCard' });

const props = defineProps<{
  starter: ApplicationStarter;
}>();

const emit = defineEmits<{
  select: [starter: ApplicationStarter];
}>();
</script>

<template>
  <button
    class="application-starter-card"
    :class="{ 'application-starter-card--blank': !props.starter.image }"
    type="button"
    @click="emit('select', props.starter)"
  >
    <img
      v-if="props.starter.image"
      class="application-starter-card__image"
      :src="props.starter.image"
      alt=""
    />
    <span v-else class="application-starter-card__blank-icon" aria-hidden="true">
      <RiAddLargeFill />
    </span>
    <span class="application-starter-card__title">{{ props.starter.title }}</span>
    <RiArrowRightSLine
      v-if="props.starter.image"
      class="application-starter-card__arrow"
      aria-hidden="true"
    />
  </button>
</template>

<style scoped lang="scss">
.application-starter-card {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 112px;
  padding: 0;
  overflow: hidden;
  align-items: flex-start;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: var(--el-fill-color-lighter);
  border: 1px solid transparent;
  border-radius: 14px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: var(--el-box-shadow-light);
    transform: translateY(-2px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &--blank {
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 13px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-color: var(--el-border-color-lighter);
  }

  &__image {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  &__blank-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary);
    font-size: 34px;
    line-height: 1;
  }

  &__title {
    position: relative;
    z-index: 1;
    padding: 16px 18px;
    color: inherit;
    font-size: 16px;
    font-weight: 650;
    line-height: 1.25;
  }

  &--blank &__title {
    padding: 0;
    font-size: 14px;
    font-weight: 500;
  }

  &__arrow {
    position: relative;
    z-index: 1;
    margin: 16px 12px 0 auto;
    color: var(--el-text-color-regular);
    font-size: 22px;
  }
}

@media (max-width: 820px) {
  .application-starter-card {
    min-height: 112px;

    &__title {
      padding: 18px;
      font-size: 18px;
    }
  }
}
</style>
