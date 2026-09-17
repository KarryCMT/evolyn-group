<script setup lang="ts">
import { ElDialog } from 'element-plus';

const open = defineModel<boolean>({ required: true });
</script>

<template>
  <ElDialog
    v-model="open"
    append-to-body
    destroy-on-close
    lock-scroll
    :show-close="false"
    class="form-submit-validator-dialog"
    aria-label="数据校验设置"
  >
    <template #header><slot name="header" /></template>
    <slot />
    <template #footer><slot name="footer" /></template>
  </ElDialog>
</template>
