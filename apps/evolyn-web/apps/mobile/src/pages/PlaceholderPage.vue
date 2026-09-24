<script setup lang="ts">
import { Empty as VanEmpty } from 'vant';

defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <section class="placeholder-page">
    <VanEmpty image="search" :description="description">
      <h1>{{ title }}</h1>
    </VanEmpty>
  </section>
</template>

<style scoped>
.placeholder-page {
  padding-top: 15vh;
}

.placeholder-page h1 {
  order: -1;
  margin: 0 0 var(--van-padding-sm);
  font-size: var(--van-font-size-lg);
  color: var(--van-text-color);
}
</style>
