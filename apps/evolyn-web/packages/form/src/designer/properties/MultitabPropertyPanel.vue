<script setup lang="ts">
import type { FormMultitabLayout, FormTabStyle } from '../../schema/types';
import FormSchemaMultitabPropertyPanel from '../FormSchemaMultitabPropertyPanel.vue';
defineProps<{ layout: FormMultitabLayout }>();
const emit = defineEmits<{
  setTabStyle: [style: FormTabStyle];
  addTab: [];
  removeTab: [tabName: string];
  duplicateTab: [tabName: string];
  renameTab: [tabName: string, title: string];
  reorderTabs: [tabNames: string[]];
}>();
</script>
<template>
  <FormSchemaMultitabPropertyPanel
    :layout="layout"
    @set-tab-style="emit('setTabStyle', $event)"
    @add-tab="emit('addTab')"
    @remove-tab="emit('removeTab', $event)"
    @duplicate-tab="emit('duplicateTab', $event)"
    @rename-tab="(tabName, title) => emit('renameTab', tabName, title)"
    @reorder-tabs="emit('reorderTabs', $event)"
  />
</template>
