<script setup lang="ts">
import type { MessageDeliveryChannel, MessagePreference } from './messageCenter.types';
import type { NotificationChannelCapability } from '~/api/notificationSettings';
import { RiInformationFill, RiUserFill } from '@remixicon/vue';

defineOptions({ name: 'MessagePreferenceTable' });

const props = defineProps<{
  preferences: MessagePreference[];
  channelCapabilities: Record<string, NotificationChannelCapability>;
}>();

const emit = defineEmits<{
  updateChannel: [
    payload: { eventCode: string; channel: MessageDeliveryChannel; checked: boolean },
  ];
  editRecipients: [eventCode: string];
}>();

const channelLabels: { key: MessageDeliveryChannel; label: string; tooltip?: string }[] = [
  { key: 'system', label: '系统消息' },
  { key: 'email', label: '邮件' },
  { key: 'sms', label: '短信', tooltip: '短信提醒将消耗云币额度。' },
];

/**
 * 渠道勾选禁用条件：必选渠道（locked）恒禁用；事件不支持的渠道禁用；
 * 渠道能力不可用（P3 前邮件/短信恒不可用）禁用并提示原因——不展示
 * 「已开启但永远无法发送」的虚假状态。
 */
function channelDisabled(preference: MessagePreference, channel: MessageDeliveryChannel): boolean {
  if (preference.lockedChannels.includes(channel)) return true;
  if (!preference.supportedChannels.includes(channel)) return true;
  return !props.channelCapabilities[channel]?.available;
}

function channelTooltip(preference: MessagePreference, channel: MessageDeliveryChannel): string {
  if (!preference.supportedChannels.includes(channel)) return '该事件不支持此渠道';
  const capability = props.channelCapabilities[channel];
  if (capability && !capability.available) return capability.reason;
  return '';
}

/** 避免将 Element Plus 的 label 值写入布尔通知偏好。 */
function updateChannel(
  eventCode: string,
  channel: MessageDeliveryChannel,
  value: boolean | string | number,
) {
  emit('updateChannel', { eventCode, channel, checked: value === true });
}
</script>

<template>
  <div class="message-preference-table">
    <div class="message-preference-table__header">
      <span>消息类型</span>
      <span>提醒方式及对象</span>
    </div>

    <!-- 仅此 body 承担滚动（el-scrollbar），表头始终固定。 -->
    <el-scrollbar class="message-preference-table__body">
      <template v-if="preferences.length">
        <div
          v-for="preference in preferences"
          :key="preference.code"
          class="message-preference-table__row"
        >
          <div class="message-preference-table__type">{{ preference.label }}</div>
          <div class="message-preference-table__delivery">
            <div class="message-preference-table__channels">
              <el-checkbox
                v-for="channel in channelLabels"
                :key="channel.key"
                :disabled="channelDisabled(preference, channel.key)"
                :model-value="preference.channels[channel.key]"
                @update:model-value="updateChannel(preference.code, channel.key, $event)"
              >
                {{ channel.label }}
                <el-tooltip
                  v-if="
                    channelDisabled(preference, channel.key) &&
                    channelTooltip(preference, channel.key)
                  "
                  :content="channelTooltip(preference, channel.key)"
                  placement="top"
                >
                  <el-icon class="message-preference-table__hint"><RiInformationFill /></el-icon>
                </el-tooltip>
                <el-tooltip v-else-if="channel.tooltip" :content="channel.tooltip" placement="top">
                  <el-icon class="message-preference-table__hint"><RiInformationFill /></el-icon>
                </el-tooltip>
              </el-checkbox>
            </div>

            <div class="message-preference-table__recipients">
              <button
                v-for="recipient in preference.recipients"
                :key="recipient.kind + (recipient.recipientId ?? '')"
                class="message-preference-table__recipient"
                type="button"
                @click="emit('editRecipients', preference.code)"
              >
                <el-icon><RiUserFill /></el-icon>
                <span>{{ recipient.label }}</span>
              </button>
              <button
                class="message-preference-table__manage"
                type="button"
                @click="emit('editRecipients', preference.code)"
              >
                修改
              </button>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="message-preference-table__empty">当前分类暂无可配置的通知。</div>
    </el-scrollbar>
  </div>
</template>

<style scoped lang="scss">
.message-preference-table {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__header,
  &__row {
    display: grid;
    min-width: 690px;
    grid-template-columns: minmax(210px, 27%) 1fr;
  }

  &__header {
    flex: 0 0 auto;
    min-height: 68px;
    align-items: center;
    padding: 0 var(--el-space-2xl);
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-base);
    font-weight: 600;
    line-height: 22px;
  }

  &__row {
    min-height: 148px;
    align-items: center;
    padding: 0 var(--el-space-2xl);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  &__body {
    min-height: 0;
    flex: 1;
  }

  &__type {
    padding-right: var(--el-space-2xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__delivery {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-3xl);
  }

  &__channels {
    display: flex;
    min-width: 112px;
    flex-direction: column;
    gap: var(--el-space-lg);

    :deep(.el-checkbox) {
      margin-right: 0;
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
    }

    :deep(.el-checkbox.is-disabled .el-checkbox__label) {
      color: var(--el-text-color-disabled);
    }

    :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary);
    }
  }

  &__hint {
    margin-left: var(--el-space-xs);
    color: var(--el-text-color-placeholder);
    font-size: var(--el-font-size-base);
    vertical-align: -2px;
  }

  &__recipients {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: var(--el-space-md);
  }

  &__recipient,
  &__manage {
    display: inline-flex;
    height: 36px;
    border: 0;
    align-items: center;
    gap: var(--el-space-sm);
    padding: 0 var(--el-space-md);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-medium);
    cursor: pointer;
    font-size: var(--el-font-size-base);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__recipient .el-icon {
    color: var(--el-color-primary);
    font-size: var(--el-font-size-medium);
  }

  &__manage {
    color: var(--el-color-primary);
    background: transparent;
  }

  &__empty {
    display: flex;
    min-height: 280px;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
}
</style>
