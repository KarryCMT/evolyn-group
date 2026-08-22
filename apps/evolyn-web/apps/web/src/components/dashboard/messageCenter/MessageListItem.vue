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
  border: 1px solid #edf0f3;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 13px;
  padding: 22px 28px;
  color: #747d8d;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 5px 14px rgb(37 50 69 / 3%);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;

  &:hover {
    border-color: var(--el-color-primary-light-7);
    box-shadow: 0 9px 22px rgb(22 119 255 / 10%);
    transform: translateY(-1px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &__time {
    color: #929aa7;
    font-size: 16px;
    line-height: 22px;
  }

  &__content {
    overflow: hidden;
    width: 100%;
    color: #8b94a2;
    font-size: 17px;
    line-height: 26px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &--unread &__time,
  &--unread &__content {
    color: #586273;
    font-weight: 500;
  }

  &__unread-dot {
    position: absolute;
    top: 25px;
    right: 26px;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #f15b5f;
  }
}
</style>
