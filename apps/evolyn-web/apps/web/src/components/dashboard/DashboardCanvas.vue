<script setup lang="ts">
import { DashboardRenderer } from '@evolyn.do/dashboard';
import { markRaw } from 'vue';
import type { DashboardSchema } from '~/types/dashboard';
import DashboardWidgetHost from './DashboardWidgetHost.vue';

const props = defineProps<{
  schema: DashboardSchema;
}>();

const components = { DashboardWidgetHost: markRaw(DashboardWidgetHost) };
</script>

<template>
  <DashboardRenderer
    :schema="schema"
    :components="components"
    widget-component="DashboardWidgetHost"
  />
</template>
