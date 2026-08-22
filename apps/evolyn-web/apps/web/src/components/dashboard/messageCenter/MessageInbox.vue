<script setup lang="ts">
import type { MessageRecord } from './messageCenter.types';
import MessageList from './MessageList.vue';
import MessageToolbar from './MessageToolbar.vue';

defineOptions({ name: 'MessageInbox' });

defineProps<{
  messages: MessageRecord[];
  showUnreadOnly: boolean;
}>();

const emit = defineEmits<{
  'update:showUnreadOnly': [value: boolean];
  markAllAsRead: [];
  openMessage: [messageId: string];
}>();
</script>

<template>
  <section class="message-inbox" aria-label="消息列表">
    <MessageToolbar
      :show-unread-only="showUnreadOnly"
      @update:show-unread-only="emit('update:showUnreadOnly', $event)"
      @mark-all-as-read="emit('markAllAsRead')"
    />
    <MessageList :messages="messages" @open-message="emit('openMessage', $event)" />
  </section>
</template>

<style scoped lang="scss">
.message-inbox {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 22px;
}
</style>
