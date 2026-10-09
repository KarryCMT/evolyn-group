<script setup lang="ts">
import type { BusinessDashboardDocument } from '@evolyn.do/dashboard';

defineProps<{
  desktop: BusinessDashboardDocument['settings']['desktop'];
}>();

const emit = defineEmits<{
  update: [patch: Partial<BusinessDashboardDocument['settings']['desktop']>];
}>();

const densityOptions = [
  { label: '紧凑', value: 64 },
  { label: '标准', value: 80 },
  { label: '宽松', value: 96 },
];
</script>

<template>
  <aside class="style-panel" aria-label="仪表盘样式">
    <header class="style-panel__header">
      <span>DASHBOARD STYLE</span>
      <strong>仪表盘样式</strong>
    </header>

    <div class="style-panel__body">
      <section class="style-panel__section">
        <div class="style-panel__section-title">
          <strong>画布布局</strong>
          <span>控制组件在设计画布中的垂直密度。</span>
        </div>

        <div class="style-panel__field">
          <label>栅格列数</label>
          <div class="style-panel__readonly">
            <strong>{{ desktop.columns }}</strong>
            <span>固定列</span>
          </div>
        </div>

        <div class="style-panel__field">
          <label>行高</label>
          <div class="style-panel__value">
            <strong>{{ desktop.rowHeight }}</strong>
            <span>px</span>
          </div>
          <el-slider
            :model-value="desktop.rowHeight"
            :min="48"
            :max="112"
            :step="4"
            :show-tooltip="false"
            @input="emit('update', { rowHeight: Number($event) })"
          />
        </div>

        <div class="style-panel__field">
          <label>密度预设</label>
          <el-segmented
            :model-value="desktop.rowHeight"
            :options="densityOptions"
            @change="emit('update', { rowHeight: Number($event) })"
          />
        </div>
      </section>

      <section class="style-panel__section style-panel__section--note">
        <strong>布局说明</strong>
        <p>组件宽高仍由画布中的拖拽与缩放决定；行高只改变每一栅格行的视觉高度。</p>
      </section>
    </div>
  </aside>
</template>

<style scoped>
/* Vue 的 :deep() 仅用于让 Element Plus 控件填满面板内容区。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.style-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
}

.style-panel__header {
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 20px 48px 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.style-panel__header span {
  font:
    800 9px/1 ui-monospace,
    monospace;
  color: var(--el-color-primary);
  letter-spacing: 0.14em;
}

.style-panel__header strong {
  font-size: 16px;
}

.style-panel__body {
  flex: 1;
  padding: 18px 20px;
  overflow: auto;
}

.style-panel__section {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 18px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}

.style-panel__section + .style-panel__section {
  margin-top: 12px;
}

.style-panel__section-title {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.style-panel__section-title strong,
.style-panel__section--note strong {
  font-size: 14px;
}

.style-panel__section-title span,
.style-panel__section--note p {
  margin: 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
}

.style-panel__field {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.style-panel__field label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.style-panel__readonly {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  height: 38px;
  padding: 0 12px;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.style-panel__readonly strong {
  line-height: 38px;
}

.style-panel__readonly span,
.style-panel__value span {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.style-panel__value {
  position: absolute;
  top: 0;
  right: 0;
  display: flex;
  gap: 4px;
  align-items: baseline;
}

.style-panel__value strong {
  font-size: 13px;
}

.style-panel__field :deep(.el-segmented) {
  width: 100%;
}

.style-panel__field :deep(.el-segmented__group) {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  width: 100%;
}

.style-panel__section--note {
  gap: 7px;
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-7);
}
</style>
