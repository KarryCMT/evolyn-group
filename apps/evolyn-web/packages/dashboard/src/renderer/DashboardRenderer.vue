<script setup lang="ts" generic="TType extends string">
import { EvolynGrid, type EvolynGridOptions } from '@evolyn.do/ui';
import { ElScrollbar } from 'element-plus';
import { computed, markRaw, type Component } from 'vue';
import {
  createDashboardGridItems,
  toDashboardWidgetContent,
  type DashboardSchema,
  type DashboardWidget,
  type DashboardWidgetContent,
} from '../schema';
import DashboardWidgetHost from './DashboardWidgetHost.vue';

const props = withDefaults(
  defineProps<{
    schema: DashboardSchema<TType>;
    widgetRegistry: Partial<Record<TType, Component>>;
    getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
    options?: EvolynGridOptions;
  }>(),
  { options: () => ({}) },
);

/** 成员端网格统一通过包内 Host 渲染，应用侧无需感知 GridStack 的组件注册方式。 */
const components = { DashboardWidgetHost: markRaw(DashboardWidgetHost) };
const gridItems = computed(() =>
  createDashboardGridItems(props.schema.widgets, {
    component: 'DashboardWidgetHost',
    createProps: getWidgetProps,
  }),
);
const gridOptions = computed<EvolynGridOptions>(() => ({
  column: 12,
  cellHeight: 72,
  // 相邻卡片的上下边距各 8px，视觉间距为 16px。
  margin: '8px 12px',
  float: true,
  ...props.options,
}));

function getWidgetProps(widget: DashboardWidget<TType>) {
  return {
    widget: toDashboardWidgetContent(widget),
    widgetRegistry: props.widgetRegistry,
    getComponentProps: props.getComponentProps,
  };
}
</script>

<template>
  <ElScrollbar class="dashboard-renderer" always>
    <main class="dashboard-renderer__surface">
      <EvolynGrid
        :model-value="gridItems"
        :options="gridOptions"
        :components="components"
        :editable="false"
      />
    </main>
  </ElScrollbar>
</template>

<style scoped lang="scss">
.dashboard-renderer {
  flex: 1;
  width: 100%;
  max-width: 1680px;
  min-height: 0;
  margin: 0 auto;

  &__surface {
    box-sizing: border-box;
    min-height: 100%;
    padding: 0 40px 24px;

    /* 工作台卡片使用独立的浮层规格，避免继承 Element Plus 过于克制的默认圆角和阴影。 */
    --el-border-radius-base: 10px;
    --el-border-color-lighter: rgba(31, 35, 41, 0.06);
    --el-box-shadow-lighter: 0 0 2px 0 rgba(19, 29, 46, 0.02), 0 1px 4px 0 rgba(19, 29, 46, 0.06);
  }
}

.dashboard-renderer :deep(.evolyn-grid .grid-stack-item-content) {
  /* 网格内容盒不能裁掉卡片向外扩散的阴影；卡片自身已负责内部内容裁剪。 */
  overflow: visible !important;
}

@media (max-width: 768px) {
  .dashboard-renderer__surface {
    padding-inline: 16px;
  }
}
</style>
