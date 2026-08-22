<script setup lang="ts">
import type { ReminderRecipient } from './messageCenter.types';
import { RiAddFill, RiArrowLeftSFill } from '@remixicon/vue';

defineOptions({ name: 'MessageRecipientManager' });

defineProps<{
  recipients: ReminderRecipient[];
}>();

const emit = defineEmits<{
  back: [];
  add: [];
  remove: [recipientId: string];
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

      <div v-if="recipients.length" class="message-recipient-manager__body" role="rowgroup">
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
      </div>
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
  padding: 20px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 4px 16px rgb(42 57 77 / 3%);

  &__toolbar {
    display: flex;
    min-height: 44px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 18px;
  }

  &__back,
  &__add,
  &__remove {
    display: inline-flex;
    border: 0;
    align-items: center;
    gap: 6px;
    background: transparent;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__back {
    height: 36px;
    border: 1px solid #d9dfe7;
    padding: 0 12px;
    color: #2d394b;
    border-radius: 7px;
    font-size: 16px;

    &:hover {
      border-color: var(--el-color-primary-light-5);
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__add,
  &__remove {
    color: var(--el-color-primary);
    font-size: 16px;

    &:hover {
      color: var(--el-color-primary-dark-2);
      text-decoration: underline;
    }
  }

  &__add .el-icon {
    font-size: 20px;
  }

  &__table {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    border: 1px solid #edf0f3;
  }

  &__row {
    display: grid;
    min-height: 58px;
    grid-template-columns: 1fr 1.3fr 1.6fr 0.7fr;
    align-items: center;
    padding: 0 20px;
    color: #445063;
    border-top: 1px solid #edf0f3;
    font-size: 15px;
  }

  &__row--header {
    min-height: 68px;
    border-top: 0;
    color: #202938;
    background: #f6f8fa;
    font-weight: 600;
  }

  &__body {
    min-height: 0;
    overflow: auto;
  }

  &__empty {
    display: flex;
    min-height: 0;
    flex: 1;
    align-items: center;
    justify-content: center;
    color: #929aa7;
    font-size: 16px;
  }
}

@media (max-width: 760px) {
  .message-recipient-manager {
    padding: 14px;

    &__row {
      min-width: 620px;
    }

    &__table {
      overflow-x: auto;
    }
  }
}
</style>
