<script setup lang="ts">
import type {
  BusinessDashboardWidget,
  BusinessDashboardWidgetDescriptor,
  DashboardWidgetContent,
  DashboardWidgetPreset,
} from '@evolyn.do/dashboard';
import { BusinessDashboardWidgetView, setupDashboardWidgetDragSources } from '@evolyn.do/dashboard';
import {
  RiArrowLeftDoubleFill,
  RiArrowRightDoubleFill,
  RiBarChartGroupedLine,
  RiTableLine,
} from '@remixicon/vue';
import { computed, markRaw, onMounted } from 'vue';

const props = defineProps<{
  descriptors: readonly BusinessDashboardWidgetDescriptor[];
  collapsed?: boolean;
}>();
const emit = defineEmits<{
  add: [descriptor: BusinessDashboardWidgetDescriptor];
  toggle: [];
}>();

const widgetRegistry = {
  chart: markRaw(BusinessDashboardWidgetView),
  table: markRaw(BusinessDashboardWidgetView),
};

const dragPresets = computed<DashboardWidgetPreset<BusinessDashboardWidget['type']>[]>(() =>
  props.descriptors.map((descriptor) => ({
    key: descriptor.type,
    type: descriptor.type,
    title: descriptor.defaultTitle,
    w: descriptor.defaultLayout.w,
    h: descriptor.defaultLayout.h,
    minW: 1,
    minH: 1,
    maxW: 12,
    config: {
      businessWidget: {
        id: `palette-${descriptor.type}`,
        type: descriptor.type,
        title: descriptor.defaultTitle,
        layout: { x: 0, y: 0, ...descriptor.defaultLayout },
        settings: structuredClone(descriptor.defaultSettings),
      } as BusinessDashboardWidget,
    },
  })),
);

function getBusinessWidgetProps(widget: DashboardWidgetContent<BusinessDashboardWidget['type']>) {
  return { widget: widget.config?.businessWidget };
}

/**
 * 组件库与画布共享 GridStack 的外部拖放协议。拖动项只携带受控业务默认值，
 * 真正的组件 ID 与持久化写入仍由业务编辑器在 drop 后统一生成。
 */
onMounted(() =>
  setupDashboardWidgetDragSources({
    presets: dragPresets.value,
    widgetRegistry,
    getWidgetProps: (widget) => ({
      widget,
      widgetRegistry,
      getComponentProps: getBusinessWidgetProps,
    }),
  }),
);
</script>

<template>
  <aside class="component-palette" :class="{ 'is-collapsed': collapsed }" aria-label="组件库">
    <button
      class="component-palette__collapse"
      type="button"
      :aria-label="collapsed ? '展开组件库' : '收起组件库'"
      @click="emit('toggle')"
    >
      <RiArrowRightDoubleFill v-if="collapsed" aria-hidden="true" />
      <RiArrowLeftDoubleFill v-else aria-hidden="true" />
    </button>
    <header class="component-palette__header">
      <strong>图表</strong>
    </header>
    <div class="component-palette__list">
      <div
        v-for="descriptor in descriptors"
        :key="descriptor.type"
        class="component-palette__item dashboard-widget-palette__drag-source"
        role="button"
        tabindex="0"
        :data-widget-key="descriptor.type"
        :title="descriptor.description"
        :aria-label="`添加${descriptor.label}`"
        @click="emit('add', descriptor)"
        @keydown.enter="emit('add', descriptor)"
        @keydown.space.prevent="emit('add', descriptor)"
      >
        <span class="component-palette__icon">
          <RiBarChartGroupedLine v-if="descriptor.type === 'chart'" aria-hidden="true" />
          <RiTableLine v-else aria-hidden="true" />
        </span>
        <span class="component-palette__copy">
          <strong>{{ descriptor.label }}</strong>
        </span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.component-palette {
  position: relative;
  display: flex;
  flex: 0 0 190px;
  flex-direction: column;
  min-height: 0;
  padding: 18px 12px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);
  transition:
    flex-basis 0.18s ease,
    padding 0.18s ease;
}

.component-palette.is-collapsed {
  flex-basis: 46px;
  padding: 18px 6px;
}

.component-palette__collapse {
  position: absolute;
  top: 14px;
  right: -14px;
  z-index: 5;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  box-shadow: var(--el-box-shadow-light);
}

.component-palette__collapse svg {
  width: 16px;
}

.component-palette__header {
  display: flex;
  flex-direction: column;
  padding: 0 8px 14px;
}

.component-palette__header strong {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.component-palette__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.component-palette__item {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 9px;
  align-items: center;
  min-height: 42px;
  padding: 0 8px;
  color: inherit;
  text-align: left;
  cursor: grab;
  background: transparent;
  border: 0;
  border-radius: 4px;
  transition:
    color 0.15s ease,
    background 0.15s ease;
}

.component-palette__item:active {
  cursor: grabbing;
}

.component-palette__item:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.component-palette__item:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -2px;
}

.component-palette__icon {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  font-size: 17px;
  color: var(--el-text-color-regular);
}

.component-palette__copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.component-palette__copy strong {
  font-size: 14px;
  font-weight: 500;
}

.component-palette.is-collapsed .component-palette__header,
.component-palette.is-collapsed .component-palette__copy {
  display: none;
}

.component-palette.is-collapsed .component-palette__item {
  grid-template-columns: 1fr;
  justify-items: center;
  padding: 0;
}
</style>
