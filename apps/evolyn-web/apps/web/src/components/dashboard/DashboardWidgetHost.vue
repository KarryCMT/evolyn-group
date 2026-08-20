<script setup lang="ts">
import { computed } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import { dashboardWidgetRegistry } from './widget-registry';

defineOptions({ name: 'DashboardWidgetHost' });

const props = defineProps<{ widget: DashboardWidgetContent }>();
const component = computed(() => dashboardWidgetRegistry[props.widget.type]);
</script>

<template>
  <component :is="component" :widget="widget" />
</template>
