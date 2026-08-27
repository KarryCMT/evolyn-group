<script setup lang="ts">
defineOptions({ name: 'FormWorkspacePlaceholder' });

defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <section class="form-workspace-placeholder" :aria-label="title">
    <div class="form-workspace-placeholder__content">
      <h1 class="form-workspace-placeholder__title">{{ title }}</h1>
      <p class="form-workspace-placeholder__description">{{ description }}</p>
      <el-empty description="该功能正在建设中" />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-workspace-placeholder {
  display: flex;
  min-height: 0;
  margin: 0 var(--el-space-md) var(--el-space-md);
  flex: 1;
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__content {
    text-align: center;
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    line-height: 30px;
  }

  &__description {
    margin-top: var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
  }
}
</style>
