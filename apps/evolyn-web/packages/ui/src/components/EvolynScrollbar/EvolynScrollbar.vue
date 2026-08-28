<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'EvolynScrollbar' });

const props = defineProps<{
  /** 滚动容器的固定高度；不传时由父级布局决定高度。 */
  height?: number | string;
}>();

const scrollbarStyle = computed(() => {
  if (props.height === undefined) return undefined;
  return { height: typeof props.height === 'number' ? `${props.height}px` : props.height };
});
</script>

<template>
  <div class="evolyn-scrollbar" :style="scrollbarStyle">
    <slot />
  </div>
</template>

<style lang="scss">
.evolyn-scrollbar {
  overflow: auto;
  scrollbar-color: var(--el-text-color-placeholder) transparent;
  scrollbar-width: thin;

  &::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--el-text-color-placeholder);
    border-radius: var(--el-border-radius-round);
  }

  &::-webkit-scrollbar-thumb:hover {
    background: var(--el-text-color-secondary);
  }
}
</style>
