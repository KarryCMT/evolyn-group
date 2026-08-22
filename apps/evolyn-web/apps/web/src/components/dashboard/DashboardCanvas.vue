<script setup lang="ts">
import { DashboardRenderer, DashboardWidgetHost } from '@evolyn.do/dashboard';
import { markRaw } from 'vue';
import type { DashboardSchema, DashboardWidget } from '~/types/dashboard';
import { dashboardWidgetRegistry, getDashboardWidgetComponentProps } from './widget-registry';

const props = defineProps<{
  schema: DashboardSchema;
}>();

const components = { DashboardWidgetHost: markRaw(DashboardWidgetHost) };

function getWidgetProps(widget: DashboardWidget) {
  return {
    widget,
    widgetRegistry: dashboardWidgetRegistry,
    getComponentProps: getDashboardWidgetComponentProps,
  };
}
</script>

<template>
  <DashboardRenderer
    :schema="schema"
    :components="components"
    widget-component="DashboardWidgetHost"
    :get-widget-props="getWidgetProps"
  />
</template>
