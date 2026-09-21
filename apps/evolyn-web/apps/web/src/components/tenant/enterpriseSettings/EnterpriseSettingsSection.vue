<script setup lang="ts">
defineOptions({ name: 'EnterpriseSettingsSection' });

defineProps<{
  title: string;
  labelledBy: string;
}>();
</script>

<template>
  <section class="enterprise-settings-section" :aria-labelledby="labelledBy">
    <h2 :id="labelledBy" class="enterprise-settings-section__title">
      <span aria-hidden="true" />
      {{ title }}
    </h2>
    <div class="enterprise-settings-section__body">
      <slot />
    </div>
  </section>
</template>

<style scoped lang="scss">
.enterprise-settings-section {
  padding: 28px 20px 30px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: 0;
  }

  &__title {
    display: flex;
    margin: 0 0 24px;
    align-items: center;
    gap: 10px;
    color: #202939;
    font-size: 18px;
    font-weight: 650;
    line-height: 26px;

    span {
      width: 4px;
      height: 18px;
      border-radius: 3px;
      background: var(--enterprise-accent);
    }
  }

  &__body {
    display: grid;
    gap: 10px;
  }
}

@media (max-width: 960px) {
  .enterprise-settings-section {
    padding-inline: 16px;
  }
}
</style>
