<script setup lang="ts">
import type { MessageRecord } from './messageCenter.types';
import MessageEmptyState from './MessageEmptyState.vue';
import MessageListItem from './MessageListItem.vue';

defineOptions({ name: 'MessageList' });

defineProps<{
  messages: MessageRecord[];
}>();

const emit = defineEmits<{
  openMessage: [messageId: string];
}>();
</script>

<template>
  <div class="message-list">
    <template v-if="messages.length">
      <MessageListItem
        v-for="message in messages"
        :key="message.id"
        :message="message"
        @open="emit('openMessage', $event)"
      />
      <div class="message-list__retention">保存最近六个月的消息记录</div>
    </template>
    <MessageEmptyState v-else />
  </div>
</template>

<style scoped lang="scss">
.message-list {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 16px;

  &__retention {
    display: flex;
    align-items: center;
    gap: 28px;
    margin: 28px 0 10px;
    color: #8d95a2;
    font-size: 15px;
    line-height: 22px;
    white-space: nowrap;

    &::before,
    &::after {
      display: block;
      width: 100%;
      height: 1px;
      background: #e2e6eb;
      content: '';
    }
  }
}
</style>
