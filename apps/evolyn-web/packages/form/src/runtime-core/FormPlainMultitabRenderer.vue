<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef } from 'vue';
import type { FormRenderMultitabNode } from '../runtime/renderer/plan';
import FormFieldHost from '../runtime/renderer/FormFieldHost.vue';
import { useFormRendererContext } from '../runtime/store/injection';

/**
 * Core 的无组件库标签页回退实现：保证嵌入方未提供终端呈现器时，字段顺序、聚焦与
 * 值状态仍正确。Web / 移动端分别提供自身的完整交互与视觉实现。
 */
const props = defineProps<{ node: FormRenderMultitabNode }>();
const activeTab = shallowRef(props.node.tabs[0]?.key ?? '');
const { registerFieldReveal, unregisterFieldReveal } = useFormRendererContext();

onMounted(() => {
  for (const tab of props.node.tabs) {
    for (const field of tab.fields) {
      registerFieldReveal(field.key, () => {
        activeTab.value = tab.key;
      });
    }
  }
});

onBeforeUnmount(() => {
  for (const tab of props.node.tabs) {
    for (const field of tab.fields) unregisterFieldReveal(field.key);
  }
});
</script>

<template>
  <section class="evf-plain-multitab" :aria-label="node.key">
    <div class="evf-plain-multitab__tabs" role="tablist">
      <button
        v-for="tab in node.tabs"
        :key="tab.key"
        class="evf-plain-multitab__tab"
        :class="{ 'is-active': activeTab === tab.key }"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.key"
        @click="activeTab = tab.key"
      >
        {{ tab.title }}
      </button>
    </div>
    <section
      v-for="tab in node.tabs"
      v-show="activeTab === tab.key"
      :key="tab.key"
      class="evf-plain-multitab__pane"
      role="tabpanel"
    >
      <FormFieldHost v-for="field in tab.fields" :key="field.key" :item="field.item" />
    </section>
  </section>
</template>

<style scoped lang="scss">
.evf-plain-multitab {
  width: 100%;
  grid-column: 1 / -1;
  margin-bottom: var(--evf-space-xl);
}

.evf-plain-multitab__tabs {
  display: flex;
  gap: var(--evf-space-xs);
  border-bottom: 1px solid var(--evf-color-border-light);
}

.evf-plain-multitab__tab {
  padding: var(--evf-space-md) var(--evf-space-xl);
  color: var(--evf-color-text-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-bottom: 2px solid transparent;
}

.evf-plain-multitab__tab.is-active {
  color: var(--evf-color-primary);
  border-bottom-color: var(--evf-color-primary);
}

.evf-plain-multitab__pane {
  display: grid;
  grid-template-columns: repeat(var(--evf-columns), minmax(0, 1fr));
  gap: var(--evf-space-xl) var(--evf-space-3xl);
  padding-top: var(--evf-space-xl);
}

@media (width <= 767px) {
  .evf-plain-multitab__pane {
    grid-template-columns: 1fr;
  }
}
</style>
