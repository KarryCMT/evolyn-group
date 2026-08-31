<script setup lang="ts">
import type { FormItem, SubformWidget } from '../../schema/types';
import FormSchemaSubformPropertyPanel from '../FormSchemaSubformPropertyPanel.vue';
const model = defineModel<FormItem<SubformWidget>>({ required: true });
const emit = defineEmits<{ renameKey: [key: string] }>();
</script>
<template>
  <FormSchemaSubformPropertyPanel v-model="model" @rename-key="emit('renameKey', $event)" />
</template>
