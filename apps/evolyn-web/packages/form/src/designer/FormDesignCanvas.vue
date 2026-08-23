<script setup lang="ts">
import { ElScrollbar } from 'element-plus';

defineSlots<{
  default?: () => unknown;
}>();
</script>

<template>
  <!-- 字段布局、拖放和实际渲染将在设计器核心接入后由默认插槽承载。 -->
  <main class="form-design-canvas" aria-label="表单设计画布">
    <ElScrollbar class="form-design-canvas__scrollbar">
      <slot />
    </ElScrollbar>
  </main>
</template>

<style scoped lang="scss">
.form-design-canvas {
  display: flex;
  min-width: 0;
  background: var(--el-fill-color-lighter);

  &__scrollbar {
    min-width: 0;
    flex: 1;
  }
}
</style>
