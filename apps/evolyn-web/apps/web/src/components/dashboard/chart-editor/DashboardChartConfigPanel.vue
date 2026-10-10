<script setup lang="ts">
import type { BusinessDashboardChartSettings } from '@evolyn.do/dashboard';
import {
  RiAreaChartLine,
  RiBarChart2Line,
  RiBarChartBoxLine,
  RiBarChartGroupedLine,
  RiBarChartHorizontalLine,
  RiBubbleChartFill,
  RiBubbleChartLine,
  RiErrorWarningFill,
  RiFilter3Fill,
  RiLineChartLine,
  RiMapFill,
  RiNumbersFill,
  RiPieChartLine,
  RiRadarLine,
  RiSpeedUpFill,
  RiStockFill,
  RiTableFill,
} from '@remixicon/vue';
import { computed, markRaw, shallowRef } from 'vue';

const props = defineProps<{ settings: BusinessDashboardChartSettings }>();
const emit = defineEmits<{ update: [settings: BusinessDashboardChartSettings] }>();

const activeTab = shallowRef<'function' | 'style'>('function');
const openedSections = shallowRef(['chart', 'bar-type', 'x-axis', 'y-axis']);
const chartTypes = [
  { key: 'number', label: '指标卡', icon: markRaw(RiNumbersFill), disabled: true },
  { key: 'gauge', label: '仪表盘', icon: markRaw(RiSpeedUpFill), disabled: true },
  { key: 'table', label: '透视表', icon: markRaw(RiTableFill), disabled: true },
  {
    key: 'bar-vertical',
    label: '普通柱形图',
    icon: markRaw(RiBarChartBoxLine),
    variant: 'bar',
    orientation: 'vertical',
    stack: 'none',
  },
  {
    key: 'bar-horizontal',
    label: '横向柱形图',
    icon: markRaw(RiBarChartHorizontalLine),
    variant: 'bar',
    orientation: 'horizontal',
    stack: 'none',
  },
  {
    key: 'line',
    label: '折线图',
    icon: markRaw(RiLineChartLine),
    variant: 'line',
    orientation: 'vertical',
    stack: 'none',
  },
  { key: 'area', label: '面积图', icon: markRaw(RiAreaChartLine), disabled: true },
  { key: 'combo', label: '组合图', icon: markRaw(RiStockFill), disabled: true },
  { key: 'scatter', label: '散点图', icon: markRaw(RiBubbleChartLine), disabled: true },
  { key: 'bubble', label: '气泡图', icon: markRaw(RiBubbleChartFill), disabled: true },
  {
    key: 'pie',
    label: '饼图',
    icon: markRaw(RiPieChartLine),
    variant: 'pie',
    orientation: 'vertical',
    stack: 'none',
  },
  { key: 'radar', label: '雷达图', icon: markRaw(RiRadarLine), disabled: true },
  { key: 'funnel', label: '漏斗图', icon: markRaw(RiFilter3Fill), disabled: true },
  { key: 'map', label: '地图', icon: markRaw(RiMapFill), disabled: true },
] as const;

const barTypes = [
  { key: 'standard', label: '普通柱形图', icon: markRaw(RiBarChartBoxLine), stack: 'none' },
  { key: 'stacked', label: '堆叠柱形图', icon: markRaw(RiBarChart2Line), stack: 'normal' },
  { key: 'percent', label: '百分比堆叠柱形图', icon: markRaw(RiBarChartGroupedLine), disabled: true },
] as const;

const activeMode = computed(() => {
  const display = props.settings.display;
  return chartTypes.find(
    (mode) =>
      'variant' in mode &&
      mode.variant === display.variant &&
      (mode.variant === 'pie' || mode.orientation === display.orientation),
  )?.key;
});
const activeBarType = computed(() =>
  props.settings.display.stack === 'normal' ? 'stacked' : 'standard',
);

function selectMode(mode: (typeof chartTypes)[number]) {
  if (!('variant' in mode) || ('disabled' in mode && mode.disabled)) return;
  emit('update', {
    ...props.settings,
    display: {
      ...props.settings.display,
      variant: mode.variant,
      orientation: mode.orientation,
      stack: mode.variant === 'bar' ? props.settings.display.stack : mode.stack,
    },
  });
}

function selectBarType(mode: (typeof barTypes)[number]) {
  if (('disabled' in mode && mode.disabled) || !('stack' in mode)) return;
  patchDisplay({ stack: mode.stack });
}

function patchDisplay(patch: Partial<BusinessDashboardChartSettings['display']>) {
  emit('update', {
    ...props.settings,
    display: { ...props.settings.display, ...patch },
  });
}

function patchLegendVisibility(value: string | number | boolean) {
  patchDisplay({ legend: { ...props.settings.display.legend, visible: Boolean(value) } });
}

function patchLabelVisibility(value: string | number | boolean) {
  patchDisplay({ labels: { visible: Boolean(value) } });
}
</script>

<template>
  <aside class="chart-config-panel" aria-label="图表配置">
    <div class="chart-config-panel__scroll">
      <el-collapse v-model="openedSections">
        <el-collapse-item title="图表类型" name="chart">
          <div class="chart-config-panel__types">
            <el-tooltip
              v-for="mode in chartTypes"
              :key="mode.key"
              :content="'disabled' in mode && mode.disabled ? `${mode.label}暂未开放` : mode.label"
              placement="bottom"
            >
              <button
                type="button"
                :class="{ 'is-active': activeMode === mode.key }"
                :disabled="'disabled' in mode && mode.disabled"
                :aria-label="'disabled' in mode && mode.disabled ? `${mode.label}暂未开放` : mode.label"
                @click="selectMode(mode)"
              >
                <component :is="mode.icon" aria-hidden="true" />
              </button>
            </el-tooltip>
          </div>
        </el-collapse-item>
      </el-collapse>

      <div class="chart-config-panel__tabs" role="tablist" aria-label="图表设置类型">
        <button
          type="button"
          role="tab"
          :class="{ 'is-active': activeTab === 'function' }"
          :aria-selected="activeTab === 'function'"
          @click="activeTab = 'function'"
        >
          功能配置
        </button>
        <button
          type="button"
          role="tab"
          :class="{ 'is-active': activeTab === 'style' }"
          :aria-selected="activeTab === 'style'"
          @click="activeTab = 'style'"
        >
          图表样式
        </button>
      </div>

      <el-collapse v-if="activeTab === 'function'" v-model="openedSections">
        <el-collapse-item
          v-if="settings.display.variant === 'bar'"
          title="柱形图类型"
          name="bar-type"
        >
          <div class="chart-config-panel__bar-types">
            <el-tooltip
              v-for="mode in barTypes"
              :key="mode.key"
              :content="'disabled' in mode && mode.disabled ? `${mode.label}暂未开放` : mode.label"
              placement="bottom"
            >
              <button
                type="button"
                :class="{ 'is-active': activeBarType === mode.key }"
                :disabled="'disabled' in mode && mode.disabled"
                :aria-label="'disabled' in mode && mode.disabled ? `${mode.label}暂未开放` : mode.label"
                @click="selectBarType(mode)"
              >
                <component :is="mode.icon" aria-hidden="true" />
              </button>
            </el-tooltip>
          </div>
        </el-collapse-item>
        <el-collapse-item title="坐标X轴" name="x-axis">
          <el-form label-position="left" label-width="62px">
            <el-form-item label="标签文字">
              <el-select model-value="horizontal">
                <el-option label="横向显示" value="horizontal" />
                <el-option label="倾斜显示" value="diagonal" />
              </el-select>
            </el-form-item>
            <el-form-item class="chart-config-panel__checkbox-row">
              <el-checkbox :model-value="false">
                <span class="chart-config-panel__checkbox-label">
                  强制显示所有标签
                  <RiErrorWarningFill aria-hidden="true" />
                </span>
              </el-checkbox>
            </el-form-item>
            <el-form-item class="chart-config-panel__checkbox-row">
              <el-checkbox :model-value="false">
                显示缩略轴
              </el-checkbox>
            </el-form-item>
          </el-form>
        </el-collapse-item>
        <el-collapse-item title="坐标Y轴" name="y-axis">
          <el-form label-position="left" label-width="62px">
            <el-form-item label="标题">
              <el-input placeholder="请输入标题" />
            </el-form-item>
            <el-form-item label="最大值">
              <el-input placeholder="自动计算" disabled />
            </el-form-item>
            <el-form-item label="最小值">
              <el-input placeholder="自动计算" disabled />
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>

      <div v-else class="chart-config-panel__style-form">
        <label>
          <span>图例位置</span>
          <el-select
            :model-value="settings.display.legend.position"
            @change="patchDisplay({ legend: { ...settings.display.legend, position: $event } })"
          >
            <el-option label="顶部" value="top" />
            <el-option label="右侧" value="right" />
            <el-option label="底部" value="bottom" />
            <el-option label="左侧" value="left" />
          </el-select>
        </label>
        <el-switch
          :model-value="settings.display.legend.visible"
          active-text="显示图例"
          @change="patchLegendVisibility"
        />
        <el-switch
          :model-value="settings.display.labels.visible"
          active-text="显示数据标签"
          @change="patchLabelVisibility"
        />
      </div>
    </div>
  </aside>
</template>

<style scoped>
.chart-config-panel {
  flex: 0 0 300px;
  min-width: 0;
  min-height: 0;
  background: var(--el-bg-color-page);
  border-left: 1px solid var(--el-border-color-lighter);
}

.chart-config-panel__scroll {
  height: 100%;
  padding: 8px 20px 32px;
  overflow: auto;
}

.chart-config-panel :deep(.el-collapse) {
  border: 0;
}

.chart-config-panel :deep(.el-collapse-item__header) {
  height: 42px;
  font-size: 14px;
  font-weight: 600;
  background: transparent;
  border-bottom: 0;
}

.chart-config-panel :deep(.el-collapse-item__wrap) {
  background: transparent;
}

.chart-config-panel :deep(.el-collapse-item) {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.chart-config-panel :deep(.el-collapse-item__content) {
  padding-bottom: 12px;
}

.chart-config-panel__types {
  display: grid;
  grid-template-columns: repeat(5, 40px);
  gap: 12px 15px;
  padding: 2px 0;
}

.chart-config-panel__types button,
.chart-config-panel__bar-types button {
  display: grid;
  width: 40px;
  height: 40px;
  padding: 0;
  color: #11afa6;
  cursor: pointer;
  background: transparent;
  border: 2px solid transparent;
  border-radius: 2px;
  place-items: center;
}

.chart-config-panel__types button:hover,
.chart-config-panel__types button.is-active,
.chart-config-panel__bar-types button:hover,
.chart-config-panel__bar-types button.is-active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
}

.chart-config-panel__types button:disabled,
.chart-config-panel__bar-types button:disabled {
  cursor: not-allowed;
  opacity: 0.24;
}

.chart-config-panel__types svg,
.chart-config-panel__bar-types svg {
  width: 26px;
  height: 26px;
}

.chart-config-panel__bar-types {
  display: grid;
  grid-template-columns: repeat(3, 40px);
  gap: 15px;
  padding: 0 0 12px;
}

.chart-config-panel__tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 2px;
  margin: 10px 0;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}

.chart-config-panel__tabs button {
  height: 28px;
  font-size: 13px;
  font: inherit;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 5px;
}

.chart-config-panel__tabs button.is-active {
  font-weight: 600;
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-lighter);
}

.chart-config-panel :deep(.el-form-item) {
  margin-bottom: 10px;
}

.chart-config-panel :deep(.el-select),
.chart-config-panel :deep(.el-input) {
  width: 100%;
}

.chart-config-panel :deep(.chart-config-panel__checkbox-row .el-form-item__content) {
  margin-left: 0 !important;
}

.chart-config-panel :deep(.el-checkbox) {
  height: 28px;
  font-size: 12px;
}

.chart-config-panel__checkbox-label {
  display: inline-flex;
  gap: 5px;
  align-items: center;
}

.chart-config-panel__checkbox-label svg {
  width: 14px;
  color: #e9a11a;
}

.chart-config-panel__style-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 18px 4px;
}

.chart-config-panel__style-form label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}
</style>
