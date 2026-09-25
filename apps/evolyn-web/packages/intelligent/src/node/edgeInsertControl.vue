<script setup lang="ts">
import { RiAddLine } from '@remixicon/vue';

defineOptions({ name: 'IntelligentEdgeInsertControl' });
</script>

<template>
  <button
    type="button"
    class="intelligent-edge-insert-control"
    aria-label="在此处添加下级节点"
    title="添加下级节点"
  >
    <RiAddLine aria-hidden="true" />
  </button>
</template>

<style scoped lang="scss">
.intelligent-edge-insert-control {
  display: inline-flex;
  width: 28px;
  height: 28px;
  padding: 0;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: #2f7cf6;
  border: 2px solid #fff;
  border-radius: 50%;
  box-shadow: 0 2px 8px rgb(47 124 246 / 28%);
  box-sizing: border-box;
  cursor: pointer;
  transition: background-color 140ms ease, box-shadow 140ms ease, transform 140ms ease;

  &:hover,
  &:focus-visible {
    background: #1f6eea;
    box-shadow: 0 0 0 4px rgb(47 124 246 / 14%), 0 3px 10px rgb(47 124 246 / 30%);
    outline: none;
    transform: scale(1.06);
  }

  svg {
    width: 18px;
    height: 18px;
  }
}
</style>
