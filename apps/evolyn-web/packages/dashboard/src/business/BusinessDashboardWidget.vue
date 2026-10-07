<script setup lang="ts">
import type { BusinessDashboardWidget } from './types.js';

defineOptions({ name: 'BusinessDashboardWidget' });
defineProps<{ widget: BusinessDashboardWidget }>();
</script>

<template>
  <article class="business-widget">
    <header class="business-widget__header">
      <span class="business-widget__eyebrow">{{
        widget.type === 'chart' ? 'CHART' : 'TABLE'
      }}</span>
      <strong class="business-widget__title">{{ widget.title || '未命名组件' }}</strong>
    </header>
    <div
      class="business-widget__placeholder"
      :class="`business-widget__placeholder--${widget.type}`"
    >
      <template v-if="widget.type === 'chart'">
        <i
          v-for="height in [42, 68, 54, 82, 61, 74]"
          :key="height"
          :style="{ height: `${height}%` }"
        />
      </template>
      <template v-else>
        <i v-for="index in 4" :key="index" />
      </template>
      <span>数据配置将在下一阶段接入</span>
    </div>
  </article>
</template>

<style scoped>
.business-widget {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 18px;
  overflow: hidden;
  color: #172033;
  background: #fff;
  border: 1px solid rgba(23, 32, 51, 0.08);
  border-radius: 14px;
  box-shadow: 0 10px 30px rgba(33, 48, 77, 0.07);
}
.business-widget__header {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.business-widget__eyebrow {
  color: #0f8f84;
  font:
    700 10px/1 ui-monospace,
    monospace;
  letter-spacing: 0.12em;
}
.business-widget__title {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.business-widget__placeholder {
  position: relative;
  display: flex;
  height: calc(100% - 34px);
  margin-top: 16px;
  overflow: hidden;
  color: #8b95a7;
  background: #f5f7fa;
  border-radius: 8px;
}
.business-widget__placeholder > span {
  position: absolute;
  right: 12px;
  bottom: 9px;
  font-size: 11px;
}
.business-widget__placeholder--chart {
  align-items: flex-end;
  gap: 7%;
  padding: 16px 18px 22px;
}
.business-widget__placeholder--chart > i {
  flex: 1;
  min-width: 8px;
  background: linear-gradient(180deg, #44b9ab, #177c74);
  border-radius: 4px 4px 1px 1px;
  opacity: 0.55;
}
.business-widget__placeholder--table {
  flex-direction: column;
  gap: 8px;
  padding: 14px;
}
.business-widget__placeholder--table > i {
  height: 12px;
  background: linear-gradient(
    90deg,
    #dce3ec 18%,
    transparent 18% 22%,
    #e7ebf1 22% 60%,
    transparent 60% 64%,
    #dce3ec 64%
  );
  border-radius: 3px;
}
</style>
