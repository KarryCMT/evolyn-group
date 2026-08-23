<script setup lang="ts">
defineOptions({ name: 'FormExtensionFeaturePlaceholder' });

defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <section class="form-extension-feature-placeholder">
    <header class="form-extension-feature-placeholder__header">
      <h1 class="form-extension-feature-placeholder__title">{{ title }}</h1>
      <p class="form-extension-feature-placeholder__description">{{ description }}</p>
    </header>
    <div class="form-extension-feature-placeholder__body">
      <el-empty description="该扩展功能正在建设中" />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-extension-feature-placeholder {
  display: flex;
  min-height: 100%;
  flex-direction: column;

  &__header {
    min-height: 68px;
    padding: 0 28px;
    display: flex;
    align-items: center;
    gap: 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: 19px;
    line-height: 28px;
  }

  &__description {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  &__body {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
  }
}
</style>
