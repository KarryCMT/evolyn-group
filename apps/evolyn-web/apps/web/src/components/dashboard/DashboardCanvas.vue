<script setup lang="ts">
import { EditPen, RefreshLeft } from '@element-plus/icons-vue';
import { EvolynGrid, type EvolynGridItem } from '@evolyn.do/ui';
import { computed, markRaw } from 'vue';
import type { DashboardWidget } from '~/types/dashboard';
import DashboardWidgetHost from './DashboardWidgetHost.vue';

const props = defineProps<{
  editable: boolean;
  widgets: DashboardWidget[];
}>();

const emit = defineEmits<{
  'update:editable': [value: boolean];
  'update:widgets': [value: DashboardWidget[]];
  reset: [];
}>();

const components = { DashboardWidgetHost: markRaw(DashboardWidgetHost) };
const gridOptions = computed(() => ({
  column: 12,
  cellHeight: 72,
  margin: 12,
  float: true,
  draggable: { handle: '.dashboard-widget__drag-handle' },
}));

function updateWidgets(value: EvolynGridItem[]) {
  // EvolynGrid 只返回通用布局字段；页面恢复所属卡片的业务元数据。
  const currentById = new Map(props.widgets.map(widget => [widget.id, widget]));
  const widgets = value.flatMap((item) => {
    const current = currentById.get(item.id);
    return current ? [{ ...current, ...item }] : [];
  });
  emit('update:widgets', widgets);
}
</script>

<template>
  <main class="dashboard-canvas">
    <div class="dashboard-canvas__toolbar">
      <span class="dashboard-canvas__hint">{{ editable ? '拖动卡片标题可调整布局' : '可按需自定义你的工作台' }}</span>
      <div class="dashboard-canvas__actions">
        <el-button v-if="editable" :icon="RefreshLeft" @click="emit('reset')">恢复默认</el-button>
        <el-button type="primary" :icon="EditPen" @click="emit('update:editable', !editable)">
          {{ editable ? '完成编辑' : '编辑工作台' }}
        </el-button>
      </div>
    </div>

    <EvolynGrid
      :model-value="widgets"
      :options="gridOptions"
      :components="components"
      :editable="editable"
      @update:model-value="updateWidgets"
    />
  </main>
</template>

<style scoped lang="scss">
.dashboard-canvas {
  max-width: 1600px;
  min-height: calc(100vh - 48px);
  padding: 16px;
  margin: 0 auto;

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: var(--el-component-size-large);
    margin-bottom: 8px;
  }

  &__hint {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__actions {
    display: flex;
    gap: 8px;
  }
}
</style>
