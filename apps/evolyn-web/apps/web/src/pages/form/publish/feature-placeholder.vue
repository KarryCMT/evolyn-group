<script setup lang="ts">
defineOptions({ name: 'FormPublishFeaturePlaceholder' });

defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <section class="form-publish-feature-placeholder">
    <header class="form-publish-feature-placeholder__header">
      <h1 class="form-publish-feature-placeholder__title">{{ title }}</h1>
      <p class="form-publish-feature-placeholder__description">{{ description }}</p>
    </header>
    <div class="form-publish-feature-placeholder__body">
      <el-empty description="该发布能力正在建设中" />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-publish-feature-placeholder {
  display: flex;
  min-height: 100%;
  flex-direction: column;

  &__header {
    min-height: 68px;
    padding: 0 var(--el-space-3xl);
    display: flex;
    align-items: center;
    gap: var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    line-height: 28px;
  }

  &__description {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
  }

  &__body {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
  }
}
</style>
