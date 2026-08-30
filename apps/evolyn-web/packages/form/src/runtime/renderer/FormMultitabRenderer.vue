<script setup lang="ts">
import { ElTabPane, ElTabs } from 'element-plus';
import { onBeforeUnmount, onMounted, shallowRef } from 'vue';
import type { FormRenderMultitabNode } from './plan';
import FormFieldHost from './FormFieldHost.vue';
import { useFormRendererContext } from '../store/injection';

/** 运行时标签页：字段会话由上层 runtime 持有，切页只改变展示，不丢失填写值。 */
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
  <section class="evf-multitab" :aria-label="node.key">
    <el-tabs v-model="activeTab" :type="node.tabStyle === 'style2' ? 'card' : undefined">
      <el-tab-pane v-for="tab in node.tabs" :key="tab.key" :name="tab.key" :label="tab.title">
        <div class="evf-multitab__pane">
          <FormFieldHost v-for="field in tab.fields" :key="field.key" :item="field.item" />
          <p v-if="tab.fields.length === 0" class="evf-multitab__empty">暂无字段</p>
        </div>
      </el-tab-pane>
    </el-tabs>
  </section>
</template>

<style scoped lang="scss">
.evf-multitab {
  width: 100%;
  grid-column: 1 / -1;
  margin-bottom: var(--el-space-xl);
}

.evf-multitab__pane {
  display: grid;
  grid-template-columns: repeat(var(--evf-columns), minmax(0, 1fr));
  gap: var(--evf-space-xl) var(--evf-space-3xl);
  align-items: flex-start;
  min-height: 72px;
  padding-top: var(--el-space-md);
}

@media (width <= 767px) {
  .evf-multitab__pane {
    grid-template-columns: 1fr;
  }
}

.evf-multitab__empty {
  width: 100%;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
