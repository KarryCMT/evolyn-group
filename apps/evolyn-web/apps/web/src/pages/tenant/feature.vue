<script setup lang="ts">
import type { TenantFeaturePageProps } from '~/types/tenant';

defineOptions({ name: 'TenantFeaturePage' });

const props = defineProps<TenantFeaturePageProps>();
</script>

<template>
  <section class="tenant-feature-page">
    <header class="tenant-feature-page__header">
      <h1 class="tenant-feature-page__title">{{ props.title }}</h1>
      <p class="tenant-feature-page__description">{{ props.description }}</p>
    </header>

    <section class="tenant-feature-page__card" aria-labelledby="feature-plan-title">
      <div class="tenant-feature-page__card-heading">
        <h2 id="feature-plan-title" class="tenant-feature-page__card-title">功能规划</h2>
        <span class="tenant-feature-page__status">规划中</span>
      </div>
      <ul class="tenant-feature-page__capabilities">
        <li v-for="capability in props.capabilities" :key="capability">
          {{ capability }}
        </li>
      </ul>
    </section>
  </section>
</template>

<style scoped lang="scss">
.tenant-feature-page {
  /* 高度与内边距合并计算，防止内容未溢出时仍触发父容器滚动。 */
  box-sizing: border-box;
  min-height: 100%;
  padding: var(--el-space-3xl);

  &__header {
    margin-bottom: var(--el-space-3xl);
  }

  &__title,
  &__card-title {
    margin: 0;
    color: var(--el-text-color-primary);
  }

  &__title {
    font-size: var(--el-font-size-extra-large);
    font-weight: 600;
    line-height: 28px;
  }

  &__description {
    margin: var(--el-space-md) 0 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  &__card {
    max-width: 840px;
    padding: var(--el-space-2xl) var(--el-space-3xl);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-large);
    background: var(--el-bg-color);
  }

  &__card-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__card-title {
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__status {
    padding: var(--el-space-xs) var(--el-space-md);
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-size: var(--el-font-size-extra-small);
    line-height: 20px;
  }

  &__capabilities {
    display: grid;
    margin: var(--el-space-xl) 0 0;
    padding: 0;
    gap: var(--el-space-md);
    list-style: none;
  }

  &__capabilities li {
    position: relative;
    padding-left: var(--el-space-xl);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    line-height: 22px;

    &::before {
      position: absolute;
      top: 8px;
      left: 0;
      width: 6px;
      height: 6px;
      border-radius: var(--el-border-radius-half);
      background: var(--el-color-primary);
      content: '';
    }
  }
}
</style>
