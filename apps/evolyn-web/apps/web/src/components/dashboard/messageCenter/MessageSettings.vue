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
  gap: var(--el-space-3xl);
  overflow: hidden;
  padding: var(--el-space-xs) 0 0;

  &__summary {
    display: flex;
    min-height: 72px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-2xl);
    padding: 0 var(--el-space-2xl);
    background: #fff;
    border-radius: var(--el-border-radius-large);
    box-shadow: var(--el-box-shadow-light);
  }

  &__quota {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--el-space-xl);

    strong {
      color: #202938;
      font-size: var(--el-font-size-large);
      line-height: 28px;
    }

    span {
      color: #828b99;
      font-size: var(--el-font-size-medium);
      line-height: 24px;
    }
  }

  &__recipient-button {
    height: 38px;
    border: 1px solid var(--el-color-primary);
    padding: 0 var(--el-space-lg);
    color: var(--el-color-primary);
    background: #fff;
    border-radius: var(--el-border-radius-medium);
    cursor: pointer;
    font-size: var(--el-font-size-medium);

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 600px) {
  .message-settings {
    gap: var(--el-space-xl);

    &__summary {
      padding: var(--el-space-lg);
      align-items: flex-start;
    }

    &__quota {
      gap: var(--el-space-xs);
      flex-direction: column;
    }

    &__recipient-button {
      flex: 0 0 auto;
    }
  }
}
</style>
