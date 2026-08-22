<script setup lang="ts">
import { computed } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import { dashboardWidgetRegistry } from './widget-registry';

defineOptions({ name: 'DashboardWidgetHost' });

const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    /** 设计器预览仅展示卡片内容，隐藏成员端的业务操作入口。 */
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
const component = computed(() => dashboardWidgetRegistry[props.widget.type]);
const componentProps = computed(() =>
  ['apps', 'favorites', 'charts', 'greeting'].includes(props.widget.type)
    ? { editorMode: props.editorMode }
    : {},
);
</script>

<template>
  <component :is="component" :widget="widget" v-bind="componentProps" />
</template>
