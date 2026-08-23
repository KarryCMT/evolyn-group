<script setup lang="ts">
import type { FormFieldPreset } from '../schema';
import FormDesignCanvas from './FormDesignCanvas.vue';
import FormFieldPalette from './FormFieldPalette.vue';
import FormPropertyPanel from './FormPropertyPanel.vue';

const emit = defineEmits<{
  selectField: [preset: FormFieldPreset];
  openRecycleBin: [];
}>();

defineSlots<{
  canvas?: () => unknown;
}>();
</script>

<template>
  <section class="form-designer">
    <FormFieldPalette
      class="form-designer__palette"
      @select-field="emit('selectField', $event)"
      @open-recycle-bin="emit('openRecycleBin')"
    />
    <FormDesignCanvas class="form-designer__canvas">
      <slot name="canvas" />
    </FormDesignCanvas>
    <FormPropertyPanel class="form-designer__inspector" />
  </section>
</template>

<style scoped lang="scss">
.form-designer {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: 260px minmax(0, 1fr) 300px;
}

@media (max-width: 760px) {
  .form-designer {
    grid-template-columns: 260px minmax(0, 1fr);

    &__inspector {
      display: none;
    }
  }
}
</style>
