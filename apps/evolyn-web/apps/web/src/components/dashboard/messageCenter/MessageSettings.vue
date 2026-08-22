<script setup lang="ts">
import type {
  MessageCategoryId,
  MessageDeliveryChannel,
  MessagePreference,
} from './messageCenter.types';
import MessagePreferenceTable from './MessagePreferenceTable.vue';
import MessageSettingsTabs from './MessageSettingsTabs.vue';

defineOptions({ name: 'MessageSettings' });

defineProps<{
  activeCategoryId: MessageCategoryId;
  preferences: MessagePreference[];
}>();

const emit = defineEmits<{
  selectCategory: [categoryId: MessageCategoryId];
  updateChannel: [
    payload: { preferenceId: string; channel: MessageDeliveryChannel; checked: boolean },
  ];
  manageRecipients: [preferenceId: string];
}>();
</script>

<template>
  <section class="message-settings" aria-label="通知设置">
    <div class="message-settings__summary">
      <div class="message-settings__quota">
        <strong>云币余额：2</strong>
        <span>最多可发送短信 33 条</span>
      </div>
      <button
        class="message-settings__recipient-button"
        type="button"
        @click="emit('manageRecipients', '')"
      >
        提醒对象管理
      </button>
    </div>

    <MessageSettingsTabs
      :active-category-id="activeCategoryId"
      @select="emit('selectCategory', $event)"
    />
    <MessagePreferenceTable
      :preferences="preferences"
      @update-channel="emit('updateChannel', $event)"
      @manage-recipients="emit('manageRecipients', $event)"
    />
  </section>
</template>

<style scoped lang="scss">
.message-settings {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 24px;
  overflow: hidden;
  padding: 2px 0 0;

  &__summary {
    display: flex;
    min-height: 72px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    padding: 0 20px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 4px 16px rgb(42 57 77 / 3%);
  }

  &__quota {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 18px;

    strong {
      color: #202938;
      font-size: 19px;
      line-height: 28px;
    }

    span {
      color: #828b99;
      font-size: 16px;
      line-height: 24px;
    }
  }

  &__recipient-button {
    height: 38px;
    border: 1px solid #00aaa7;
    padding: 0 14px;
    color: #00aaa7;
    background: #fff;
    border-radius: 8px;
    cursor: pointer;
    font-size: 16px;

    &:hover {
      background: #e7f7f6;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 600px) {
  .message-settings {
    gap: 16px;

    &__summary {
      padding: 14px;
      align-items: flex-start;
    }

    &__quota {
      gap: 2px;
      flex-direction: column;
    }

    &__recipient-button {
      flex: 0 0 auto;
    }
  }
}
</style>
