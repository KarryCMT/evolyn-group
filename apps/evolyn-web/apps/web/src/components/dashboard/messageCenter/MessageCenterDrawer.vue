<script setup lang="ts">
import type { ReminderRecipientInput } from './messageCenter.types';
import { RiCloseFill } from '@remixicon/vue';
import { shallowRef, watch } from 'vue';
import AddMessageRecipientDialog from './AddMessageRecipientDialog.vue';
import MessageCenterSidebar from './MessageCenterSidebar.vue';
import MessageInbox from './MessageInbox.vue';
import MessageRecipientManager from './MessageRecipientManager.vue';
import MessageSettings from './MessageSettings.vue';
import { useMessageCenter } from './useMessageCenter';

defineOptions({ name: 'MessageCenterDrawer' });

const emit = defineEmits<{
  unreadChange: [count: number];
}>();

/** 作为顶栏通知入口的唯一对外状态，父组件以 v-model 控制显隐。 */
const visible = defineModel<boolean>({ default: false });

const {
  activeCategoryId,
  activeCategoryLabel,
  activePreferences,
  activeView,
  addRecipient,
  markAllAsRead,
  markAsRead,
  openRecipientManagement,
  openSettings,
  recipients,
  removeRecipient,
  selectCategory,
  selectSettingsCategory,
  showUnreadOnly,
  unreadCount,
  unreadCountByCategory,
  updatePreference,
  visibleRecords,
} = useMessageCenter();
const recipientDialogVisible = shallowRef(false);

// 顶栏红点由未读总数驱动，抽屉内操作后无需依赖父组件额外同步状态。
watch(
  unreadCount,
  (count) => {
    emit('unreadChange', count);
  },
  { immediate: true },
);

function openAddRecipientDialog() {
  recipientDialogVisible.value = true;
}

function handleRecipientSubmit(payload: ReminderRecipientInput) {
  addRecipient(payload);
  recipientDialogVisible.value = false;
}
</script>

<template>
  <el-drawer
    v-model="visible"
    class="message-center-drawer"
    direction="btt"
    size="100%"
    :with-header="false"
    :append-to-body="true"
  >
    <section class="message-center" :aria-label="`消息中心：${activeCategoryLabel}`">
      <header class="message-center__header">
        <h2 class="message-center__title">消息中心</h2>
        <button
          class="message-center__close"
          type="button"
          aria-label="关闭消息中心"
          @click="visible = false"
        >
          <el-icon><RiCloseFill /></el-icon>
        </button>
      </header>

      <div class="message-center__body">
        <MessageCenterSidebar
          :active-category-id="activeCategoryId"
          :active-view="activeView"
          :unread-count-by-category="unreadCountByCategory"
          @select-category="selectCategory"
          @open-settings="openSettings"
        />

        <main class="message-center__content">
          <MessageInbox
            v-if="activeView === 'inbox'"
            :messages="visibleRecords"
            :show-unread-only="showUnreadOnly"
            @update:show-unread-only="showUnreadOnly = $event"
            @mark-all-as-read="markAllAsRead"
            @open-message="markAsRead"
          />
          <MessageSettings
            v-else-if="activeView === 'settings'"
            :active-category-id="activeCategoryId"
            :preferences="activePreferences"
            @select-category="selectSettingsCategory"
            @update-channel="updatePreference($event.preferenceId, $event.channel, $event.checked)"
            @manage-recipients="openRecipientManagement"
          />
          <MessageRecipientManager
            v-else
            :recipients="recipients"
            @back="openSettings"
            @add="openAddRecipientDialog"
            @remove="removeRecipient"
          />
        </main>
      </div>
    </section>
  </el-drawer>
  <AddMessageRecipientDialog v-model="recipientDialogVisible" @submit="handleRecipientSubmit" />
</template>

<style scoped lang="scss">
.message-center {
  display: flex;
  height: 100%;
  flex-direction: column;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);

  &__header {
    display: flex;
    height: 56px;
    min-height: 56px;
    box-sizing: border-box;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--el-space-xl) 0 var(--el-space-2xl);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  &__title {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 26px;
  }

  &__close {
    display: inline-flex;
    width: 32px;
    height: 32px;
    border: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-primary);
    background: transparent;
    border-radius: var(--el-border-radius-medium);
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    .el-icon {
      font-size: var(--el-font-size-extra-large);
    }
  }

  &__body {
    display: flex;
    min-height: 0;
    flex: 1;
  }

  &__content {
    display: flex;
    min-width: 0;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    padding: var(--el-space-4xl) var(--el-space-4xl) var(--el-space-3xl);
  }
}

@media (max-width: 760px) {
  .message-center {
    &__header {
      height: 52px;
      min-height: 52px;
      padding: 0 var(--el-space-lg) 0 var(--el-space-xl);
    }

    &__title {
      font-size: var(--el-font-size-large);
    }

    &__content {
      padding: var(--el-space-2xl) var(--el-space-xl);
    }
  }
}
</style>

<!-- Drawer 内容由 Element Plus 传送至 body，需以稳定类名覆盖默认内边距。 -->
<style lang="scss">
.message-center-drawer .el-drawer__body {
  overflow: hidden;
  padding: 0;
}
</style>
