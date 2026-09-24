<script setup lang="ts">
import { RiInformationLine } from '@remixicon/vue';

defineOptions({ name: 'IntelligentPageTip' });

defineProps<{
  text: string;
}>();
</script>

<template>
  <p class="intelligent-page-tip">
    <RiInformationLine aria-hidden="true" />
    <span>{{ text }}</span>
  </p>
</template>

<style scoped>
.intelligent-page-tip {
  display: flex;
  margin: 0;
  padding: 10px 12px;
  align-items: flex-start;
  gap: 7px;
  color: #657184;
  background: #f4f7fa;
  border-radius: 6px;
  font-size: 12px;
  line-height: 18px;
}

.intelligent-page-tip svg { width: 16px; height: 16px; margin-top: 1px; flex: 0 0 auto; }
</style>
