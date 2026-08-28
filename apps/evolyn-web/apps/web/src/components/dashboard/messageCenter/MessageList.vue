<script setup lang="ts">
import type { MessageRecord } from './messageCenter.types';
import type { ScrollbarInstance } from 'element-plus';
import { RiRefreshFill } from '@remixicon/vue';
import { ref } from 'vue';
import MessageEmptyState from './MessageEmptyState.vue';
import MessageListItem from './MessageListItem.vue';

defineOptions({ name: 'MessageList' });

const props = defineProps<{
  messages: MessageRecord[];
  loading: boolean;
  error: boolean;
  hasMore: boolean;
  retentionMonths: number;
}>();

const emit = defineEmits<{
  openMessage: [messageId: number];
  loadMore: [];
  retry: [];
}>();

const scrollbarRef = ref<ScrollbarInstance>();

/** 列表滚动触底：仅内容区滚动的 el-scrollbar 内做增量分页。 */
function handleScroll({ scrollTop }: { scrollTop: number }) {
  if (!props.hasMore || props.loading) return;
  const wrap = scrollbarRef.value?.wrapRef;
  if (!wrap) return;
  if (scrollTop + wrap.clientHeight >= wrap.scrollHeight - 24) emit('loadMore');
}

/** 加载失败重试：触发上层重新加载首页。 */
function retry() {
  emit('retry');
}
</script>

<template>
  <!-- 仅此内容区承担滚动（页面标题/工具栏固定），增量分页在触底时触发 -->
  <el-scrollbar ref="scrollbarRef" class="message-list" @scroll="handleScroll">
    <div class="message-list__inner">
      <template v-if="messages.length">
        <MessageListItem
          v-for="message in messages"
          :key="message.id"
          :message="message"
          @open="emit('openMessage', $event)"
        />
        <div v-if="loading" class="message-list__state">加载中…</div>
        <div v-else-if="hasMore" class="message-list__state">上滑加载更多</div>
        <div class="message-list__retention">保存最近 {{ retentionMonths }} 个月的消息记录</div>
      </template>
      <template v-else-if="loading">
        <div class="message-list__state">加载中…</div>
      </template>
      <button v-else-if="error" class="message-list__retry" type="button" @click="retry">
        <el-icon><RiRefreshFill /></el-icon>
        <span>加载失败，点击重试</span>
      </button>
      <MessageEmptyState v-else />
    </div>
  </el-scrollbar>
</template>

<style scoped lang="scss">
.message-list {
  min-height: 0;
  flex: 1;

  &__inner {
    display: flex;
    min-height: 100%;
    flex-direction: column;
    gap: var(--el-space-xl);
    padding-bottom: var(--el-space-md);
  }

  &__state,
  &__retry {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    min-height: 56px;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  &__retry {
    border: 0;
    background: transparent;
    cursor: pointer;

    &:hover {
      color: var(--el-color-primary);
    }
  }

  &__retention {
    display: flex;
    align-items: center;
    gap: var(--el-space-3xl);
    margin: var(--el-space-3xl) 0 var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
    white-space: nowrap;

    &::before,
    &::after {
      display: block;
      width: 100%;
      height: 1px;
      background: var(--el-border-color-light);
      content: '';
    }
  }
}
</style>
