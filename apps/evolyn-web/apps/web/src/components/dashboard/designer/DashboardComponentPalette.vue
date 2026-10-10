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
  RiBarChartBoxLine,
  RiBarChartHorizontalLine,
  RiCalendar2Line,
  RiCodeBoxLine,
  RiCursorLine,
  RiDatabase2Line,
  RiFilter2Line,
  RiFilter3Line,
  RiImageLine,
  RiLayout2Line,
  RiListUnordered,
  RiMapPin2Line,
  RiOrganizationChart,
  RiSignpostLine,
  RiTableLine,
  RiTBoxLine,
  RiTimeLine,
} from '@remixicon/vue';
import { computed, markRaw, onMounted, toRaw } from 'vue';

const props = defineProps<{
  descriptors: readonly BusinessDashboardWidgetDescriptor[];
  collapsed?: boolean;
}>();
const emit = defineEmits<{
  add: [descriptor: BusinessDashboardWidgetDescriptor];
  toggle: [];
}>();

interface PaletteCatalogItem {
  key: string;
  label: string;
  icon: typeof RiTableLine;
  widgetType?: BusinessDashboardWidget['type'];
  muted?: boolean;
}

interface PaletteCatalogGroup {
  key: string;
  label: string;
  items: readonly PaletteCatalogItem[];
}

/**
 * 目录完整呈现目标组件体系；仅绑定 widgetType 的条目接入现有协议，
 * 其余条目作为不可交互占位，后续实现时再逐项开放拖放能力。
 */
const paletteCatalog: readonly PaletteCatalogGroup[] = [
  {
    key: 'charts',
    label: '图表',
    items: [
      { key: 'statistical-table', label: '统计表', icon: RiBarChartBoxLine, widgetType: 'chart' },
      { key: 'detail-table', label: '明细表', icon: RiTableLine, widgetType: 'table' },
      { key: 'data-management-table', label: '数据管理表', icon: RiDatabase2Line },
      { key: 'point-map', label: '点地图', icon: RiMapPin2Line, muted: true },
      { key: 'calendar', label: '日历', icon: RiCalendar2Line },
      { key: 'gantt', label: '甘特图', icon: RiBarChartHorizontalLine },
      { key: 'data-list', label: '数据列表', icon: RiListUnordered },
      { key: 'process-analysis', label: '流程分析表', icon: RiOrganizationChart },
    ],
  },
  {
    key: 'components',
    label: '组件',
    items: [
      { key: 'image', label: '图片组件', icon: RiImageLine },
      { key: 'text', label: '文本组件', icon: RiTBoxLine },
      { key: 'real-time', label: '实时时间', icon: RiTimeLine },
      { key: 'shortcut', label: '快捷入口', icon: RiSignpostLine },
      { key: 'embedded-page', label: '嵌入页面', icon: RiCodeBoxLine },
      { key: 'layout-container', label: '布局容器', icon: RiLayout2Line },
    ],
  },
  {
    key: 'tools',
    label: '工具',
    items: [
      { key: 'filter', label: '筛选组件', icon: RiFilter3Line },
      { key: 'quick-filter', label: '快捷筛选', icon: RiFilter2Line },
      { key: 'filter-button', label: '筛选按钮', icon: RiCursorLine },
    ],
  },
];

const paletteGroups = computed(() => {
  const descriptors = new Map(props.descriptors.map((descriptor) => [descriptor.type, descriptor]));
  return paletteCatalog.map((group) => ({
    ...group,
    items: group.items.map((item) => ({
      ...item,
      descriptor: item.widgetType ? descriptors.get(item.widgetType) : undefined,
    })),
  }));
});

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
        settings: structuredClone(toRaw(descriptor.defaultSettings)),
      } as BusinessDashboardWidget,
    },
  })),
);

function getBusinessWidgetProps(widget: DashboardWidgetContent<BusinessDashboardWidget['type']>) {
  return { widget: widget.config?.businessWidget };
}

function addItem(descriptor?: BusinessDashboardWidgetDescriptor) {
  if (descriptor) emit('add', descriptor);
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
    <div class="component-palette__scroll">
      <section
        v-for="group in paletteGroups"
        :key="group.key"
        class="component-palette__group"
        :aria-labelledby="`dashboard-palette-${group.key}`"
      >
        <h2 :id="`dashboard-palette-${group.key}`" class="component-palette__group-title">
          {{ group.label }}
        </h2>
        <div class="component-palette__list">
          <button
            v-for="item in group.items"
            :key="item.key"
            class="component-palette__item"
            :class="{
              'dashboard-widget-palette__drag-source': item.descriptor,
              'is-placeholder': !item.descriptor,
              'is-muted': item.muted,
            }"
            type="button"
            :disabled="!item.descriptor"
            :data-widget-key="item.descriptor?.type"
            :title="item.descriptor?.description ?? `${item.label}暂未开放`"
            :aria-label="item.descriptor ? `添加${item.label}` : `${item.label}暂未开放`"
            @click="addItem(item.descriptor)"
          >
            <span class="component-palette__icon">
              <component :is="item.icon" aria-hidden="true" />
            </span>
            <span class="component-palette__copy">{{ item.label }}</span>
          </button>
        </div>
      </section>
    </div>
  </aside>
</template>

<style scoped>
.component-palette {
  position: relative;
  z-index: 1;
  display: flex;
  flex: 0 0 190px;
  flex-direction: column;
  min-height: 0;
  /* 折叠按钮需要跨出侧栏边界，列表滚动仍由内部滚动容器负责裁剪。 */
  overflow: visible;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);
  transition: flex-basis 0.18s ease;
}

.component-palette.is-collapsed {
  flex-basis: 46px;
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

.component-palette__scroll {
  flex: 1;
  min-height: 0;
  padding: 14px 12px 20px;
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-color: var(--el-border-color) transparent;
  scrollbar-width: thin;
}

.component-palette__scroll::-webkit-scrollbar {
  width: 8px;
}

.component-palette__scroll::-webkit-scrollbar-thumb {
  background: var(--el-border-color);
  border: 2px solid transparent;
  border-radius: 999px;
  background-clip: padding-box;
}

.component-palette__group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.component-palette__group + .component-palette__group {
  margin-top: 14px;
}

.component-palette__group-title {
  margin: 0;
  padding: 0 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.component-palette__list {
  display: flex;
  flex-direction: column;
}

.component-palette__item {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 9px;
  align-items: center;
  width: 100%;
  min-height: 36px;
  padding: 0 8px;
  margin: 0;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: grab;
  background: transparent;
  border: 0;
  border-radius: 4px;
  transition:
    color 0.15s ease,
    background 0.15s ease;
}

.component-palette__item:not(:disabled):active {
  cursor: grabbing;
}

.component-palette__item:not(:disabled):hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.component-palette__item:not(:disabled):focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -2px;
}

.component-palette__item.is-placeholder {
  color: inherit;
  cursor: default;
  opacity: 1;
}

.component-palette__item.is-muted {
  color: var(--el-text-color-placeholder);
}

.component-palette__icon {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  font-size: 18px;
  color: var(--el-text-color-regular);
}

.component-palette__icon svg {
  width: 18px;
  height: 18px;
}

.component-palette__item.is-muted .component-palette__icon {
  color: inherit;
}

.component-palette__copy {
  min-width: 0;
  font-size: 14px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.component-palette.is-collapsed .component-palette__group-title,
.component-palette.is-collapsed .component-palette__copy {
  display: none;
}

.component-palette.is-collapsed .component-palette__scroll {
  padding: 14px 6px 20px;
}

.component-palette.is-collapsed .component-palette__group + .component-palette__group {
  margin-top: 10px;
}

.component-palette.is-collapsed .component-palette__item {
  grid-template-columns: 1fr;
  justify-items: center;
  padding: 0;
}
</style>
