<script setup lang="ts">
import type { MessageDeliveryChannel, MessagePreference } from './messageCenter.types';
import { InfoFilled, UserFilled } from '@element-plus/icons-vue';

defineOptions({ name: 'MessagePreferenceTable' });

defineProps<{
  preferences: MessagePreference[];
}>();

const emit = defineEmits<{
  updateChannel: [
    payload: { preferenceId: string; channel: MessageDeliveryChannel; checked: boolean },
  ];
  manageRecipients: [preferenceId: string];
}>();

const channelLabels: { key: MessageDeliveryChannel; label: string; tooltip?: string }[] = [
  { key: 'system', label: '系统消息' },
  { key: 'email', label: '邮件' },
  { key: 'sms', label: '短信', tooltip: '短信提醒将消耗云币额度。' },
];

/** 避免将 Element Plus 的 label 值写入布尔通知偏好。 */
function updateChannel(
  preferenceId: string,
  channel: MessageDeliveryChannel,
  value: boolean | string | number,
) {
  emit('updateChannel', { preferenceId, channel, checked: value === true });
}
</script>

<template>
  <div class="message-preference-table">
    <div class="message-preference-table__header">
      <span>消息类型</span>
      <span>提醒方式及对象</span>
    </div>

    <!-- 仅此 body 承担滚动，表头始终固定在红框外。 -->
    <div class="message-preference-table__body">
      <template v-if="preferences.length">
        <div
          v-for="preference in preferences"
          :key="preference.id"
          class="message-preference-table__row"
        >
          <div class="message-preference-table__type">{{ preference.label }}</div>
          <div class="message-preference-table__delivery">
            <div class="message-preference-table__channels">
              <el-checkbox
                v-for="channel in channelLabels"
                :key="channel.key"
                :model-value="preference.channels[channel.key]"
                :disabled="channel.key === 'system'"
                @update:model-value="updateChannel(preference.id, channel.key, $event)"
              >
                {{ channel.label }}
                <el-tooltip v-if="channel.tooltip" :content="channel.tooltip" placement="top">
                  <el-icon class="message-preference-table__hint"><InfoFilled /></el-icon>
                </el-tooltip>
              </el-checkbox>
            </div>

            <div class="message-preference-table__recipients">
              <button
                v-for="recipient in preference.recipients"
                :key="recipient"
                class="message-preference-table__recipient"
                type="button"
                @click="emit('manageRecipients', preference.id)"
              >
                <el-icon><UserFilled /></el-icon>
                <span>{{ recipient }}</span>
              </button>
              <button
                class="message-preference-table__manage"
                type="button"
                @click="emit('manageRecipients', preference.id)"
              >
                修改
              </button>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="message-preference-table__empty">当前分类暂无可配置的通知。</div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.message-preference-table {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #eef1f4;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 4px 16px rgb(42 57 77 / 3%);

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
    padding: 0 20px;
    color: #202938;
    background: #f6f8fa;
    font-size: 15px;
    font-weight: 600;
    line-height: 22px;
  }

  &__row {
    min-height: 148px;
    align-items: center;
    padding: 0 20px;
    border-top: 1px solid #e8edf1;
  }

  &__body {
    min-height: 0;
    flex: 1;
    overflow: auto;
    overscroll-behavior: contain;
  }

  &__type {
    padding-right: 20px;
    color: #293445;
    font-size: 16px;
    line-height: 24px;
  }

  &__delivery {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 28px;
  }

  &__channels {
    display: flex;
    min-width: 112px;
    flex-direction: column;
    gap: 12px;

    :deep(.el-checkbox) {
      margin-right: 0;
      color: #273142;
      font-size: 16px;
    }

    :deep(.el-checkbox.is-disabled .el-checkbox__label) {
      color: #9aa3b0;
    }

    :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
      border-color: #0eb5ae;
      background: #0eb5ae;
    }
  }

  &__hint {
    margin-left: 3px;
    color: #aab2bf;
    font-size: 14px;
    vertical-align: -2px;
  }

  &__recipients {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  &__recipient,
  &__manage {
    display: inline-flex;
    height: 36px;
    border: 0;
    align-items: center;
    gap: 6px;
    padding: 0 10px;
    color: #344052;
    background: #f4f6f8;
    border-radius: 6px;
    cursor: pointer;
    font-size: 15px;

    &:hover {
      color: #008f8d;
      background: #e6f7f5;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__recipient .el-icon {
    color: #10b5ae;
    font-size: 17px;
  }

  &__manage {
    color: #00aaa7;
    background: transparent;
  }

  &__empty {
    display: flex;
    min-height: 280px;
    align-items: center;
    justify-content: center;
    color: #929aa7;
    font-size: 16px;
  }
}
</style>
