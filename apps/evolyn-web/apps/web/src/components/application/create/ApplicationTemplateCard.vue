<script setup lang="ts">
import type { ApplicationTemplate } from './applicationTemplateCatalog';

defineOptions({ name: 'ApplicationTemplateCard' });

const props = defineProps<{
  template: ApplicationTemplate;
}>();

const emit = defineEmits<{
  select: [template: ApplicationTemplate];
}>();
</script>

<template>
  <button class="application-template-card" type="button" @click="emit('select', props.template)">
    <img
      class="application-template-card__image"
      :class="`application-template-card__image--${props.template.imageVariant}`"
      :src="props.template.image"
      :alt="`${props.template.title}模板预览`"
    />
    <span class="application-template-card__content">
      <strong class="application-template-card__title">{{ props.template.title }}</strong>
      <span class="application-template-card__description">{{ props.template.description }}</span>
    </span>
  </button>
</template>

<style scoped lang="scss">
.application-template-card {
  display: flex;
  min-width: 0;
  padding: 0;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  box-shadow: 0 1px 2px rgb(31 35 41 / 4%);
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

  &__image {
    display: block;
    width: 100%;
    aspect-ratio: 1.72;
    object-fit: cover;
    object-position: center top;

    &--inventory {
      filter: hue-rotate(24deg) saturate(0.88);
    }

    &--project {
      filter: hue-rotate(78deg) saturate(0.78);
    }

    &--work-order {
      filter: hue-rotate(148deg) saturate(0.7) brightness(0.8);
    }
  }

  &__content {
    display: flex;
    min-width: 0;
    padding: 10px 12px 12px;
    flex-direction: column;
    gap: 6px;
  }

  &__title {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 15px;
    font-weight: 650;
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__description {
    display: -webkit-box;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.45;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}
</style>
