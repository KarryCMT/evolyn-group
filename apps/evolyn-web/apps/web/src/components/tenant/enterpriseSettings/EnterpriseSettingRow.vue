<script setup lang="ts">
defineOptions({ name: 'EnterpriseSettingRow' });

withDefaults(
  defineProps<{
    label: string;
    description?: string;
    beta?: boolean;
    stacked?: boolean;
  }>(),
  {
    description: '',
    beta: false,
    stacked: false,
  },
);
</script>

<template>
  <div class="enterprise-setting-row" :class="{ 'enterprise-setting-row--stacked': stacked }">
    <div class="enterprise-setting-row__label">
      <span>{{ label }}</span>
      <span v-if="beta" class="enterprise-setting-row__beta">Beta</span>
    </div>
    <div class="enterprise-setting-row__control">
      <slot />
    </div>
    <div v-if="description || $slots.description" class="enterprise-setting-row__description">
      <slot name="description">
        {{ description }}
      </slot>
    </div>
  </div>
</template>

<style scoped lang="scss">
.enterprise-setting-row {
  display: grid;
  min-height: 52px;
  grid-template-columns: 184px minmax(180px, auto) minmax(260px, 1fr);
  align-items: center;
  column-gap: 18px;

  &__label {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #3e4758;
    font-size: 15px;
    font-weight: 600;
    line-height: 22px;
  }

  &__beta {
    padding: 0 5px;
    border: 1px solid #f3ba55;
    border-radius: 4px;
    color: #dc9220;
    font-size: 10px;
    font-weight: 500;
    line-height: 17px;
  }

  &__control {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 12px;
  }

  &__description {
    min-width: 0;
    color: #7a8495;
    font-size: 14px;
    line-height: 22px;
  }

  &--stacked {
    align-items: start;

    .enterprise-setting-row__label,
    .enterprise-setting-row__control {
      padding-top: 4px;
    }
  }
}

@media (max-width: 1180px) {
  .enterprise-setting-row {
    grid-template-columns: 160px minmax(180px, auto) minmax(220px, 1fr);
  }
}

@media (max-width: 860px) {
  .enterprise-setting-row {
    grid-template-columns: 150px 1fr;
    row-gap: 4px;

    &__description {
      grid-column: 2;
    }
  }
}
</style>
