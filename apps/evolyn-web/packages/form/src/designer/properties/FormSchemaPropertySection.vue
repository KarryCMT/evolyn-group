<script setup lang="ts">
import type { VNode } from 'vue';

defineProps<{ title: string }>();
defineSlots<{ default?: () => VNode[] }>();
</script>

<template>
  <section class="form-schema-property-section">
    <h3 class="form-schema-property-section__heading">{{ title }}</h3>
    <div v-if="$slots.default" class="form-schema-property-section__content">
      <slot />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-schema-property-section {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-md);

  &__heading {
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 1.5;
    color: var(--el-text-color-primary);
  }

  &__content {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-md);
  }
}
</style>
