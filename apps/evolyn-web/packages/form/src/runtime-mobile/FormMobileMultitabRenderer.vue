<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef } from 'vue';
import type { FormRenderMultitabNode } from '../runtime/renderer/plan';
import FormFieldHost from '../runtime/renderer/FormFieldHost.vue';
import { useFormRendererContext } from '../runtime/store/injection';

/**
 * 移动端标签页：使用原生按钮和单列内容区，保留所有页签的字段会话，切换页面不丢失值。
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
  <section class="evf-mobile-multitab" :aria-label="node.key">
    <div class="evf-mobile-multitab__tabs" role="tablist">
      <button
        v-for="tab in node.tabs"
        :key="tab.key"
        class="evf-mobile-multitab__tab"
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
      class="evf-mobile-multitab__pane"
      role="tabpanel"
    >
      <FormFieldHost v-for="field in tab.fields" :key="field.key" :item="field.item" />
      <p v-if="tab.fields.length === 0" class="evf-mobile-multitab__empty">暂无字段</p>
    </section>
  </section>
</template>

<style scoped lang="scss">
.evf-mobile-multitab {
  grid-column: 1 / -1;
  width: 100%;
}

.evf-mobile-multitab__tabs {
  display: flex;
  gap: var(--evf-space-xs);
  overflow-x: auto;
  padding-bottom: var(--evf-space-sm);
  border-bottom: 1px solid var(--evf-color-border-light);
  scrollbar-width: none;
}

.evf-mobile-multitab__tabs::-webkit-scrollbar {
  display: none;
}

.evf-mobile-multitab__tab {
  flex: 0 0 auto;
  min-height: 40px;
  padding: 0 var(--evf-space-xl);
  color: var(--evf-color-text-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--evf-radius-base);
}

.evf-mobile-multitab__tab.is-active {
  color: var(--evf-color-primary);
  background: color-mix(in srgb, var(--evf-color-primary) 10%, transparent);
}

.evf-mobile-multitab__pane {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--evf-space-xl);
  padding-top: var(--evf-space-xl);
}

.evf-mobile-multitab__empty {
  margin: 0;
  color: var(--evf-color-text-secondary);
  text-align: center;
}
</style>
