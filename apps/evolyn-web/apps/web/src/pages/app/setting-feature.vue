<script setup lang="ts">
defineOptions({ name: 'ApplicationSettingFeaturePage' });

const props = defineProps<{
  title: string;
  description: string;
  capabilities: string[];
}>();
</script>

<template>
  <section class="application-setting-feature">
    <header class="application-setting-feature__header">
      <h1 class="application-setting-feature__title">{{ props.title }}</h1>
      <p class="application-setting-feature__description">{{ props.description }}</p>
    </header>

    <section
      class="application-setting-feature__card"
      aria-labelledby="application-setting-plan-title"
    >
      <div class="application-setting-feature__card-heading">
        <h2 id="application-setting-plan-title" class="application-setting-feature__card-title">
          功能规划
        </h2>
        <span class="application-setting-feature__status">规划中</span>
      </div>
      <ul class="application-setting-feature__capabilities">
        <li v-for="capability in props.capabilities" :key="capability">{{ capability }}</li>
      </ul>
    </section>
  </section>
</template>

<style scoped lang="scss">
.application-setting-feature {
  box-sizing: border-box;
  min-height: 100%;
  padding: 28px;

  &__header {
    padding-bottom: 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title,
  &__description,
  &__card-title {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 650;
    line-height: 28px;
  }

  &__description {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }

  &__card {
    max-width: 760px;
    margin-top: 24px;
    padding: 20px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    background: var(--el-fill-color-blank);
  }

  &__card-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  &__card-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }

  &__status {
    padding: 2px 8px;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-size: 12px;
    line-height: 20px;
  }

  &__capabilities {
    display: grid;
    margin: 16px 0 0;
    padding-left: 20px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    color: var(--el-text-color-regular);
    font-size: 14px;
    line-height: 28px;
  }
}

@media (max-width: 720px) {
  .application-setting-feature {
    padding: 20px;

    &__capabilities {
      grid-template-columns: 1fr;
    }
  }
}
</style>
