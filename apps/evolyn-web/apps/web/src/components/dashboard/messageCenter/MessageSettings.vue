<script setup lang="ts">
import type {
  MessageCategoryId,
  MessageDeliveryChannel,
  MessagePreference,
} from './messageCenter.types';
import type {
  NotificationChannelCapability,
  NotificationSettingAggregate,
} from '~/api/notificationSettings';
import { RiRefreshFill } from '@remixicon/vue';
import MessagePreferenceTable from './MessagePreferenceTable.vue';
import MessageSettingsTabs from './MessageSettingsTabs.vue';

defineOptions({ name: 'MessageSettings' });

defineProps<{
  activeCategoryId: MessageCategoryId;
  preferences: MessagePreference[];
  /** 渠道能力：email/sms 能力未就绪时禁用勾选并展示原因（P3 前恒不可用） */
  channelCapabilities: Record<string, NotificationChannelCapability>;
  loading: boolean;
  settingsAggregate: NotificationSettingAggregate | null;
  /** 云币/短信额度：计费事实源未接入时为 null，隐藏数值摘要（不用样例值兜底） */
  smsBudget: NotificationSettingAggregate['smsBudget'];
}>();

const emit = defineEmits<{
  selectCategory: [categoryId: MessageCategoryId];
  updateChannel: [
    payload: { eventCode: string; channel: MessageDeliveryChannel; checked: boolean },
  ];
  /** 每行「修改」：打开该事件的接收对象选择器 */
  editRecipients: [eventCode: string];
  /** 顶部按钮：管理可复用联系人池 */
  manageRecipients: [];
}>();
</script>

<template>
  <section class="message-settings" aria-label="通知设置">
    <div class="message-settings__summary">
      <!-- 云币余额与可发短信数未有计费事实源前不展示数值（smsBudget=null） -->
      <div v-if="smsBudget" class="message-settings__quota">
        <strong>云币余额：{{ smsBudget.coinBalance }}</strong>
        <span>最多可发送短信 {{ smsBudget.remainingCount }} 条</span>
      </div>
      <p v-else class="message-settings__quota-hint">
        邮件与短信提醒通道开放后，可在此查看云币余额与可发短信数量。
      </p>
      <button
        class="message-settings__recipient-button"
        type="button"
        @click="emit('manageRecipients')"
      >
        提醒对象管理
      </button>
    </div>

    <MessageSettingsTabs
      :active-category-id="activeCategoryId"
      :categories="settingsAggregate?.categories ?? []"
      @select="emit('selectCategory', $event)"
    />
    <div v-if="loading && !preferences.length" class="message-settings__loading">
      <el-icon class="is-loading"><RiRefreshFill /></el-icon>
      <span>正在加载通知设置…</span>
    </div>
    <MessagePreferenceTable
      v-else
      :channel-capabilities="channelCapabilities"
      :preferences="preferences"
      @update-channel="emit('updateChannel', $event)"
      @edit-recipients="emit('editRecipients', $event)"
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
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-large);
    box-shadow: var(--el-box-shadow-light);
  }

  &__quota {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--el-space-xl);

    strong {
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-large);
      line-height: 28px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: var(--el-font-size-medium);
      line-height: 24px;
    }
  }

  &__quota-hint {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__loading {
    display: flex;
    min-height: 280px;
    flex: 1;
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }

  &__recipient-button {
    height: 38px;
    flex: 0 0 auto;
    border: 1px solid var(--el-color-primary);
    padding: 0 var(--el-space-lg);
    color: var(--el-color-primary);
    background: var(--el-bg-color);
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
  }
}
</style>
