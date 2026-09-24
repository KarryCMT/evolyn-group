<template>
  <div>
    <div class="node-add-title">添加通用节点:{{model.id}}</div>
    <ul>
      <li @click="clickMe">11</li>
      <li>14</li>
      <li>13</li>
      <li>11</li>
      <li>14</li>
      <li>13</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'IntelligentPopoverContent' });

defineProps<{
  model: { id: string };
}>();

// 预留节点菜单动作，避免模板直接依赖画布实例。
function clickMe(): void {}
</script>

<style>

</style>
