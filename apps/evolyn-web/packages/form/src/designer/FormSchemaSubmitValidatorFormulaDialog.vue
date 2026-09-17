<script setup lang="ts">
import { ElDialog } from 'element-plus';
import { shallowRef } from 'vue';

const open = defineModel<boolean>({ required: true });
const expanded = shallowRef(false);

const emit = defineEmits<{
  closed: [];
}>();

function toggleExpanded(): void {
  expanded.value = !expanded.value;
}

function handleClosed(): void {
  expanded.value = false;
  emit('closed');
}
</script>

<template>
  <ElDialog
    v-model="open"
    append-to-body
    destroy-on-close
    lock-scroll
    :show-close="false"
    class="form-submit-formula-dialog"
    :class="{ 'is-expanded': expanded }"
    aria-label="提交校验公式编辑器"
    @closed="handleClosed"
  >
    <template #header
      ><slot name="header" :expanded="expanded" :toggle-expanded="toggleExpanded"
    /></template>
    <slot />
    <template #footer><slot name="footer" /></template>
  </ElDialog>
</template>
