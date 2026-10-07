<script setup lang="ts">
import type { BusinessDashboardWidgetDescriptor } from '@evolyn.do/dashboard';
import { RiBarChartGroupedLine, RiTableLine } from '@remixicon/vue';

defineProps<{ descriptors: readonly BusinessDashboardWidgetDescriptor[] }>();
const emit = defineEmits<{ add: [descriptor: BusinessDashboardWidgetDescriptor] }>();
</script>

<template>
  <aside class="component-palette" aria-label="组件库">
    <header class="component-palette__header">
      <span>COMPONENTS</span>
      <strong>组件库</strong>
    </header>
    <div class="component-palette__list">
      <button
        v-for="descriptor in descriptors"
        :key="descriptor.type"
        class="component-palette__item"
        type="button"
        @click="emit('add', descriptor)"
      >
        <span class="component-palette__icon">
          <RiBarChartGroupedLine v-if="descriptor.type === 'chart'" />
          <RiTableLine v-else />
        </span>
        <span class="component-palette__copy">
          <strong>{{ descriptor.label }}</strong>
          <small>{{ descriptor.description }}</small>
        </span>
        <span class="component-palette__add">+</span>
      </button>
    </div>
    <p class="component-palette__footnote">
      组件不会携带演示数据。添加后从右侧完成基础设置。
    </p>
  </aside>
</template>

<style scoped>
.component-palette {
  display: flex;
  flex: 0 0 232px;
  flex-direction: column;
  min-height: 0;
  padding: 22px 16px;
  color: #182238;
  background: #fbfcfd;
  border-right: 1px solid rgba(24, 34, 56, 0.09);
}
.component-palette__header {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 0 5px 18px;
}
.component-palette__header span {
  color: #0f8f84;
  font:
    800 9px/1 ui-monospace,
    monospace;
  letter-spacing: 0.15em;
}
.component-palette__header strong {
  font-size: 16px;
}
.component-palette__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.component-palette__item {
  display: grid;
  grid-template-columns: 40px 1fr 20px;
  gap: 10px;
  align-items: center;
  padding: 13px 11px;
  text-align: left;
  color: inherit;
  cursor: pointer;
  background: #fff;
  border: 1px solid rgba(24, 34, 56, 0.09);
  border-radius: 13px;
  box-shadow: 0 6px 18px rgba(30, 44, 68, 0.04);
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}
.component-palette__item:hover {
  border-color: rgba(15, 143, 132, 0.4);
  box-shadow: 0 10px 24px rgba(23, 99, 94, 0.1);
  transform: translateY(-2px);
}
.component-palette__icon {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  color: #0d766f;
  background: #e8f5f3;
  border-radius: 10px;
  font-size: 21px;
}
.component-palette__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}
.component-palette__copy strong {
  font-size: 13px;
}
.component-palette__copy small {
  color: #8490a3;
  font-size: 10px;
  line-height: 1.35;
}
.component-palette__add {
  color: #0f8f84;
  font:
    500 21px/1 ui-monospace,
    monospace;
}
.component-palette__footnote {
  margin: auto 5px 0;
  color: #98a1af;
  font-size: 11px;
  line-height: 1.6;
}
</style>
