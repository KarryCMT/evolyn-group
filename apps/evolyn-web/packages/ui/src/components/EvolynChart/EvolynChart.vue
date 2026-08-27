<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, useTemplateRef, watch } from 'vue';
import type { EvolynChartEmits, EvolynChartProps } from './EvolynChart.types';
import { useEvolynChart } from './useEvolynChart';

defineOptions({ name: 'EvolynChart' });

const props = withDefaults(defineProps<EvolynChartProps>(), {
  theme: 'light',
  width: '100%',
  height: '100%',
  autoFit: true,
});
const emit = defineEmits<EvolynChartEmits>();
const container = useTemplateRef<HTMLElement>('container');

const containerStyle = computed(() => ({
  width: typeof props.width === 'number' ? `${props.width}px` : props.width,
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
}));

const { create, getChart, rebuild, release, updateSpec } = useEvolynChart({
  container,
  spec: () => props.spec,
  theme: () => props.theme,
  autoFit: () => props.autoFit,
  options: () => props.options,
  onReady: (chart) => emit('ready', chart),
  onError: (error) => emit('error', error),
});

onMounted(create);
onBeforeUnmount(release);

// Spec 约定为不可变输入：页面替换对象时走 VChart 的增量更新，避免销毁交互状态。
watch(() => props.spec, updateSpec);
// VChart 的主题属于初始化配置，明暗切换时需重建以同步 tooltip、坐标轴等全部图层。
watch(() => props.theme, rebuild);

defineExpose({ getChart });
</script>

<template>
  <div ref="container" class="evolyn-chart" :style="containerStyle" />
</template>

<style lang="scss">
@use './EvolynChart.scss' as *;
</style>
