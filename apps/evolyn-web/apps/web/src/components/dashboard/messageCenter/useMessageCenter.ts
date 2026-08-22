import type {
  MessageCategoryId,
  MessageCenterView,
  MessageDeliveryChannel,
  ReminderRecipient,
  ReminderRecipientInput,
} from './messageCenter.types';
import { computed, ref, shallowRef } from 'vue';
import {
  initialMessagePreferences,
  initialMessageRecords,
  messageCategories,
} from './messageCenter.constants';

/**
 * 消息中心的状态边界。
 * 目前以本地样本数据支撑完整交互；接口落地时仅需替换 records/preferences 的初始化和写操作。
 */
export function useMessageCenter() {
  const activeCategoryId = shallowRef<MessageCategoryId>('app-log');
  const activeView = shallowRef<MessageCenterView>('inbox');
  const showUnreadOnly = shallowRef(false);
  const records = ref(initialMessageRecords.map((item) => ({ ...item })));
  const recipients = ref<ReminderRecipient[]>([]);
  const preferences = ref(
    initialMessagePreferences.map((item) => ({
      ...item,
      channels: { ...item.channels },
      recipients: [...item.recipients],
    })),
  );

  const activeCategory = computed(
    () =>
      messageCategories.find((item) => item.id === activeCategoryId.value) ?? messageCategories[0],
  );
  const activeCategoryLabel = computed(() => activeCategory.value.label);
  const activePreferences = computed(() =>
    preferences.value.filter((item) => item.categoryId === activeCategoryId.value),
  );
  const visibleRecords = computed(() =>
    records.value.filter(
      (item) => item.categoryId === activeCategoryId.value && (!showUnreadOnly.value || !item.read),
    ),
  );
  const unreadCount = computed(() => records.value.filter((item) => !item.read).length);
  const unreadCountByCategory = computed(() =>
    records.value.reduce<Partial<Record<MessageCategoryId, number>>>((counts, item) => {
      if (!item.read) counts[item.categoryId] = (counts[item.categoryId] ?? 0) + 1;
      return counts;
    }, {}),
  );

  function selectCategory(categoryId: MessageCategoryId) {
    activeCategoryId.value = categoryId;
    activeView.value = 'inbox';
  }

  /** 设置页切换 Tab 时保留当前设置视图，不返回收件箱。 */
  function selectSettingsCategory(categoryId: MessageCategoryId) {
    activeCategoryId.value = categoryId;
  }

  function openSettings() {
    activeView.value = 'settings';
  }

  /** 提醒对象管理与通知设置共享左侧「通知设置」高亮状态。 */
  function openRecipientManagement() {
    activeView.value = 'recipient-management';
  }

  function markAllAsRead() {
    records.value = records.value.map((item) => ({ ...item, read: true }));
  }

  function markAsRead(messageId: string) {
    records.value = records.value.map((item) =>
      item.id === messageId ? { ...item, read: true } : item,
    );
  }

  function updatePreference(
    preferenceId: string,
    channel: MessageDeliveryChannel,
    checked: boolean,
  ) {
    preferences.value = preferences.value.map((item) =>
      item.id === preferenceId
        ? { ...item, channels: { ...item.channels, [channel]: checked } }
        : item,
    );
  }

  function addRecipient(payload: ReminderRecipientInput) {
    recipients.value = [
      ...recipients.value,
      {
        id: `recipient-${Date.now()}`,
        name: payload.name,
        mobile: payload.mobile,
        email: payload.email,
      },
    ];
  }

  function removeRecipient(recipientId: string) {
    recipients.value = recipients.value.filter((item) => item.id !== recipientId);
  }

  return {
    activeCategoryId,
    activeCategoryLabel,
    activePreferences,
    activeView,
    recipients,
    addRecipient,
    showUnreadOnly,
    unreadCount,
    unreadCountByCategory,
    visibleRecords,
    markAllAsRead,
    markAsRead,
    openSettings,
    openRecipientManagement,
    removeRecipient,
    selectCategory,
    selectSettingsCategory,
    updatePreference,
  };
}
