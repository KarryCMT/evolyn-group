<script setup lang="ts">
/**
 * 无障碍关联的错误展示（文档 §7.1/§7.4）：由 FormFieldHost 统一渲染，
 * 字段组件不重复实现错误列表；aria-live 保证校验结果对读屏可感知。
 */
defineProps<{ id: string; messages: readonly string[] }>();
</script>

<template>
  <ul v-if="messages.length > 0" :id="id" class="evf-field__errors" aria-live="polite">
    <li v-for="message in messages" :key="message" class="evf-field__error">
      {{ message }}
    </li>
  </ul>
</template>
