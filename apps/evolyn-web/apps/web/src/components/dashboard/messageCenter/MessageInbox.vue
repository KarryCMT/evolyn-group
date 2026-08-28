<script setup lang="ts">
import type { MessageRecord } from './messageCenter.types';
import MessageList from './MessageList.vue';
import MessageToolbar from './MessageToolbar.vue';

defineOptions({ name: 'MessageInbox' });

defineProps<{
  messages: MessageRecord[];
  showUnreadOnly: boolean;
  /** 事件筛选：服务端目录选项（空则隐藏下拉） */
  eventCode: string;
  eventOptions: { code: string; label: string }[];
  /** 列表状态：加载/错误/增量分页 */
  loading: boolean;
  error: boolean;
  hasMore: boolean;
  retentionMonths: number;
}>();

const emit = defineEmits<{
  'update:showUnreadOnly': [value: boolean];
  'update:eventCode': [value: string];
  markAllAsRead: [];
  openMessage: [messageId: number];
  loadMore: [];
  retry: [];
}>();
</script>

<template>
  <section class="message-inbox" aria-label="消息列表">
    <MessageToolbar
      :event-code="eventCode"
      :event-options="eventOptions"
      :show-unread-only="showUnreadOnly"
      @update:event-code="emit('update:eventCode', $event)"
      @update:show-unread-only="emit('update:showUnreadOnly', $event)"
      @mark-all-as-read="emit('markAllAsRead')"
    />
    <MessageList
      :error="error"
      :has-more="hasMore"
      :loading="loading"
      :messages="messages"
      :retention-months="retentionMonths"
      @open-message="emit('openMessage', $event)"
      @load-more="emit('loadMore')"
      @retry="emit('retry')"
    />
  </section>
</template>

<style scoped lang="scss">
.message-inbox {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--el-space-2xl);
}
</style>
