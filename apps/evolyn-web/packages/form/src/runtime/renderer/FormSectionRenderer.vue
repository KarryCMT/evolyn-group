<script setup lang="ts">
import { computed } from 'vue';
import { buildRenderPlan } from './plan';
import { useFormRendererContext } from '../store/injection';
import FormFieldHost from './FormFieldHost.vue';

/**
 * 区块渲染器：按预编译 render plan 渲染区块序列。当前为平铺单区块；
 * 后续阶段按显隐规则编译多区块并对非首屏区块延迟挂载。
 */
const { runtime } = useFormRendererContext();

const plan = computed(() => (runtime.value ? buildRenderPlan(runtime.value.schema) : null));
</script>

<template>
  <div v-if="plan" class="evf-form__body">
    <section
      v-for="section in plan.sections"
      :key="section.key"
      class="evf-form__section"
      :aria-label="section.key === 'main' ? undefined : section.key"
    >
      <FormFieldHost v-for="item in section.items" :key="item.widget.widgetName" :item="item" />
    </section>
  </div>
</template>
