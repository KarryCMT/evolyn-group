<script setup lang="ts">
defineOptions({ name: 'FormCanvasPlaceholder' });
</script>

<template>
  <!-- 子路由统一使用空白占位，后续由 @evolyn.do/form 的设计器视图替换。 -->
  <section class="form-canvas-placeholder" aria-label="待接入的表单设计器" />
</template>

<style scoped lang="scss">
.form-canvas-placeholder {
  width: 100%;
  min-height: 100%;
}
</style>
