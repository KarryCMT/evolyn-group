<script setup lang="ts">
import type { MessageRecord } from './messageCenter.types';

defineOptions({ name: 'MessageListItem' });

defineProps<{
  message: MessageRecord;
}>();

const emit = defineEmits<{
  open: [messageId: string];
}>();
</script>

<template>
  <button
    class="message-list-item"
    :class="{ 'message-list-item--unread': !message.read }"
    type="button"
    @click="emit('open', message.id)"
  >
    <span v-if="!message.read" class="message-list-item__unread-dot" aria-label="未读" />
    <time class="message-list-item__time">{{ message.createdAt }}</time>
    <span class="message-list-item__content">{{ message.content }}</span>
  </button>
</template>

<style scoped lang="scss">
.message-list-item {
  position: relative;
  display: flex;
  width: 100%;
  min-height: 120px;
  border: 1px solid var(--el-border-color-lighter);
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: var(--el-space-lg);
  padding: var(--el-space-2xl) var(--el-space-3xl);
  color: var(--el-text-color-regular);
  background: var(--el-bg-color);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;

  &:hover {
    border-color: var(--el-color-primary-light-7);
    box-shadow: var(--el-box-shadow-light);
    transform: translateY(-1px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &__time {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 22px;
  }

  &__content {
    overflow: hidden;
    width: 100%;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 26px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &--unread &__time,
  &--unread &__content {
    color: var(--el-text-color-primary);
    font-weight: 500;
  }

  &__unread-dot {
    position: absolute;
    top: 25px;
    right: 26px;
    width: 7px;
    height: 7px;
    border-radius: var(--el-border-radius-half);
    background: var(--el-color-danger);
  }
}
</style>
