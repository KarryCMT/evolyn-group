<script setup lang="ts" generic="TType extends string">
import { computed, type Component } from 'vue';
import type { DashboardWidgetContent } from '../schema';

defineOptions({ name: 'DashboardWidgetHost' });

const props = defineProps<{
  widget: DashboardWidgetContent<TType>;
  widgetRegistry: Partial<Record<TType, Component>>;
  getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
}>();

/** registry 与业务 props 均由接入应用提供，包不依赖任何具体卡片类型。 */
const component = computed(() => props.widgetRegistry[props.widget.type]);
const componentProps = computed(() => props.getComponentProps?.(props.widget) ?? {});
</script>

<template>
  <component v-if="component" :is="component" :widget="widget" v-bind="componentProps" />
</template>
