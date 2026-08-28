<script setup lang="ts">
import type { ReminderRecipient } from './messageCenter.types';
import { RiAddFill, RiArrowLeftSFill } from '@remixicon/vue';

defineOptions({ name: 'MessageRecipientManager' });

defineProps<{
  recipients: ReminderRecipient[];
  loading: boolean;
}>();

const emit = defineEmits<{
  back: [];
  add: [];
  remove: [recipientId: number];
}>();
</script>

<template>
  <section class="message-recipient-manager" aria-label="提醒对象管理">
    <div class="message-recipient-manager__toolbar">
      <button class="message-recipient-manager__back" type="button" @click="emit('back')">
        <el-icon><RiArrowLeftSFill /></el-icon>
        <span>返回</span>
      </button>
      <button class="message-recipient-manager__add" type="button" @click="emit('add')">
        <el-icon><RiAddFill /></el-icon>
        <span>添加提醒对象</span>
      </button>
    </div>

    <div class="message-recipient-manager__table" role="table" aria-label="提醒对象列表">
      <div class="message-recipient-manager__row message-recipient-manager__row--header" role="row">
        <span role="columnheader">姓名</span>
        <span role="columnheader">手机</span>
        <span role="columnheader">邮箱</span>
        <span role="columnheader">操作</span>
      </div>

      <!-- 仅此 body 承担滚动（el-scrollbar），表头固定。 -->
      <el-scrollbar
        v-if="recipients.length"
        class="message-recipient-manager__body"
        role="rowgroup"
      >
        <div
          v-for="recipient in recipients"
          :key="recipient.id"
          class="message-recipient-manager__row"
          role="row"
        >
          <span role="cell">{{ recipient.name }}</span>
          <span role="cell">{{ recipient.mobile || '—' }}</span>
          <span role="cell">{{ recipient.email || '—' }}</span>
          <span role="cell">
            <button
              class="message-recipient-manager__remove"
              type="button"
              @click="emit('remove', recipient.id)"
            >
              删除
            </button>
          </span>
        </div>
      </el-scrollbar>
      <div v-else class="message-recipient-manager__empty">暂无提醒对象</div>
    </div>
  </section>
</template>

<style scoped lang="scss">
.message-recipient-manager {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  padding: var(--el-space-2xl);
  background: var(--el-bg-color);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__toolbar {
    display: flex;
    min-height: 44px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--el-space-xl);
  }

  &__back,
  &__add,
  &__remove {
    display: inline-flex;
    border: 0;
    align-items: center;
    gap: var(--el-space-sm);
    background: transparent;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__back {
    height: 36px;
    border: 1px solid var(--el-border-color);
    padding: 0 var(--el-space-lg);
    color: var(--el-text-color-primary);
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-medium);

    &:hover {
      border-color: var(--el-color-primary-light-5);
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__add,
  &__remove {
    color: var(--el-color-primary);
    font-size: var(--el-font-size-medium);

    &:hover {
      color: var(--el-color-primary-dark-2);
      text-decoration: underline;
    }
  }

  &__add .el-icon {
    font-size: var(--el-font-size-medium);
  }

  &__table {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    border: 1px solid var(--el-border-color-lighter);
  }

  &__row {
    display: grid;
    min-height: 58px;
    grid-template-columns: 1fr 1.3fr 1.6fr 0.7fr;
    align-items: center;
    padding: 0 var(--el-space-2xl);
    color: var(--el-text-color-regular);
    border-top: 1px solid var(--el-border-color-lighter);
    font-size: var(--el-font-size-base);
  }

  &__row--header {
    min-height: 68px;
    border-top: 0;
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    font-weight: 600;
  }

  &__body {
    min-height: 0;
    flex: 1;
  }

  &__empty {
    display: flex;
    min-height: 0;
    flex: 1;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
}

@media (max-width: 760px) {
  .message-recipient-manager {
    padding: var(--el-space-lg);

    &__row {
      min-width: 620px;
    }

    &__table {
      overflow-x: auto;
    }
  }
}
</style>
